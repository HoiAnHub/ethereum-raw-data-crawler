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

		// Block scheduler service
		fx.Provide(
			fx.Annotate(
				blockchain.NewWebSocketScheduler,
				fx.As(new(service.BlockSchedulerService)),
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
		fx.Provide(appservice.NewSchedulerService),

		// Lifecycle hooks
		fx.Invoke(registerSchedulerHooks),
	)

	app.Run()
}

// registerSchedulerHooks registers scheduler application lifecycle hooks
func registerSchedulerHooks(
	lc fx.Lifecycle,
	cfg *config.Config,
	logger *logger.Logger,
	db *database.MongoDB,
	crawlerService *appservice.CrawlerService,
	schedulerService *appservice.SchedulerService,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting Ethereum Block Scheduler",
				zap.String("version", "1.0.0"),
				zap.String("network", cfg.Ethereum.Network),
				zap.String("scheduler_mode", cfg.Scheduler.Mode))

			// Create database indexes
			if err := db.CreateIndexes(ctx); err != nil {
				logger.Error("Failed to create database indexes", zap.Error(err))
				return err
			}

			// Configure crawler for external scheduler mode
			crawlerService.SetExternalSchedulerMode(true)

			// Start crawler service (without internal worker)
			if err := crawlerService.Start(ctx); err != nil {
				logger.Error("Failed to start crawler service", zap.Error(err))
				return err
			}

			// Start scheduler service (this will handle block scheduling)
			if err := schedulerService.Start(ctx); err != nil {
				logger.Error("Failed to start scheduler service", zap.Error(err))
				return err
			}

			// Setup graceful shutdown
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				<-sigChan

				logger.Info("Received shutdown signal")
				
				// Stop scheduler service first
				if err := schedulerService.Stop(); err != nil {
					logger.Error("Error stopping scheduler service", zap.Error(err))
				}
				
				// Then stop crawler service
				if err := crawlerService.Stop(ctx); err != nil {
					logger.Error("Error stopping crawler service", zap.Error(err))
				}
			}()

			logger.Info("Ethereum Block Scheduler started successfully",
				zap.String("mode", cfg.Scheduler.Mode),
				zap.Bool("realtime_enabled", cfg.Scheduler.EnableRealtime),
				zap.Bool("polling_enabled", cfg.Scheduler.EnablePolling))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping Ethereum Block Scheduler")

			// Stop scheduler service first
			if err := schedulerService.Stop(); err != nil {
				logger.Error("Error stopping scheduler service", zap.Error(err))
			}

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

			logger.Info("Ethereum Block Scheduler stopped")
			return nil
		},
	})
}
