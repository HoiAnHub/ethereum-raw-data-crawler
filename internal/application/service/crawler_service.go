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
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CrawlerService handles the main crawling logic
type CrawlerService struct {
	blockchainService service.BlockchainService
	messagingService  service.MessagingService
	blockRepo         repository.BlockRepository
	txRepo            repository.TransactionRepository
	metricsRepo       repository.MetricsRepository
	config            *config.Config
	logger            *logger.Logger

	// State management
	isRunning            bool
	currentBlock         *big.Int
	workerPool           chan struct{}
	stopChan             chan struct{}
	wg                   sync.WaitGroup
	mu                   sync.RWMutex
	useExternalScheduler bool // Flag to disable internal crawler worker

	// Metrics
	metrics *CrawlerMetrics

	// Health check counters
	consecutiveHealthCheckFailures int
	lastHealthCheckTime            time.Time
}

// CrawlerMetrics holds runtime metrics
type CrawlerMetrics struct {
	BlocksProcessed       uint64
	TransactionsProcessed uint64
	ErrorCount            uint64
	LastErrorMessage      string
	StartTime             time.Time
	LastProcessedBlock    uint64
	mu                    sync.RWMutex
}

// NewCrawlerService creates new crawler service
func NewCrawlerService(
	blockchainService service.BlockchainService,
	messagingService service.MessagingService,
	blockRepo repository.BlockRepository,
	txRepo repository.TransactionRepository,
	metricsRepo repository.MetricsRepository,
	config *config.Config,
	logger *logger.Logger,
) *CrawlerService {
	return &CrawlerService{
		blockchainService: blockchainService,
		messagingService:  messagingService,
		blockRepo:         blockRepo,
		txRepo:            txRepo,
		metricsRepo:       metricsRepo,
		config:            config,
		logger:            logger.WithComponent("crawler-service"),
		workerPool:        make(chan struct{}, config.Crawler.ConcurrentWorkers),
		stopChan:          make(chan struct{}),
		metrics: &CrawlerMetrics{
			StartTime: time.Now(),
		},
	}
}

// Start starts the crawler
func (s *CrawlerService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("crawler is already running")
	}
	s.isRunning = true
	s.mu.Unlock()

	s.logger.Info("Starting crawler service")

	// Connect to blockchain
	if err := s.blockchainService.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	// Initialize starting block
	if err := s.initializeStartingBlock(ctx); err != nil {
		return fmt.Errorf("failed to initialize starting block: %w", err)
	}

	// Start worker routines
	if s.useExternalScheduler {
		// Only start metrics and health check workers when using external scheduler
		s.wg.Add(2)
		go s.metricsWorker(ctx)
		go s.healthCheckWorker(ctx)
		s.logger.Info("Crawler started in external scheduler mode")
	} else {
		// Start all workers including internal crawler worker
		s.wg.Add(3)
		go s.crawlerWorker(ctx)
		go s.metricsWorker(ctx)
		go s.healthCheckWorker(ctx)
		s.logger.Info("Crawler started in internal polling mode")
	}

	s.logger.Info("Crawler service started successfully",
		zap.String("current_block", s.currentBlock.String()))

	return nil
}

// Stop stops the crawler
func (s *CrawlerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return nil
	}
	s.isRunning = false
	s.mu.Unlock()

	s.logger.Info("Stopping crawler service")

	// Signal stop to all workers
	close(s.stopChan)

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Timeout waiting for workers to stop")
	}

	// Disconnect from blockchain
	if err := s.blockchainService.Disconnect(); err != nil {
		s.logger.Error("Error disconnecting from blockchain", zap.Error(err))
	}

	s.logger.Info("Crawler service stopped")
	return nil
}

// IsRunning returns if crawler is running
func (s *CrawlerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// GetCurrentBlock returns current block being processed
func (s *CrawlerService) GetCurrentBlock() *big.Int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentBlock == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(s.currentBlock)
}

// GetMetrics returns current crawler metrics
func (s *CrawlerService) GetMetrics() *CrawlerMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return &CrawlerMetrics{
		BlocksProcessed:       s.metrics.BlocksProcessed,
		TransactionsProcessed: s.metrics.TransactionsProcessed,
		ErrorCount:            s.metrics.ErrorCount,
		LastErrorMessage:      s.metrics.LastErrorMessage,
		StartTime:             s.metrics.StartTime,
		LastProcessedBlock:    s.metrics.LastProcessedBlock,
	}
}

// initializeStartingBlock initializes the starting block number
func (s *CrawlerService) initializeStartingBlock(ctx context.Context) error {
	// Check if we have any processed blocks in database
	lastBlock, err := s.blockRepo.GetLastProcessedBlock(ctx, s.config.Ethereum.Network)
	if err != nil {
		return err
	}

	if lastBlock != nil {
		// Resume from next block after last processed
		lastBlockNum, ok := new(big.Int).SetString(lastBlock.Number, 10)
		if !ok {
			s.logger.Error("Failed to parse last block number", zap.String("number", lastBlock.Number))
			s.currentBlock = big.NewInt(int64(s.config.Ethereum.StartBlock))
		} else {
			s.currentBlock = new(big.Int).Add(lastBlockNum, big.NewInt(1))
		}
		s.logger.Info("Resuming from last processed block",
			zap.String("last_block", lastBlock.Number),
			zap.String("current_block", s.currentBlock.String()))
	} else {
		// Start from configured start block
		s.currentBlock = big.NewInt(int64(s.config.Ethereum.StartBlock))
		s.logger.Info("Starting from configured start block",
			zap.String("start_block", s.currentBlock.String()))
	}

	return nil
}

// crawlerWorker is the main crawler worker routine
func (s *CrawlerService) crawlerWorker(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(3 * time.Second) // Process every 3 seconds (reduced from 1s)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.processNextBlocks(ctx); err != nil {
				s.updateErrorMetrics(err)
				s.logger.Error("Error processing blocks", zap.Error(err))
			}
		}
	}
}

// processNextBlocks processes the next batch of blocks
func (s *CrawlerService) processNextBlocks(ctx context.Context) error {
	// Get latest block from blockchain
	latestBlock, err := s.blockchainService.GetLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	s.logger.Debug("Checking blocks for processing",
		zap.String("current_block", s.currentBlock.String()),
		zap.String("latest_block", latestBlock.String()))

	// Check if we're caught up
	if s.currentBlock.Cmp(latestBlock) > 0 {
		// We're ahead of the latest block, wait
		s.logger.Debug("Caught up with latest block, waiting")
		return nil
	}

	// Calculate batch end block
	batchSize := big.NewInt(int64(s.config.Crawler.BatchSize))
	endBlock := new(big.Int).Add(s.currentBlock, batchSize)
	if endBlock.Cmp(latestBlock) > 0 {
		endBlock.Set(latestBlock)
	}

	s.logger.Info("Processing block range",
		zap.String("start", s.currentBlock.String()),
		zap.String("end", endBlock.String()),
		zap.String("latest", latestBlock.String()))

	// Process blocks in parallel
	return s.processBlockRange(ctx, s.currentBlock, endBlock)
}

// processBlockRange processes a range of blocks
func (s *CrawlerService) processBlockRange(ctx context.Context, startBlock, endBlock *big.Int) error {
	var wg sync.WaitGroup
	errChan := make(chan error, s.config.Crawler.ConcurrentWorkers)

	// Add delay between batches to prevent overwhelming API
	batchDelay := 5 * time.Second // Increased from 2s to 5s
	blockCount := 0

	for i := new(big.Int).Set(startBlock); i.Cmp(endBlock) <= 0; i.Add(i, big.NewInt(1)) {
		// Add delay every batch to prevent rate limiting (but not on the first batch)
		if blockCount > 0 && blockCount%s.config.Crawler.BatchSize == 0 {
			s.logger.Info("Pausing between batches to avoid rate limiting",
				zap.Duration("delay", batchDelay),
				zap.Int("blocks_processed", blockCount))
			time.Sleep(batchDelay)
		}

		// Add small delay between blocks to be gentler on API
		if blockCount > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		// Acquire worker slot
		s.workerPool <- struct{}{}
		wg.Add(1)

		go func(blockNum *big.Int) {
			defer func() {
				wg.Done()
				<-s.workerPool // Release worker slot
			}()

			if err := s.processBlock(ctx, new(big.Int).Set(blockNum)); err != nil {
				errChan <- err
			}
		}(new(big.Int).Set(i))

		blockCount++
	}

	// Wait for all workers to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors processing block range: %v", errors)
	}

	// Update current block
	s.mu.Lock()
	s.currentBlock.Add(endBlock, big.NewInt(1))
	s.mu.Unlock()

	return nil
}

// ProcessSpecificBlock processes a specific block (used by scheduler)
func (s *CrawlerService) ProcessSpecificBlock(ctx context.Context, blockNumber *big.Int) error {
	if !s.IsRunning() {
		return fmt.Errorf("crawler service is not running")
	}

	s.logger.Info("Processing specific block from scheduler",
		zap.String("block_number", blockNumber.String()))

	// Acquire worker slot
	s.workerPool <- struct{}{}
	defer func() {
		<-s.workerPool // Release worker slot
	}()

	return s.processBlock(ctx, blockNumber)
}

// SetExternalSchedulerMode sets whether to use external scheduler
func (s *CrawlerService) SetExternalSchedulerMode(useExternal bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.useExternalScheduler = useExternal
}

// processBlock processes a single block
func (s *CrawlerService) processBlock(ctx context.Context, blockNumber *big.Int) error {
	logger := s.logger.WithBlock(blockNumber.Uint64())
	logger.Info("Starting to process block", zap.Uint64("block_number", blockNumber.Uint64()))

	// Create timeout context for the entire block processing
	blockCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Get block from blockchain
	block, err := s.blockchainService.GetBlockByNumber(blockCtx, blockNumber)
	if err != nil {
		logger.Error("Failed to get block", zap.Error(err))
		return fmt.Errorf("failed to get block %s: %w", blockNumber.String(), err)
	}

	// Check if block already exists by number
	existingBlock, err := s.blockRepo.GetBlockByNumber(blockCtx, blockNumber)
	if err != nil {
		logger.Error("Failed to check if block exists", zap.Error(err))
		return fmt.Errorf("failed to check block existence %s: %w", blockNumber.String(), err)
	}

	if existingBlock != nil {
		logger.Info("Block already exists, skipping block save",
			zap.String("block_hash", block.Hash),
			zap.String("existing_hash", existingBlock.Hash))
	} else {
		// Save block to database
		if err := s.blockRepo.CreateBlock(blockCtx, block); err != nil {
			// Check if it's a duplicate key error (race condition)
			if strings.Contains(err.Error(), "E11000") || strings.Contains(err.Error(), "duplicate key") {
				logger.Warn("Block already exists (race condition), continuing",
					zap.String("block_hash", block.Hash))
			} else {
				logger.Error("Failed to save block", zap.Error(err))
				return fmt.Errorf("failed to save block %s: %w", blockNumber.String(), err)
			}
		} else {
			logger.Info("Block saved to database")
		}
	}

	// Get all transactions for this block
	logger.Info("Getting transactions for block", zap.Int("tx_hash_count", len(block.TransactionHashes)))
	transactions, err := s.blockchainService.GetTransactionsByBlock(blockCtx, blockNumber)
	if err != nil {
		logger.Error("Failed to get transactions", zap.Error(err))
		return fmt.Errorf("failed to get transactions for block %s: %w", blockNumber.String(), err)
	}
	logger.Info("Retrieved transactions", zap.Int("count", len(transactions)))

	// Save transactions to database
	if len(transactions) > 0 {
		if err := s.saveTransactions(blockCtx, transactions, logger); err != nil {
			return fmt.Errorf("failed to save transactions for block %s: %w", blockNumber.String(), err)
		}
		logger.Info("Transactions saved to database", zap.Int("count", len(transactions)))
	}

	// Mark block as processed
	if err := s.blockRepo.MarkBlockAsProcessed(ctx, block.Hash); err != nil {
		logger.Error("Failed to mark block as processed", zap.Error(err))
	}

	// Update metrics
	s.updateProcessingMetrics(block, transactions)

	logger.Info("Block processed successfully",
		zap.Int("transaction_count", len(transactions)))

	return nil
}

// saveTransactions saves transactions using configured method (upsert or insert)
func (s *CrawlerService) saveTransactions(ctx context.Context, transactions []*entity.Transaction, logger *logger.Logger) error {
	start := time.Now()
	txCount := len(transactions)

	// Publish transactions to NATS JetStream before saving to MongoDB
	if err := s.publishTransactions(ctx, transactions, logger); err != nil {
		logger.Warn("Failed to publish transactions to messaging service",
			zap.Error(err),
			zap.Int("transaction_count", txCount))
		// Continue with database save even if messaging fails
	}

	if s.config.Crawler.UseUpsert {
		// Try upsert first
		logger.Debug("Attempting batch upsert", zap.Int("transaction_count", txCount))

		if err := s.txRepo.UpsertTransactions(ctx, transactions); err != nil {
			duration := time.Since(start)
			logger.Warn("Batch upsert failed",
				zap.Error(err),
				zap.Int("transaction_count", txCount),
				zap.Duration("duration", duration))

			// Fallback to insert if configured
			if s.config.Crawler.UpsertFallback {
				logger.Info("Falling back to batch insert", zap.Int("transaction_count", txCount))
				fallbackStart := time.Now()

				if insertErr := s.txRepo.CreateTransactions(ctx, transactions); insertErr != nil {
					fallbackDuration := time.Since(fallbackStart)

					// Check if this is a duplicate key error which means data already exists
					if strings.Contains(insertErr.Error(), "E11000") || strings.Contains(insertErr.Error(), "duplicate key") {
						logger.Warn("Transactions already exist in database, considering as success",
							zap.Int("transaction_count", txCount),
							zap.Duration("fallback_duration", fallbackDuration),
							zap.Duration("total_duration", time.Since(start)))
						return nil // Consider this as success since data is already there
					}

					logger.Error("Fallback batch insert also failed",
						zap.Error(insertErr),
						zap.Int("transaction_count", txCount),
						zap.Duration("fallback_duration", fallbackDuration),
						zap.Duration("total_duration", time.Since(start)))
					return fmt.Errorf("both upsert and insert failed: upsert=%w, insert=%w", err, insertErr)
				}

				fallbackDuration := time.Since(fallbackStart)
				logger.Info("Fallback batch insert succeeded",
					zap.Int("transaction_count", txCount),
					zap.Duration("fallback_duration", fallbackDuration),
					zap.Duration("total_duration", time.Since(start)))
				return nil
			}
			return err
		}

		duration := time.Since(start)
		logger.Debug("Batch upsert succeeded",
			zap.Int("transaction_count", txCount),
			zap.Duration("duration", duration))
		return nil
	} else {
		// Use traditional insert
		logger.Debug("Attempting batch insert", zap.Int("transaction_count", txCount))

		if err := s.txRepo.CreateTransactions(ctx, transactions); err != nil {
			duration := time.Since(start)
			logger.Error("Batch insert failed",
				zap.Error(err),
				zap.Int("transaction_count", txCount),
				zap.Duration("duration", duration))
			return err
		}

		duration := time.Since(start)
		logger.Debug("Batch insert succeeded",
			zap.Int("transaction_count", txCount),
			zap.Duration("duration", duration))
		return nil
	}
}

// publishTransactions publishes transactions to messaging service
func (s *CrawlerService) publishTransactions(ctx context.Context, transactions []*entity.Transaction, logger *logger.Logger) error {
	if s.messagingService == nil {
		logger.Debug("Messaging service not available, skipping transaction publishing")
		return nil
	}

	if !s.messagingService.IsConnected() {
		logger.Debug("Messaging service not connected, skipping transaction publishing")
		return nil
	}

	start := time.Now()
	err := s.messagingService.PublishTransactions(ctx, transactions)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Failed to publish transactions to messaging service",
			zap.Error(err),
			zap.Int("transaction_count", len(transactions)),
			zap.Duration("duration", duration))
		return err
	}

	logger.Debug("Successfully published transactions to messaging service",
		zap.Int("transaction_count", len(transactions)),
		zap.Duration("duration", duration))

	return nil
}

// metricsWorker periodically saves metrics to database
func (s *CrawlerService) metricsWorker(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Save metrics every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.saveMetrics(ctx); err != nil {
				s.logger.Error("Failed to save metrics", zap.Error(err))
			}
		}
	}
}

// healthCheckWorker periodically performs health checks
func (s *CrawlerService) healthCheckWorker(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.Monitoring.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.performHealthCheck(ctx); err != nil {
				s.logger.Error("Health check failed", zap.Error(err))
			}
		}
	}
}

// saveMetrics saves current metrics to database
func (s *CrawlerService) saveMetrics(ctx context.Context) error {
	metrics := s.GetMetrics()

	// Get current block from blockchain
	latestBlock, err := s.blockchainService.GetLatestBlockNumber(ctx)
	if err != nil {
		latestBlock = big.NewInt(0)
	}

	// Calculate processing rate
	elapsed := time.Since(metrics.StartTime)
	var blocksPerSecond, transactionsPerSecond float64
	if elapsed.Seconds() > 0 {
		blocksPerSecond = float64(metrics.BlocksProcessed) / elapsed.Seconds()
		transactionsPerSecond = float64(metrics.TransactionsProcessed) / elapsed.Seconds()
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metricsEntity := &entity.CrawlerMetrics{
		Timestamp:             time.Now(),
		LastProcessedBlock:    metrics.LastProcessedBlock,
		CurrentBlock:          latestBlock.Uint64(),
		BlocksProcessed:       metrics.BlocksProcessed,
		BlocksPerSecond:       blocksPerSecond,
		TransactionsProcessed: metrics.TransactionsProcessed,
		TransactionsPerSecond: transactionsPerSecond,
		ErrorCount:            metrics.ErrorCount,
		LastErrorMessage:      metrics.LastErrorMessage,
		MemoryUsage:           memStats.Alloc,
		GoroutineCount:        runtime.NumGoroutine(),
		Network:               s.config.Ethereum.Network,
	}

	return s.metricsRepo.SaveCrawlerMetrics(ctx, metricsEntity)
}

// performHealthCheck performs comprehensive system health check
func (s *CrawlerService) performHealthCheck(ctx context.Context) error {
	start := time.Now()
	componentsHealth := make(map[string]entity.ComponentHealth)

	overallStatus := entity.HealthStatusHealthy
	var messages []string

	// Check blockchain connection
	blockchainStart := time.Now()
	blockchainErr := s.blockchainService.HealthCheck(ctx)
	blockchainLatency := time.Since(blockchainStart)

	blockchainStatus := entity.HealthStatusHealthy
	blockchainMessage := "Blockchain connection healthy"

	if blockchainErr != nil {
		blockchainStatus = entity.HealthStatusUnhealthy
		blockchainMessage = fmt.Sprintf("Blockchain connection failed: %v", blockchainErr)
		overallStatus = entity.HealthStatusUnhealthy
		messages = append(messages, blockchainMessage)
	} else if blockchainLatency > 5*time.Second {
		blockchainStatus = entity.HealthStatusDegraded
		blockchainMessage = "High blockchain network latency detected"
		if overallStatus == entity.HealthStatusHealthy {
			overallStatus = entity.HealthStatusDegraded
		}
		messages = append(messages, blockchainMessage)
	}

	componentsHealth["blockchain"] = entity.ComponentHealth{
		Status:       blockchainStatus,
		LastChecked:  time.Now(),
		Message:      blockchainMessage,
		ResponseTime: blockchainLatency,
	}

	// Check MongoDB connection
	mongoStart := time.Now()
	mongoErr := s.checkMongoDBHealth(ctx)
	mongoLatency := time.Since(mongoStart)

	mongoStatus := entity.HealthStatusHealthy
	mongoMessage := "MongoDB connection healthy"

	if mongoErr != nil {
		mongoStatus = entity.HealthStatusUnhealthy
		mongoMessage = fmt.Sprintf("MongoDB connection failed: %v", mongoErr)
		overallStatus = entity.HealthStatusUnhealthy
		messages = append(messages, mongoMessage)
	} else if mongoLatency > 2*time.Second {
		mongoStatus = entity.HealthStatusDegraded
		mongoMessage = "High MongoDB latency detected"
		if overallStatus == entity.HealthStatusHealthy {
			overallStatus = entity.HealthStatusDegraded
		}
		messages = append(messages, mongoMessage)
	}

	componentsHealth["mongodb"] = entity.ComponentHealth{
		Status:       mongoStatus,
		LastChecked:  time.Now(),
		Message:      mongoMessage,
		ResponseTime: mongoLatency,
	}

	// Check messaging service (NATS)
	messagingStart := time.Now()
	messagingErr := s.checkMessagingHealth(ctx)
	messagingLatency := time.Since(messagingStart)

	messagingStatus := entity.HealthStatusHealthy
	messagingMessage := "Messaging service healthy"

	if messagingErr != nil {
		messagingStatus = entity.HealthStatusDegraded // Non-critical for crawler operation
		messagingMessage = fmt.Sprintf("Messaging service issues: %v", messagingErr)
		if overallStatus == entity.HealthStatusHealthy {
			overallStatus = entity.HealthStatusDegraded
		}
		messages = append(messages, messagingMessage)
	}

	componentsHealth["messaging"] = entity.ComponentHealth{
		Status:       messagingStatus,
		LastChecked:  time.Now(),
		Message:      messagingMessage,
		ResponseTime: messagingLatency,
	}

	// Determine overall message
	overallMessage := "All systems operational"
	if len(messages) > 0 {
		overallMessage = strings.Join(messages, "; ")
	}

	health := &entity.SystemHealth{
		Timestamp:        time.Now(),
		Status:           overallStatus,
		Message:          overallMessage,
		Network:          s.config.Ethereum.Network,
		ComponentsHealth: componentsHealth,
	}

	// Try to save health status, but don't fail if MongoDB is down
	if err := s.metricsRepo.SaveSystemHealth(ctx, health); err != nil {
		s.logger.Warn("Failed to save system health metrics", zap.Error(err))
		// Don't return error here as health check should continue even if we can't save metrics
	}

	// Update health check tracking
	s.mu.Lock()
	s.lastHealthCheckTime = time.Now()
	if overallStatus == entity.HealthStatusUnhealthy {
		s.consecutiveHealthCheckFailures++
	} else {
		s.consecutiveHealthCheckFailures = 0
	}
	failures := s.consecutiveHealthCheckFailures
	s.mu.Unlock()

	// Log health status
	s.logger.Info("Health check completed",
		zap.String("overall_status", string(overallStatus)),
		zap.String("message", overallMessage),
		zap.Duration("total_time", time.Since(start)),
		zap.Int("consecutive_failures", failures))

	// Take recovery actions if needed
	if failures >= 3 {
		s.logger.Warn("Multiple consecutive health check failures detected, attempting recovery",
			zap.Int("failure_count", failures))

		// Attempt to recover connections
		go s.attemptRecovery(context.Background())
	}

	return nil
}

// checkMongoDBHealth checks MongoDB connection health
func (s *CrawlerService) checkMongoDBHealth(ctx context.Context) error {
	// Try to get last processed block to test MongoDB connection
	_, err := s.blockRepo.GetLastProcessedBlock(ctx, s.config.Ethereum.Network)
	if err != nil {
		// If it's a "no documents" error, MongoDB is working fine
		if strings.Contains(err.Error(), "no documents") {
			return nil
		}
		return fmt.Errorf("mongodb health check failed: %w", err)
	}
	return nil
}

// checkMessagingHealth checks messaging service health
func (s *CrawlerService) checkMessagingHealth(ctx context.Context) error {
	// For now, we'll assume messaging is healthy if the service is not nil
	// In the future, we could add a proper health check method to the messaging service
	if s.messagingService == nil {
		return fmt.Errorf("messaging service is not initialized")
	}
	return nil
}

// attemptRecovery attempts to recover from health check failures
func (s *CrawlerService) attemptRecovery(ctx context.Context) {
	s.logger.Info("Starting recovery process")

	// Try to reconnect to blockchain
	if err := s.blockchainService.Connect(ctx); err != nil {
		s.logger.Error("Failed to reconnect to blockchain during recovery", zap.Error(err))
	} else {
		s.logger.Info("Successfully reconnected to blockchain")
	}

	// Reset failure counter on successful recovery attempt
	s.mu.Lock()
	s.consecutiveHealthCheckFailures = 0
	s.mu.Unlock()

	s.logger.Info("Recovery process completed")
}

// updateProcessingMetrics updates processing metrics
func (s *CrawlerService) updateProcessingMetrics(block *entity.Block, transactions []*entity.Transaction) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.BlocksProcessed++
	s.metrics.TransactionsProcessed += uint64(len(transactions))

	// Convert block number string to uint64
	if blockNum, ok := new(big.Int).SetString(block.Number, 10); ok {
		s.metrics.LastProcessedBlock = blockNum.Uint64()
	}
}

// updateErrorMetrics updates error metrics
func (s *CrawlerService) updateErrorMetrics(err error) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.ErrorCount++
	s.metrics.LastErrorMessage = err.Error()
}
