package service

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/service"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SchedulerMode defines the mode of operation for the scheduler
type SchedulerMode string

const (
	// PollingMode uses traditional polling every few seconds
	PollingMode SchedulerMode = "polling"
	// RealtimeMode uses WebSocket to listen for new blocks
	RealtimeMode SchedulerMode = "realtime"
	// HybridMode uses both realtime and polling as fallback
	HybridMode SchedulerMode = "hybrid"
)

// SchedulerService manages block crawling scheduling
type SchedulerService struct {
	blockScheduler service.BlockSchedulerService
	crawlerService *CrawlerService
	config         *config.Config
	logger         *logger.Logger
	mode           SchedulerMode
	isRunning      bool
	stopChan       chan struct{}
	mu             sync.RWMutex

	// Fallback polling
	pollingTicker   *time.Ticker
	pollingStopChan chan struct{} // Channel to stop polling worker
	lastBlockTime   time.Time
	fallbackTimeout time.Duration

	// Error tracking for blocks
	failedBlocks  map[string]int       // block_number -> failure_count
	skippedBlocks map[string]time.Time // block_number -> skip_time
	maxRetries    int
	skipDuration  time.Duration
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(
	blockScheduler service.BlockSchedulerService,
	crawlerService *CrawlerService,
	config *config.Config,
	logger *logger.Logger,
) *SchedulerService {
	// Validate required dependencies
	if crawlerService == nil {
		panic("crawlerService cannot be nil")
	}
	if config == nil {
		panic("config cannot be nil")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}

	// Determine mode from config
	var mode SchedulerMode
	switch config.Scheduler.Mode {
	case "polling":
		mode = PollingMode
	case "realtime":
		mode = RealtimeMode
	case "hybrid":
		mode = HybridMode
	default:
		mode = HybridMode // Default fallback
	}

	// Get max retries and skip duration from config with defaults
	maxRetries := 3
	skipDuration := time.Minute

	if config.Scheduler.MaxRetries > 0 {
		maxRetries = config.Scheduler.MaxRetries
	}
	if config.Scheduler.SkipDuration > 0 {
		skipDuration = config.Scheduler.SkipDuration
	}

	return &SchedulerService{
		blockScheduler:  blockScheduler, // Can be nil for polling-only mode
		crawlerService:  crawlerService,
		config:          config,
		logger:          logger.WithComponent("scheduler-service"),
		mode:            mode,
		stopChan:        make(chan struct{}),
		pollingStopChan: nil, // Will be created when polling starts
		fallbackTimeout: config.Scheduler.FallbackTimeout,
		failedBlocks:    make(map[string]int),
		skippedBlocks:   make(map[string]time.Time),
		maxRetries:      maxRetries,
		skipDuration:    skipDuration,
	}
}

// Start starts the scheduler service
func (s *SchedulerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler service is already running")
	}

	s.logger.Info("Starting scheduler service", zap.String("mode", string(s.mode)))

	// Start based on mode
	switch s.mode {
	case RealtimeMode:
		return s.startRealtimeMode(ctx)
	case PollingMode:
		return s.startPollingMode(ctx)
	case HybridMode:
		return s.startHybridMode(ctx)
	default:
		return fmt.Errorf("unknown scheduler mode: %s", s.mode)
	}
}

// Stop stops the scheduler service
func (s *SchedulerService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	s.logger.Info("Stopping scheduler service")

	close(s.stopChan)
	s.isRunning = false

	// Stop block scheduler
	if s.blockScheduler != nil {
		if err := s.blockScheduler.Stop(); err != nil {
			s.logger.Error("Failed to stop block scheduler", zap.Error(err))
		}
	}

	// Stop polling ticker
	if s.pollingTicker != nil {
		s.pollingTicker.Stop()
		s.pollingTicker = nil
	}

	// Stop polling worker
	if s.pollingStopChan != nil {
		close(s.pollingStopChan)
		s.pollingStopChan = nil
	}

	return nil
}

// IsRunning checks if the scheduler is running
func (s *SchedulerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// SetMode sets the scheduler mode
func (s *SchedulerService) SetMode(mode SchedulerMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("cannot change mode while scheduler is running")
	}

	s.mode = mode
	s.logger.Info("Scheduler mode changed", zap.String("mode", string(mode)))
	return nil
}

// GetMode returns the current scheduler mode
func (s *SchedulerService) GetMode() SchedulerMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

// startRealtimeMode starts the scheduler in realtime mode
func (s *SchedulerService) startRealtimeMode(ctx context.Context) error {
	if err := s.blockScheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start block scheduler: %w", err)
	}

	if err := s.blockScheduler.SubscribeNewBlocks(ctx, s.handleNewBlock); err != nil {
		return fmt.Errorf("failed to subscribe to new blocks: %w", err)
	}

	s.isRunning = true
	s.logger.Info("Scheduler started in realtime mode")
	return nil
}

// startPollingMode starts the scheduler in polling mode
func (s *SchedulerService) startPollingMode(ctx context.Context) error {
	// Validate dependencies
	if s.config == nil {
		return fmt.Errorf("config is nil")
	}
	if s.crawlerService == nil {
		return fmt.Errorf("crawlerService is nil")
	}
	if s.config.Scheduler.PollingInterval <= 0 {
		return fmt.Errorf("invalid polling interval: %v", s.config.Scheduler.PollingInterval)
	}

	// Use configured polling interval
	s.pollingTicker = time.NewTicker(s.config.Scheduler.PollingInterval)
	s.pollingStopChan = make(chan struct{})

	// Validate before starting worker
	if s.pollingTicker == nil || s.pollingStopChan == nil {
		return fmt.Errorf("failed to create polling resources")
	}

	go s.pollingWorker(ctx, s.pollingTicker, s.pollingStopChan)

	s.isRunning = true
	s.logger.Info("Scheduler started in polling mode",
		zap.Duration("polling_interval", s.config.Scheduler.PollingInterval))

	// Update last block time to current time for polling mode
	s.mu.Lock()
	s.lastBlockTime = time.Now()
	s.mu.Unlock()

	return nil
}

// startHybridMode starts the scheduler in hybrid mode
func (s *SchedulerService) startHybridMode(ctx context.Context) error {
	// Start realtime mode first
	if err := s.startRealtimeMode(ctx); err != nil {
		s.logger.Warn("Failed to start realtime mode, falling back to polling", zap.Error(err))
		return s.startPollingMode(ctx)
	}

	// Start fallback polling monitor
	go s.fallbackMonitor(ctx)

	s.logger.Info("Scheduler started in hybrid mode")
	return nil
}

// handleNewBlock handles new block notifications from WebSocket
func (s *SchedulerService) handleNewBlock(blockNumber *big.Int) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic recovered in handleNewBlock",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	// Validate input
	if blockNumber == nil {
		s.logger.Error("handleNewBlock called with nil blockNumber")
		return
	}

	blockNumStr := blockNumber.String()
	s.logger.Info("Received new block notification",
		zap.String("block_number", blockNumStr))

	// Update last block time
	s.mu.Lock()
	s.lastBlockTime = time.Now()

	// Check if this block is currently being skipped due to previous failures
	if skipTime, isSkipped := s.skippedBlocks[blockNumStr]; isSkipped {
		if time.Since(skipTime) < s.skipDuration {
			s.logger.Warn("Skipping block due to recent failures",
				zap.String("block_number", blockNumStr),
				zap.Duration("remaining_skip_time", s.skipDuration-time.Since(skipTime)))
			s.mu.Unlock()
			return
		} else {
			// Skip duration has passed, remove from skipped blocks
			delete(s.skippedBlocks, blockNumStr)
			delete(s.failedBlocks, blockNumStr) // Reset failure count
		}
	}
	s.mu.Unlock()

	// Trigger crawler to process the new block with nil check
	if s.crawlerService == nil {
		s.logger.Error("crawlerService is nil in handleNewBlock")
		return
	}

	ctx := context.Background()
	if err := s.crawlerService.ProcessSpecificBlock(ctx, blockNumber); err != nil {
		s.handleBlockProcessingError(blockNumStr, err)
	} else {
		// Success: remove from failed blocks if it was there
		s.mu.Lock()
		delete(s.failedBlocks, blockNumStr)
		s.mu.Unlock()
	}
}

// handleBlockProcessingError handles errors during block processing with retry logic
func (s *SchedulerService) handleBlockProcessingError(blockNumStr string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Increment failure count
	failureCount := s.failedBlocks[blockNumStr]
	failureCount++
	s.failedBlocks[blockNumStr] = failureCount

	s.logger.Error("Failed to process new block",
		zap.String("block_number", blockNumStr),
		zap.Int("failure_count", failureCount),
		zap.Int("max_retries", s.maxRetries),
		zap.Error(err))

	// Check if we've exceeded max retries
	if failureCount >= s.maxRetries {
		// Add to skipped blocks to prevent infinite retries
		s.skippedBlocks[blockNumStr] = time.Now()
		s.logger.Warn("Block processing failed too many times, skipping temporarily",
			zap.String("block_number", blockNumStr),
			zap.Int("failure_count", failureCount),
			zap.Duration("skip_duration", s.skipDuration))

		// Check if this is a MongoDB upsert/duplicate error - these might resolve themselves
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "_id") && strings.Contains(errorMsg, "immutable") ||
			strings.Contains(errorMsg, "duplicate key error") ||
			strings.Contains(errorMsg, "E11000") {
			s.logger.Info("MongoDB conflict error detected, block might already be processed",
				zap.String("block_number", blockNumStr))
		}
	}
}

// pollingWorker runs the polling loop
func (s *SchedulerService) pollingWorker(ctx context.Context, ticker *time.Ticker, stopChan chan struct{}) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic recovered in pollingWorker",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}

		if ticker != nil {
			ticker.Stop()
		}
		s.logger.Info("Polling worker stopped")
	}()

	// Validate inputs
	if ticker == nil {
		s.logger.Error("pollingWorker called with nil ticker")
		return
	}

	if stopChan == nil {
		s.logger.Error("pollingWorker called with nil stopChan")
		return
	}

	if s.crawlerService == nil {
		s.logger.Error("pollingWorker called with nil crawlerService")
		return
	}

	s.logger.Info("Polling worker started")

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("Polling worker received global stop signal")
			return
		case <-stopChan:
			s.logger.Info("Polling worker received specific stop signal")
			return
		case <-ctx.Done():
			s.logger.Info("Polling worker context cancelled")
			return
		case <-ticker.C:
			s.logger.Debug("Polling worker tick - checking for new blocks")

			// Add nil check before calling crawlerService
			if s.crawlerService != nil {
				if err := s.crawlerService.processNextBlocks(ctx); err != nil {
					s.logger.Error("Error in polling worker", zap.Error(err))
				} else {
					// Update last block time on successful processing
					s.mu.Lock()
					s.lastBlockTime = time.Now()
					s.mu.Unlock()
					s.logger.Debug("Polling worker completed successfully")
				}
			} else {
				s.logger.Error("crawlerService is nil in polling worker")
				return
			}
		}
	}
}

// fallbackMonitor monitors for WebSocket failures and activates polling fallback
func (s *SchedulerService) fallbackMonitor(ctx context.Context) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic recovered in fallbackMonitor",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	s.logger.Info("Fallback monitor started")

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("Fallback monitor received stop signal")
			return
		case <-ctx.Done():
			s.logger.Info("Fallback monitor context cancelled")
			return
		case <-ticker.C:
			s.mu.RLock()
			timeSinceLastBlock := time.Since(s.lastBlockTime)
			pollingActive := s.pollingTicker != nil
			s.mu.RUnlock()

			// Check if WebSocket is working with nil check
			wsWorking := s.blockScheduler != nil && s.blockScheduler.IsRunning()

			s.logger.Debug("Fallback monitor check",
				zap.Duration("time_since_last_block", timeSinceLastBlock),
				zap.Bool("websocket_running", wsWorking),
				zap.Bool("polling_active", pollingActive))

			// If no blocks received for fallbackTimeout, start polling
			if timeSinceLastBlock > s.fallbackTimeout {
				if !pollingActive {
					s.logger.Warn("No blocks received via WebSocket, starting fallback polling",
						zap.Duration("time_since_last_block", timeSinceLastBlock),
						zap.Bool("websocket_running", wsWorking))

					// Create polling with proper validation
					s.mu.Lock()
					if s.config != nil && s.config.Scheduler.PollingInterval > 0 {
						s.pollingTicker = time.NewTicker(s.config.Scheduler.PollingInterval)
						ticker := s.pollingTicker // Store reference before unlocking

						stopChan := make(chan struct{})
						s.pollingStopChan = stopChan
						s.mu.Unlock()

						// Start pollingWorker with proper nil checks
						if ticker != nil && stopChan != nil && s.crawlerService != nil {
							go s.pollingWorker(ctx, ticker, stopChan)
						} else {
							s.logger.Error("Failed to start polling worker: nil dependencies",
								zap.Bool("ticker_nil", ticker == nil),
								zap.Bool("stopChan_nil", stopChan == nil),
								zap.Bool("crawlerService_nil", s.crawlerService == nil))
						}
					} else {
						s.mu.Unlock()
						s.logger.Error("Invalid polling configuration")
					}
				}
			} else {
				// If blocks are coming via WebSocket, stop polling
				if pollingActive && wsWorking {
					s.logger.Info("WebSocket blocks resumed, stopping fallback polling")
					s.mu.Lock()
					if s.pollingTicker != nil {
						s.pollingTicker.Stop()
						s.pollingTicker = nil
					}
					if s.pollingStopChan != nil {
						// Use safe channel close
						select {
						case <-s.pollingStopChan:
							// Channel already closed
						default:
							close(s.pollingStopChan)
						}
						s.pollingStopChan = nil
					}
					s.mu.Unlock()
				}
			}
		}
	}
}

// GetStats returns scheduler statistics
func (s *SchedulerService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"mode":                    string(s.mode),
		"is_running":              s.isRunning,
		"block_scheduler_running": false,
		"polling_active":          s.pollingTicker != nil,
		"last_block_time":         s.lastBlockTime,
	}

	if s.blockScheduler != nil {
		stats["block_scheduler_running"] = s.blockScheduler.IsRunning()
	}

	return stats
}
