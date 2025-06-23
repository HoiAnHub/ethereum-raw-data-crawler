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
	"ethereum-raw-data-crawler/internal/infrastructure/messaging"
	"os"
	"os/signal"
	"syscall"
	"time"

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

// provideWebSocketConfig extracts WebSocket configuration from main config
func provideWebSocketConfig(cfg *config.Config) *config.WebSocketConfig {
	return &cfg.WebSocket
}

func main() {
	app := fx.New(
		// Increase startup timeout
		fx.StartTimeout(5*time.Minute),
		fx.StopTimeout(30*time.Second),

		// Configuration
		fx.Provide(config.LoadConfig),
		fx.Provide(provideMongoDBConfig),
		fx.Provide(provideEthereumConfig),
		fx.Provide(provideWebSocketConfig),

		// Infrastructure
		fx.Provide(logger.NewLogger),
		fx.Provide(database.NewMongoDB),

		// Messaging service (optional, for notifications)
		fx.Provide(
			fx.Annotate(
				messaging.NewNATSMessagingService,
				fx.As(new(service.MessagingService)),
			),
		),

		// Blockchain service
		fx.Provide(
			fx.Annotate(
				blockchain.NewEthereumService,
				fx.As(new(service.BlockchainService)),
			),
		),

		// WebSocket listener service
		fx.Provide(
			fx.Annotate(
				blockchain.NewWebSocketListener,
				fx.As(new(service.WebSocketListenerService)),
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
		fx.Provide(appservice.NewWebSocketListenerAppService),

		// Lifecycle hooks
		fx.Invoke(registerWebSocketListenerHooks),
	)

	app.Run()
}

// registerWebSocketListenerHooks registers websocket listener application lifecycle hooks
func registerWebSocketListenerHooks(
	lc fx.Lifecycle,
	cfg *config.Config,
	logger *logger.Logger,
	db *database.MongoDB,
	blockchainService service.BlockchainService,
	messagingService service.MessagingService,
	webSocketListenerService *appservice.WebSocketListenerAppService,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting Ethereum WebSocket Listener",
				zap.String("version", "1.0.0"),
				zap.String("network", cfg.Ethereum.Network),
				zap.String("ws_url", cfg.Ethereum.WSURL))

			// Create database indexes
			if err := db.CreateIndexes(ctx); err != nil {
				logger.Error("Failed to create database indexes", zap.Error(err))
				return err
			}

			// Connect to blockchain service (required for fetching block details)
			if err := blockchainService.Connect(ctx); err != nil {
				logger.Error("Failed to connect to blockchain service", zap.Error(err))
				return err
			}
			logger.Info("Successfully connected to blockchain service")

			// Connect to messaging service (optional)
			if cfg.NATS.Enabled {
				if err := messagingService.Connect(ctx); err != nil {
					logger.Error("Failed to connect to messaging service", zap.Error(err))
					// Don't fail startup if NATS is unavailable
				}
			}

			// Start websocket listener service
			if err := webSocketListenerService.Start(ctx); err != nil {
				logger.Error("Failed to start websocket listener service", zap.Error(err))
				return err
			}

			// Setup graceful shutdown
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				<-sigChan

				logger.Info("Received shutdown signal")

				// Stop websocket listener service
				if err := webSocketListenerService.Stop(ctx); err != nil {
					logger.Error("Error stopping websocket listener service", zap.Error(err))
				}
			}()

			logger.Info("Ethereum WebSocket Listener started successfully",
				zap.String("ws_url", cfg.Ethereum.WSURL),
				zap.Bool("realtime_enabled", true))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping Ethereum WebSocket Listener")

			// Stop websocket listener service
			if err := webSocketListenerService.Stop(ctx); err != nil {
				logger.Error("Error stopping websocket listener service", zap.Error(err))
			}

			// Disconnect from blockchain service
			if err := blockchainService.Disconnect(); err != nil {
				logger.Error("Error disconnecting from blockchain service", zap.Error(err))
			}

			// Disconnect from messaging service
			if cfg.NATS.Enabled {
				if err := messagingService.Disconnect(); err != nil {
					logger.Error("Error disconnecting from messaging service", zap.Error(err))
				}
			}

			// Close database connection
			if err := db.Close(ctx); err != nil {
				logger.Error("Error closing database connection", zap.Error(err))
			}

			// Sync logger
			if err := logger.Sync(); err != nil {
				// Ignore errors on sync as this is expected on some systems
			}

			logger.Info("Ethereum WebSocket Listener stopped")
			return nil
		},
	})
}
