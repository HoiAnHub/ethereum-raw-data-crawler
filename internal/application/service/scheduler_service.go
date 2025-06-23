package service

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/service"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"fmt"
	"math/big"
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
	lastBlockTime   time.Time
	fallbackTimeout time.Duration
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(
	blockScheduler service.BlockSchedulerService,
	crawlerService *CrawlerService,
	config *config.Config,
	logger *logger.Logger,
) *SchedulerService {
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

	return &SchedulerService{
		blockScheduler:  blockScheduler,
		crawlerService:  crawlerService,
		config:          config,
		logger:          logger.WithComponent("scheduler-service"),
		mode:            mode,
		stopChan:        make(chan struct{}),
		fallbackTimeout: config.Scheduler.FallbackTimeout,
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
	// Use configured polling interval
	s.pollingTicker = time.NewTicker(s.config.Scheduler.PollingInterval)

	go s.pollingWorker(ctx)

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
	s.logger.Info("Received new block notification",
		zap.String("block_number", blockNumber.String()))

	// Update last block time
	s.mu.Lock()
	s.lastBlockTime = time.Now()
	s.mu.Unlock()

	// Trigger crawler to process the new block
	ctx := context.Background()
	if err := s.crawlerService.ProcessSpecificBlock(ctx, blockNumber); err != nil {
		s.logger.Error("Failed to process new block",
			zap.String("block_number", blockNumber.String()),
			zap.Error(err))
	}
}

// pollingWorker runs the polling loop
func (s *SchedulerService) pollingWorker(ctx context.Context) {
	defer func() {
		if s.pollingTicker != nil {
			s.pollingTicker.Stop()
		}
	}()

	s.logger.Info("Polling worker started")

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("Polling worker received stop signal")
			return
		case <-ctx.Done():
			s.logger.Info("Polling worker context cancelled")
			return
		case <-s.pollingTicker.C:
			s.logger.Debug("Polling worker tick - checking for new blocks")
			if err := s.crawlerService.processNextBlocks(ctx); err != nil {
				s.logger.Error("Error in polling worker", zap.Error(err))
			} else {
				// Update last block time on successful processing
				s.mu.Lock()
				s.lastBlockTime = time.Now()
				s.mu.Unlock()
				s.logger.Debug("Polling worker completed successfully")
			}
		}
	}
}

// fallbackMonitor monitors for WebSocket failures and activates polling fallback
func (s *SchedulerService) fallbackMonitor(ctx context.Context) {
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

			// Check if WebSocket is working
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

					s.mu.Lock()
					s.pollingTicker = time.NewTicker(s.config.Scheduler.PollingInterval)
					s.mu.Unlock()

					go s.pollingWorker(ctx)
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
