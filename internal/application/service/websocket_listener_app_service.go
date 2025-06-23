package service

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/domain/repository"
	"ethereum-raw-data-crawler/internal/domain/service"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"fmt"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WebSocketListenerAppService handles real-time data streaming from WebSocket
type WebSocketListenerAppService struct {
	webSocketService  service.WebSocketListenerService
	blockchainService service.BlockchainService
	blockRepo         repository.BlockRepository
	transactionRepo   repository.TransactionRepository
	metricsRepo       repository.MetricsRepository
	messagingService  service.MessagingService
	config            *config.Config
	logger            *logger.Logger

	isRunning bool
	stopChan  chan struct{}
	mu        sync.RWMutex

	// Data buffering
	blockBuffer       []*entity.Block
	transactionBuffer []*entity.Transaction
	bufferMu          sync.Mutex
	lastFlush         time.Time
}

// NewWebSocketListenerAppService creates a new websocket listener application service
func NewWebSocketListenerAppService(
	webSocketService service.WebSocketListenerService,
	blockchainService service.BlockchainService,
	blockRepo repository.BlockRepository,
	transactionRepo repository.TransactionRepository,
	metricsRepo repository.MetricsRepository,
	messagingService service.MessagingService,
	config *config.Config,
	logger *logger.Logger,
) *WebSocketListenerAppService {
	return &WebSocketListenerAppService{
		webSocketService:  webSocketService,
		blockchainService: blockchainService,
		blockRepo:         blockRepo,
		transactionRepo:   transactionRepo,
		metricsRepo:       metricsRepo,
		messagingService:  messagingService,
		config:            config,
		logger:            logger.WithComponent("websocket-listener-app"),
		stopChan:          make(chan struct{}),
		blockBuffer:       make([]*entity.Block, 0, config.WebSocket.BatchSize),
		transactionBuffer: make([]*entity.Transaction, 0, config.WebSocket.BatchSize*100), // Assume avg 100 txs per block
		lastFlush:         time.Now(),
	}
}

// Start starts the websocket listener application service
func (w *WebSocketListenerAppService) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return fmt.Errorf("websocket listener service is already running")
	}

	w.logger.Info("Starting WebSocket Listener Application Service")

	// Start websocket service
	if err := w.webSocketService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start websocket service: %w", err)
	}

	// Subscribe to different data streams based on configuration
	if w.config.WebSocket.SubscribeToBlocks {
		if err := w.webSocketService.SubscribeNewBlocks(ctx, w.handleNewBlock); err != nil {
			return fmt.Errorf("failed to subscribe to new blocks: %w", err)
		}
		w.logger.Info("Subscribed to new blocks")
	}

	if w.config.WebSocket.SubscribeToTxs {
		if err := w.webSocketService.SubscribePendingTransactions(ctx, w.handlePendingTransaction); err != nil {
			return fmt.Errorf("failed to subscribe to pending transactions: %w", err)
		}
		w.logger.Info("Subscribed to pending transactions")
	}

	if w.config.WebSocket.SubscribeToLogs {
		if err := w.webSocketService.SubscribeLogs(ctx, w.handleLog); err != nil {
			return fmt.Errorf("failed to subscribe to logs: %w", err)
		}
		w.logger.Info("Subscribed to contract logs")
	}

	w.isRunning = true

	// Start periodic flush worker
	go w.flushWorker(ctx)

	// Start health monitor
	go w.healthMonitor(ctx)

	w.logger.Info("WebSocket Listener Application Service started successfully")
	return nil
}

// Stop stops the websocket listener application service
func (w *WebSocketListenerAppService) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return nil
	}

	w.logger.Info("Stopping WebSocket Listener Application Service")

	close(w.stopChan)
	w.isRunning = false

	// Stop websocket service
	if err := w.webSocketService.Stop(); err != nil {
		w.logger.Error("Failed to stop websocket service", zap.Error(err))
	}

	// Flush remaining data
	w.flushBuffers(ctx)

	w.logger.Info("WebSocket Listener Application Service stopped")
	return nil
}

// IsRunning checks if the service is running
func (w *WebSocketListenerAppService) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isRunning
}

// handleNewBlock handles new block events
func (w *WebSocketListenerAppService) handleNewBlock(blockNumber *big.Int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	w.logger.Debug("Received new block", zap.String("block_number", blockNumber.String()))

	// Fetch block details
	block, err := w.blockchainService.GetBlockByNumber(ctx, blockNumber)
	if err != nil {
		w.logger.Error("Failed to get block details",
			zap.String("block_number", blockNumber.String()),
			zap.Error(err))
		return
	}

	// Add to buffer
	w.bufferMu.Lock()
	w.blockBuffer = append(w.blockBuffer, block)

	// Get transactions for this block separately
	transactions, err := w.blockchainService.GetTransactionsByBlock(ctx, blockNumber)
	if err != nil {
		w.logger.Warn("Failed to get transactions for block",
			zap.String("block_number", blockNumber.String()),
			zap.Error(err))
	} else {
		// Add transactions to buffer
		for _, tx := range transactions {
			w.transactionBuffer = append(w.transactionBuffer, tx)
		}
	}
	w.bufferMu.Unlock()

	// Check if we need to flush
	if w.shouldFlush() {
		go w.flushBuffers(ctx)
	}

	// Send notification if messaging is enabled
	w.notifyNewBlock(block)
}

// handlePendingTransaction handles pending transaction events
func (w *WebSocketListenerAppService) handlePendingTransaction(txHash string) {
	w.logger.Debug("Received pending transaction", zap.String("tx_hash", txHash))

	// Optional: Store pending transactions or just log them
	// This depends on your use case
}

// handleLog handles contract log events
func (w *WebSocketListenerAppService) handleLog(log interface{}) {
	w.logger.Debug("Received contract log", zap.Any("log", log))

	// Optional: Store contract logs
	// This depends on your use case
}

// shouldFlush determines if buffers should be flushed
func (w *WebSocketListenerAppService) shouldFlush() bool {
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	// Flush if buffer is full or enough time has passed
	return len(w.blockBuffer) >= w.config.WebSocket.BatchSize ||
		time.Since(w.lastFlush) >= w.config.WebSocket.FlushInterval
}

// flushBuffers flushes all buffered data to database
func (w *WebSocketListenerAppService) flushBuffers(ctx context.Context) {
	w.bufferMu.Lock()
	blocksToFlush := make([]*entity.Block, len(w.blockBuffer))
	copy(blocksToFlush, w.blockBuffer)
	w.blockBuffer = w.blockBuffer[:0]

	txsToFlush := make([]*entity.Transaction, len(w.transactionBuffer))
	copy(txsToFlush, w.transactionBuffer)
	w.transactionBuffer = w.transactionBuffer[:0]

	w.lastFlush = time.Now()
	w.bufferMu.Unlock()

	if len(blocksToFlush) == 0 && len(txsToFlush) == 0 {
		return
	}

	w.logger.Info("Flushing buffers to database",
		zap.Int("blocks", len(blocksToFlush)),
		zap.Int("transactions", len(txsToFlush)))

	// Save blocks
	if len(blocksToFlush) > 0 {
		if err := w.blockRepo.CreateBlocks(ctx, blocksToFlush); err != nil {
			w.logger.Error("Failed to save blocks", zap.Error(err))
			// TODO: Add retry logic
		}
	}

	// Save transactions
	if len(txsToFlush) > 0 {
		if err := w.transactionRepo.CreateTransactions(ctx, txsToFlush); err != nil {
			w.logger.Error("Failed to save transactions", zap.Error(err))
			// TODO: Add retry logic
		}
	}

	// Update metrics
	w.updateMetrics(len(blocksToFlush), len(txsToFlush))
}

// flushWorker periodically flushes buffered data
func (w *WebSocketListenerAppService) flushWorker(ctx context.Context) {
	ticker := time.NewTicker(w.config.WebSocket.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.flushBuffers(ctx)
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// healthMonitor monitors service health
func (w *WebSocketListenerAppService) healthMonitor(ctx context.Context) {
	ticker := time.NewTicker(w.config.WebSocket.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkHealth()
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkHealth performs health checks
func (w *WebSocketListenerAppService) checkHealth() {
	if !w.webSocketService.IsConnected() {
		w.logger.Warn("WebSocket service is not connected")
		// TODO: Trigger reconnection if needed
	}
}

// notifyNewBlock sends notification about new block
func (w *WebSocketListenerAppService) notifyNewBlock(block *entity.Block) {
	if w.config.NATS.Enabled {
		w.logger.Debug("NATS notification disabled for WebSocket listener - using direct database writes only")
		// Note: Current MessagingService interface doesn't support generic publish
		// Consider extending interface if real-time notifications are needed
	}
}

// updateMetrics updates service metrics
func (w *WebSocketListenerAppService) updateMetrics(blocksProcessed, txsProcessed int) {
	w.logger.Debug("Metrics update",
		zap.Int("blocks_processed", blocksProcessed),
		zap.Int("transactions_processed", txsProcessed))

	// Note: Current MetricsRepository interface uses CrawlerMetrics/SystemHealth
	// Consider extending interface for WebSocket listener specific metrics if needed
}
