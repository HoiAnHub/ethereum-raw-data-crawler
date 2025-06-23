package service

import (
	"context"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBlockSchedulerService is a mock implementation of BlockSchedulerService
type MockBlockSchedulerService struct {
	mock.Mock
}

func (m *MockBlockSchedulerService) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBlockSchedulerService) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBlockSchedulerService) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockBlockSchedulerService) SubscribeNewBlocks(ctx context.Context, callback func(*big.Int)) error {
	args := m.Called(ctx, callback)
	return args.Error(0)
}

func (m *MockBlockSchedulerService) Unsubscribe() error {
	args := m.Called()
	return args.Error(0)
}

// MockCrawlerService is a mock implementation of CrawlerService
type MockCrawlerService struct {
	mock.Mock
}

func (m *MockCrawlerService) ProcessSpecificBlock(ctx context.Context, blockNumber *big.Int) error {
	args := m.Called(ctx, blockNumber)
	return args.Error(0)
}

func (m *MockCrawlerService) processNextBlocks(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewSchedulerService(t *testing.T) {
	mockBlockScheduler := &MockBlockSchedulerService{}
	mockCrawlerService := &MockCrawlerService{}
	
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Mode:            "hybrid",
			FallbackTimeout: 30 * time.Second,
		},
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewSchedulerService(mockBlockScheduler, mockCrawlerService, cfg, logger)

	assert.NotNil(t, scheduler)
	assert.Equal(t, HybridMode, scheduler.GetMode())
	assert.False(t, scheduler.IsRunning())
}

func TestSchedulerService_SetMode(t *testing.T) {
	mockBlockScheduler := &MockBlockSchedulerService{}
	mockCrawlerService := &MockCrawlerService{}
	
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Mode:            "polling",
			FallbackTimeout: 30 * time.Second,
		},
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewSchedulerService(mockBlockScheduler, mockCrawlerService, cfg, logger)

	// Test setting mode when not running
	err := scheduler.SetMode(RealtimeMode)
	assert.NoError(t, err)
	assert.Equal(t, RealtimeMode, scheduler.GetMode())

	// Test setting mode when running (should fail)
	scheduler.isRunning = true
	err = scheduler.SetMode(PollingMode)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change mode while scheduler is running")
}

func TestSchedulerService_StartPollingMode(t *testing.T) {
	mockBlockScheduler := &MockBlockSchedulerService{}
	mockCrawlerService := &MockCrawlerService{}
	
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Mode:            "polling",
			PollingInterval: 1 * time.Second,
			FallbackTimeout: 30 * time.Second,
		},
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewSchedulerService(mockBlockScheduler, mockCrawlerService, cfg, logger)

	ctx := context.Background()
	
	// Mock crawler service to expect processNextBlocks calls
	mockCrawlerService.On("processNextBlocks", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Wait a bit to ensure polling starts
	time.Sleep(100 * time.Millisecond)

	// Stop the scheduler
	err = scheduler.Stop()
	assert.NoError(t, err)
	assert.False(t, scheduler.IsRunning())
}

func TestSchedulerService_StartRealtimeMode(t *testing.T) {
	mockBlockScheduler := &MockBlockSchedulerService{}
	mockCrawlerService := &MockCrawlerService{}
	
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Mode:            "realtime",
			FallbackTimeout: 30 * time.Second,
		},
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewSchedulerService(mockBlockScheduler, mockCrawlerService, cfg, logger)

	ctx := context.Background()
	
	// Mock block scheduler expectations
	mockBlockScheduler.On("Start", ctx).Return(nil)
	mockBlockScheduler.On("SubscribeNewBlocks", ctx, mock.AnythingOfType("func(*big.Int)")).Return(nil)
	mockBlockScheduler.On("Stop").Return(nil)

	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Stop the scheduler
	err = scheduler.Stop()
	assert.NoError(t, err)
	assert.False(t, scheduler.IsRunning())

	// Verify mock expectations
	mockBlockScheduler.AssertExpectations(t)
}

func TestSchedulerService_HandleNewBlock(t *testing.T) {
	mockBlockScheduler := &MockBlockSchedulerService{}
	mockCrawlerService := &MockCrawlerService{}
	
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Mode:            "realtime",
			FallbackTimeout: 30 * time.Second,
		},
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewSchedulerService(mockBlockScheduler, mockCrawlerService, cfg, logger)

	blockNumber := big.NewInt(12345)
	
	// Mock crawler service to expect ProcessSpecificBlock call
	mockCrawlerService.On("ProcessSpecificBlock", mock.AnythingOfType("*context.backgroundCtx"), blockNumber).Return(nil)

	// Call handleNewBlock directly
	scheduler.handleNewBlock(blockNumber)

	// Verify mock expectations
	mockCrawlerService.AssertExpectations(t)
}

func TestSchedulerService_GetStats(t *testing.T) {
	mockBlockScheduler := &MockBlockSchedulerService{}
	mockCrawlerService := &MockCrawlerService{}
	
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Mode:            "hybrid",
			FallbackTimeout: 30 * time.Second,
		},
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewSchedulerService(mockBlockScheduler, mockCrawlerService, cfg, logger)

	// Mock block scheduler
	mockBlockScheduler.On("IsRunning").Return(false)

	stats := scheduler.GetStats()

	assert.NotNil(t, stats)
	assert.Equal(t, "hybrid", stats["mode"])
	assert.Equal(t, false, stats["is_running"])
	assert.Equal(t, false, stats["block_scheduler_running"])
	assert.Equal(t, false, stats["polling_active"])

	// Verify mock expectations
	mockBlockScheduler.AssertExpectations(t)
}
