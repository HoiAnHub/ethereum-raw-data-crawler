package main

import (
	"context"
	"ethereum-raw-data-crawler/internal/adapters/secondary"
	appservice "ethereum-raw-data-crawler/internal/application/service"
	"ethereum-raw-data-crawler/internal/domain/repository"
	"ethereum-raw-data-crawler/internal/domain/service"
	"ethereum-raw-data-crawler/internal/infrastructure/blockchain"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/database"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// provideMongoDBConfig extracts MongoDB configuration from main config
func provideMongoDBConfig(cfg *config.Config) *config.MongoDBConfig {
	return &cfg.MongoDB
}

// provideEthereumConfig extracts Ethereum configuration from main config
func provideEthereumConfig(cfg *config.Config) *config.EthereumConfig {
	return &cfg.Ethereum
}

func main() {
	app := fx.New(
		// Configuration
		fx.Provide(config.LoadConfig),
		fx.Provide(provideMongoDBConfig),
		fx.Provide(provideEthereumConfig),

		// Infrastructure
		fx.Provide(logger.NewLogger),
		fx.Provide(database.NewMongoDB),

		// Blockchain service
		fx.Provide(
			fx.Annotate(
				blockchain.NewEthereumService,
				fx.As(new(service.BlockchainService)),
			),
		),

		// Repositories
		fx.Provide(
			fx.Annotate(
				secondary.NewBlockRepository,
				fx.As(new(repository.BlockRepository)),
			),
		),
		fx.Provide(
			fx.Annotate(
				secondary.NewTransactionRepository,
				fx.As(new(repository.TransactionRepository)),
			),
		),
		fx.Provide(
			fx.Annotate(
				secondary.NewMetricsRepository,
				fx.As(new(repository.MetricsRepository)),
			),
		),

		// Application services
		fx.Provide(appservice.NewCrawlerService),

		// Lifecycle hooks
		fx.Invoke(registerHooks),
	)

	app.Run()
}

// registerHooks registers application lifecycle hooks
func registerHooks(
	lc fx.Lifecycle,
	cfg *config.Config,
	logger *logger.Logger,
	db *database.MongoDB,
	crawlerService *appservice.CrawlerService,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting Ethereum Raw Data Crawler",
				zap.String("version", "1.0.0"),
				zap.String("network", cfg.Ethereum.Network))

			// Create database indexes
			if err := db.CreateIndexes(ctx); err != nil {
				logger.Error("Failed to create database indexes", zap.Error(err))
				return err
			}

			// Start crawler service
			if err := crawlerService.Start(ctx); err != nil {
				logger.Error("Failed to start crawler service", zap.Error(err))
				return err
			}

			// Setup graceful shutdown
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				<-sigChan

				logger.Info("Received shutdown signal")
				if err := crawlerService.Stop(ctx); err != nil {
					logger.Error("Error stopping crawler service", zap.Error(err))
				}
			}()

			logger.Info("Ethereum Raw Data Crawler started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping Ethereum Raw Data Crawler")

			// Stop crawler service
			if err := crawlerService.Stop(ctx); err != nil {
				logger.Error("Error stopping crawler service", zap.Error(err))
			}

			// Close database connection
			if err := db.Close(ctx); err != nil {
				logger.Error("Error closing database connection", zap.Error(err))
			}

			// Sync logger
			if err := logger.Sync(); err != nil {
				// Ignore errors on sync as this is expected on some systems
			}

			logger.Info("Ethereum Raw Data Crawler stopped")
			return nil
		},
	})
}
