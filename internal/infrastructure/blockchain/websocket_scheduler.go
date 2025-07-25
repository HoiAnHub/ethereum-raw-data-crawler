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
	config          *config.EthereumConfig
	logger          *logger.Logger
	conn            *websocket.Conn
	isRunning       bool
	callback        func(*big.Int)
	stopChan        chan struct{}
	mu              sync.RWMutex
	subID           string
	reconnectCh     chan struct{}
	lastMessageTime time.Time
	schedulerCtx    context.Context    // Context riêng cho scheduler
	schedulerCancel context.CancelFunc // Cancel function cho scheduler context
}

// NewWebSocketScheduler creates a new WebSocket scheduler
func NewWebSocketScheduler(cfg *config.EthereumConfig, logger *logger.Logger) service.BlockSchedulerService {
	return &WebSocketScheduler{
		config:          cfg,
		logger:          logger.WithComponent("websocket-scheduler"),
		stopChan:        make(chan struct{}),
		reconnectCh:     make(chan struct{}, 1),
		lastMessageTime: time.Now(),
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

	// Tạo context riêng cho scheduler, không phụ thuộc vào context từ bên ngoài
	w.schedulerCtx, w.schedulerCancel = context.WithCancel(context.Background())

	if err := w.connect(ctx); err != nil {
		w.schedulerCancel() // Cancel context nếu connect thất bại
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	w.isRunning = true

	// Start connection monitor với scheduler context
	go w.connectionMonitor(w.schedulerCtx)

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

	// Cancel scheduler context trước
	if w.schedulerCancel != nil {
		w.schedulerCancel()
	}

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

	// Start message listener với scheduler context thay vì context từ tham số
	go w.messageListener(w.schedulerCtx)

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
		w.logger.Info("Message listener stopped")

		// Auto-restart message listener if scheduler is still running
		w.mu.RLock()
		running := w.isRunning
		schedulerCtx := w.schedulerCtx
		w.mu.RUnlock()

		if running && schedulerCtx != nil {
			w.logger.Info("Scheduler still running, attempting to restart message listener")
			// Wait a bit before restarting to avoid tight restart loops
			time.Sleep(2 * time.Second)

			select {
			case <-w.stopChan:
				w.logger.Info("Stop signal received during restart attempt")
				return
			case <-schedulerCtx.Done():
				w.logger.Info("Scheduler context cancelled during restart attempt")
				return
			default:
				w.logger.Info("Restarting message listener")
				// Sử dụng scheduler context thay vì context bị cancel
				go w.messageListener(schedulerCtx)
			}
		}
	}()

	w.logger.Info("Message listener started")

	for {
		select {
		case <-w.stopChan:
			w.logger.Info("Message listener received stop signal")
			return
		case <-ctx.Done():
			w.logger.Info("Message listener context cancelled")
			// Kiểm tra xem có phải scheduler context bị cancel không
			w.mu.RLock()
			schedulerCtx := w.schedulerCtx
			w.mu.RUnlock()

			if schedulerCtx != nil && schedulerCtx == ctx {
				// Nếu là scheduler context bị cancel thì dừng hẳn
				w.logger.Info("Scheduler context cancelled, stopping message listener permanently")
				return
			} else {
				// Nếu chỉ là context từ bên ngoài bị cancel, tiếp tục với scheduler context
				w.logger.Info("External context cancelled, but scheduler still running - continuing with scheduler context")
				return // Defer function sẽ restart với scheduler context
			}
		default:
			w.mu.RLock()
			conn := w.conn
			running := w.isRunning
			w.mu.RUnlock()

			if !running {
				w.logger.Debug("Scheduler not running, stopping message listener")
				return
			}

			if conn == nil {
				w.logger.Debug("No WebSocket connection, waiting...")
				time.Sleep(1 * time.Second)
				continue
			}

			// Set read deadline to prevent hanging
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			var message map[string]interface{}
			if err := conn.ReadJSON(&message); err != nil {
				// Check if it's a normal close or timeout
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					w.logger.Warn("WebSocket connection closed", zap.Error(err))
				} else if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					w.logger.Debug("WebSocket read timeout, continuing...")
					continue
				} else {
					w.logger.Error("Failed to read WebSocket message", zap.Error(err))
				}

				// Trigger reconnection for any error
				select {
				case w.reconnectCh <- struct{}{}:
					w.logger.Info("Triggered WebSocket reconnection")
				default:
					w.logger.Debug("Reconnection already in progress")
				}
				return
			}

			w.logger.Debug("Received WebSocket message", zap.Any("message_type", message["method"]))
			w.handleMessage(message)
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (w *WebSocketScheduler) handleMessage(message map[string]interface{}) {
	// Update last message time
	w.mu.Lock()
	w.lastMessageTime = time.Now()
	w.mu.Unlock()

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
			w.logger.Debug("Invalid params in eth_subscription message")
			return
		}

		result, ok := params["result"].(map[string]interface{})
		if !ok {
			w.logger.Debug("Invalid result in eth_subscription params")
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
			} else {
				w.logger.Error("Failed to parse block number", zap.String("number_hex", numberHex))
			}
		} else {
			w.logger.Debug("No block number in subscription result")
		}
	}
}

// connectionMonitor monitors WebSocket connection and handles reconnection
func (w *WebSocketScheduler) connectionMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Retry ticker for failed reconnections
	retryTicker := time.NewTicker(60 * time.Second)
	defer retryTicker.Stop()

	w.logger.Info("Connection monitor started")

	for {
		select {
		case <-w.stopChan:
			w.logger.Info("Connection monitor received stop signal")
			return
		case <-ctx.Done():
			w.logger.Info("Connection monitor context cancelled")
			return
		case <-w.reconnectCh:
			w.logger.Info("Connection monitor handling reconnection request")
			// Sử dụng scheduler context cho reconnection
			w.mu.RLock()
			schedulerCtx := w.schedulerCtx
			w.mu.RUnlock()
			if schedulerCtx != nil {
				w.handleReconnection(schedulerCtx)
			}
		case <-retryTicker.C:
			// Check if we need to retry connection
			w.mu.RLock()
			running := w.isRunning
			conn := w.conn
			schedulerCtx := w.schedulerCtx
			w.mu.RUnlock()

			if running && conn == nil && schedulerCtx != nil {
				w.logger.Info("No active connection, attempting reconnection")
				w.handleReconnection(schedulerCtx)
			}
		case <-ticker.C:
			// Check connection health
			w.mu.RLock()
			conn := w.conn
			running := w.isRunning
			lastMessageTime := w.lastMessageTime
			w.mu.RUnlock()

			if running && conn != nil {
				// Check if we haven't received messages for too long
				timeSinceLastMessage := time.Since(lastMessageTime)
				if timeSinceLastMessage > 2*time.Minute {
					w.logger.Warn("No messages received for too long, triggering reconnection",
						zap.Duration("time_since_last_message", timeSinceLastMessage))
					select {
					case w.reconnectCh <- struct{}{}:
						w.logger.Info("Triggered reconnection due to message timeout")
					default:
						w.logger.Debug("Reconnection already in progress")
					}
					continue
				}

				// Send ping to check connection health
				w.logger.Debug("Sending WebSocket ping")
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					w.logger.Warn("WebSocket ping failed, triggering reconnection", zap.Error(err))
					select {
					case w.reconnectCh <- struct{}{}:
						w.logger.Info("Triggered reconnection due to ping failure")
					default:
						w.logger.Debug("Reconnection already in progress")
					}
				} else {
					w.logger.Debug("WebSocket ping successful")
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
		w.logger.Debug("Scheduler not running, skipping reconnection")
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
	maxRetries := 10 // Increased retries
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if !w.isRunning {
			w.logger.Info("Scheduler stopped during reconnection")
			return
		}

		backoff := time.Duration(attempt) * 2 * time.Second // Linear backoff
		if backoff > 30*time.Second {
			backoff = 30 * time.Second // Cap at 30 seconds
		}

		w.logger.Info("Reconnection attempt",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Duration("backoff", backoff))

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
				// Close the connection and try again
				if w.conn != nil {
					w.conn.Close()
					w.conn = nil
				}
				continue
			}

			// Note: Message listener will be automatically restarted by its own defer logic
			w.logger.Info("WebSocket reconnection completed, message listener will restart automatically")
		}

		w.logger.Info("Successfully reconnected WebSocket", zap.Int("attempt", attempt))
		return
	}

	w.logger.Error("Failed to reconnect after maximum retries, scheduler will continue trying...")
	// Don't set isRunning to false, keep trying in background
}
