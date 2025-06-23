package blockchain

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/service"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketScheduler implements BlockSchedulerService using WebSocket
type WebSocketScheduler struct {
	config      *config.EthereumConfig
	logger      *logger.Logger
	conn        *websocket.Conn
	isRunning   bool
	callback    func(*big.Int)
	stopChan    chan struct{}
	mu          sync.RWMutex
	subID       string
	reconnectCh chan struct{}
}

// NewWebSocketScheduler creates a new WebSocket scheduler
func NewWebSocketScheduler(cfg *config.EthereumConfig, logger *logger.Logger) service.BlockSchedulerService {
	return &WebSocketScheduler{
		config:      cfg,
		logger:      logger.WithComponent("websocket-scheduler"),
		stopChan:    make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
	}
}

// Start starts the WebSocket scheduler
func (w *WebSocketScheduler) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	w.logger.Info("Starting WebSocket scheduler", zap.String("ws_url", w.config.WSURL))

	if err := w.connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	w.isRunning = true

	// Start connection monitor
	go w.connectionMonitor(ctx)

	return nil
}

// Stop stops the WebSocket scheduler
func (w *WebSocketScheduler) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return nil
	}

	w.logger.Info("Stopping WebSocket scheduler")

	close(w.stopChan)
	w.isRunning = false

	if w.conn != nil {
		// Send unsubscribe message if we have a subscription
		if w.subID != "" {
			w.unsubscribeFromBlocks()
		}
		w.conn.Close()
		w.conn = nil
	}

	return nil
}

// IsRunning checks if the scheduler is running
func (w *WebSocketScheduler) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isRunning
}

// SubscribeNewBlocks subscribes to new block events
func (w *WebSocketScheduler) SubscribeNewBlocks(ctx context.Context, callback func(*big.Int)) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	w.callback = callback

	// Subscribe to new heads
	if err := w.subscribeToBlocks(); err != nil {
		return fmt.Errorf("failed to subscribe to blocks: %w", err)
	}

	// Start message listener
	go w.messageListener(ctx)

	w.logger.Info("Successfully subscribed to new block events")
	return nil
}

// Unsubscribe unsubscribes from new block events
func (w *WebSocketScheduler) Unsubscribe() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.subID != "" {
		return w.unsubscribeFromBlocks()
	}

	return nil
}

// connect establishes WebSocket connection
func (w *WebSocketScheduler) connect(ctx context.Context) error {
	if w.config.WSURL == "" {
		return fmt.Errorf("WebSocket URL is not configured")
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second

	conn, _, err := dialer.DialContext(ctx, w.config.WSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	w.conn = conn
	w.logger.Info("Successfully connected to WebSocket")
	return nil
}

// subscribeToBlocks sends subscription request for new blocks
func (w *WebSocketScheduler) subscribeToBlocks() error {
	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_subscribe",
		"params":  []interface{}{"newHeads"},
	}

	if err := w.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	w.logger.Debug("Sent subscription request for new blocks")
	return nil
}

// unsubscribeFromBlocks sends unsubscribe request
func (w *WebSocketScheduler) unsubscribeFromBlocks() error {
	if w.conn == nil || w.subID == "" {
		return nil
	}

	unsubscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "eth_unsubscribe",
		"params":  []interface{}{w.subID},
	}

	if err := w.conn.WriteJSON(unsubscribeMsg); err != nil {
		w.logger.Error("Failed to send unsubscribe message", zap.Error(err))
		return err
	}

	w.subID = ""
	w.logger.Debug("Sent unsubscribe request")
	return nil
}

// messageListener listens for WebSocket messages
func (w *WebSocketScheduler) messageListener(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Message listener panic recovered", zap.Any("panic", r))
		}
	}()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			if w.conn == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			var message map[string]interface{}
			if err := w.conn.ReadJSON(&message); err != nil {
				w.logger.Error("Failed to read WebSocket message", zap.Error(err))
				// Trigger reconnection
				select {
				case w.reconnectCh <- struct{}{}:
				default:
				}
				return
			}

			w.handleMessage(message)
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (w *WebSocketScheduler) handleMessage(message map[string]interface{}) {
	// Handle subscription confirmation
	if result, ok := message["result"].(string); ok && w.subID == "" {
		w.subID = result
		w.logger.Info("Received subscription ID", zap.String("sub_id", w.subID))
		return
	}

	// Handle new block notifications
	if method, ok := message["method"].(string); ok && method == "eth_subscription" {
		params, ok := message["params"].(map[string]interface{})
		if !ok {
			return
		}

		result, ok := params["result"].(map[string]interface{})
		if !ok {
			return
		}

		// Extract block number
		if numberHex, ok := result["number"].(string); ok {
			blockNumber := new(big.Int)
			if _, success := blockNumber.SetString(numberHex[2:], 16); success {
				w.logger.Info("Received new block notification",
					zap.String("block_number", blockNumber.String()))

				// Call the callback function
				if w.callback != nil {
					go w.callback(blockNumber)
				}
			}
		}
	}
}

// connectionMonitor monitors WebSocket connection and handles reconnection
func (w *WebSocketScheduler) connectionMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		case <-w.reconnectCh:
			w.handleReconnection(ctx)
		case <-ticker.C:
			// Ping to check connection health
			if w.conn != nil {
				if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					w.logger.Warn("WebSocket ping failed, triggering reconnection", zap.Error(err))
					select {
					case w.reconnectCh <- struct{}{}:
					default:
					}
				}
			}
		}
	}
}

// handleReconnection handles WebSocket reconnection
func (w *WebSocketScheduler) handleReconnection(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return
	}

	w.logger.Warn("Attempting to reconnect WebSocket")

	// Close existing connection
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}

	// Reset subscription ID
	w.subID = ""

	// Retry connection with exponential backoff
	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		backoff := time.Duration(attempt*attempt) * time.Second
		time.Sleep(backoff)

		if err := w.connect(ctx); err != nil {
			w.logger.Error("Reconnection attempt failed",
				zap.Int("attempt", attempt),
				zap.Error(err))
			continue
		}

		// Re-subscribe to blocks
		if w.callback != nil {
			if err := w.subscribeToBlocks(); err != nil {
				w.logger.Error("Failed to re-subscribe after reconnection", zap.Error(err))
				continue
			}

			// Restart message listener
			go w.messageListener(ctx)
		}

		w.logger.Info("Successfully reconnected WebSocket")
		return
	}

	w.logger.Error("Failed to reconnect after maximum retries")
	w.isRunning = false
}
