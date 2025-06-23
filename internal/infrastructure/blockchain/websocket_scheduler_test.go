package blockchain

import (
	"context"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWebSocketScheduler(t *testing.T) {
	cfg := &config.EthereumConfig{
		WSURL: "wss://mainnet.infura.io/ws/v3/test",
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger)

	assert.NotNil(t, scheduler)
	assert.False(t, scheduler.IsRunning())
}

func TestWebSocketScheduler_StartStop(t *testing.T) {
	cfg := &config.EthereumConfig{
		WSURL: "", // Empty URL to test error case
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger)
	ctx := context.Background()

	// Test start with empty URL (should fail)
	err := scheduler.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WebSocket URL is not configured")
	assert.False(t, scheduler.IsRunning())

	// Test stop when not running
	err = scheduler.Stop()
	assert.NoError(t, err)
}

func TestWebSocketScheduler_DoubleStart(t *testing.T) {
	cfg := &config.EthereumConfig{
		WSURL: "wss://mainnet.infura.io/ws/v3/test",
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger).(*WebSocketScheduler)
	
	// Manually set running to test double start
	scheduler.isRunning = true

	ctx := context.Background()
	err := scheduler.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheduler is already running")
}

func TestWebSocketScheduler_SubscribeWithoutStart(t *testing.T) {
	cfg := &config.EthereumConfig{
		WSURL: "wss://mainnet.infura.io/ws/v3/test",
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger)
	ctx := context.Background()

	// Test subscribe without starting
	callback := func(blockNumber *big.Int) {
		// Mock callback
	}

	err := scheduler.SubscribeNewBlocks(ctx, callback)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheduler is not running")
}

func TestWebSocketScheduler_HandleMessage(t *testing.T) {
	cfg := &config.EthereumConfig{
		WSURL: "wss://mainnet.infura.io/ws/v3/test",
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger).(*WebSocketScheduler)

	// Test subscription confirmation message
	subscriptionMessage := map[string]interface{}{
		"result": "0x123456",
	}

	scheduler.handleMessage(subscriptionMessage)
	assert.Equal(t, "0x123456", scheduler.subID)

	// Test new block notification
	var receivedBlockNumber *big.Int
	scheduler.callback = func(blockNumber *big.Int) {
		receivedBlockNumber = blockNumber
	}

	blockMessage := map[string]interface{}{
		"method": "eth_subscription",
		"params": map[string]interface{}{
			"result": map[string]interface{}{
				"number": "0x1234", // Block number 4660 in hex
			},
		},
	}

	scheduler.handleMessage(blockMessage)
	
	// Give some time for the goroutine to execute
	time.Sleep(10 * time.Millisecond)
	
	assert.NotNil(t, receivedBlockNumber)
	assert.Equal(t, int64(4660), receivedBlockNumber.Int64())
}

func TestWebSocketScheduler_HandleInvalidMessage(t *testing.T) {
	cfg := &config.EthereumConfig{
		WSURL: "wss://mainnet.infura.io/ws/v3/test",
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger).(*WebSocketScheduler)

	// Test invalid message format
	invalidMessage := map[string]interface{}{
		"method": "eth_subscription",
		"params": "invalid_params", // Should be a map, not string
	}

	// This should not panic or cause errors
	scheduler.handleMessage(invalidMessage)

	// Test message with invalid block number
	invalidBlockMessage := map[string]interface{}{
		"method": "eth_subscription",
		"params": map[string]interface{}{
			"result": map[string]interface{}{
				"number": "invalid_hex", // Invalid hex number
			},
		},
	}

	scheduler.handleMessage(invalidBlockMessage)
}

func TestWebSocketScheduler_Unsubscribe(t *testing.T) {
	cfg := &config.EthereumConfig{
		WSURL: "wss://mainnet.infura.io/ws/v3/test",
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger).(*WebSocketScheduler)

	// Test unsubscribe without subscription ID
	err := scheduler.Unsubscribe()
	assert.NoError(t, err)

	// Test unsubscribe with subscription ID but no connection
	scheduler.subID = "test_sub_id"
	err = scheduler.Unsubscribe()
	assert.NoError(t, err)
}

// Integration test that would require a real WebSocket server
// This is commented out as it requires external dependencies
/*
func TestWebSocketScheduler_Integration(t *testing.T) {
	// This test would require a mock WebSocket server
	// or connection to a real Ethereum node
	t.Skip("Integration test requires WebSocket server")
	
	cfg := &config.EthereumConfig{
		WSURL: "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID",
	}
	
	logger := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	scheduler := NewWebSocketScheduler(cfg, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start scheduler
	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Subscribe to new blocks
	var receivedBlocks []*big.Int
	callback := func(blockNumber *big.Int) {
		receivedBlocks = append(receivedBlocks, blockNumber)
	}

	err = scheduler.SubscribeNewBlocks(ctx, callback)
	assert.NoError(t, err)

	// Wait for some blocks
	time.Sleep(10 * time.Second)

	// Stop scheduler
	err = scheduler.Stop()
	assert.NoError(t, err)
	assert.False(t, scheduler.IsRunning())

	// Check if we received any blocks
	assert.Greater(t, len(receivedBlocks), 0)
}
*/
