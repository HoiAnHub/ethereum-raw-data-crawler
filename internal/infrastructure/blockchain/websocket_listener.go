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

// WebSocketListener implements WebSocketListenerService
type WebSocketListener struct {
	config         *config.WebSocketConfig
	ethereumConfig *config.EthereumConfig
	logger         *logger.Logger
	conn           *websocket.Conn
	isConnected    bool
	mu             sync.RWMutex

	stopChan    chan struct{}
	reconnectCh chan struct{}

	// Subscriptions
	blockCallback   func(*big.Int)
	txCallback      func(string)
	logCallback     func(interface{})
	subscriptionIDs map[string]string // subscription_type -> subscription_id

	// Connection stats
	lastMessageTime  time.Time
	messagesReceived int64
	reconnectCount   int64
	errors           int64
}

// NewWebSocketListener creates a new WebSocket listener
func NewWebSocketListener(
	wsConfig *config.WebSocketConfig,
	ethConfig *config.EthereumConfig,
	logger *logger.Logger,
) service.WebSocketListenerService {
	return &WebSocketListener{
		config:          wsConfig,
		ethereumConfig:  ethConfig,
		logger:          logger.WithComponent("websocket-listener"),
		stopChan:        make(chan struct{}),
		reconnectCh:     make(chan struct{}, 1),
		subscriptionIDs: make(map[string]string),
		lastMessageTime: time.Now(),
	}
}

// Start starts the WebSocket listener
func (w *WebSocketListener) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isConnected {
		return fmt.Errorf("websocket listener is already connected")
	}

	w.logger.Info("Starting WebSocket listener", zap.String("ws_url", w.ethereumConfig.WSURL))

	if err := w.connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	w.isConnected = true

	// Start message listener
	go w.messageListener(ctx)

	// Start connection monitor
	go w.connectionMonitor(ctx)

	return nil
}

// Stop stops the WebSocket listener
func (w *WebSocketListener) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isConnected {
		return nil
	}

	w.logger.Info("Stopping WebSocket listener")

	close(w.stopChan)
	w.isConnected = false

	if w.conn != nil {
		// Unsubscribe from all subscriptions
		for subType, subID := range w.subscriptionIDs {
			w.unsubscribe(subType, subID)
		}

		w.conn.Close()
		w.conn = nil
	}

	return nil
}

// IsConnected checks if the listener is connected
func (w *WebSocketListener) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isConnected
}

// SubscribeNewBlocks subscribes to new block events
func (w *WebSocketListener) SubscribeNewBlocks(ctx context.Context, callback func(*big.Int)) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isConnected {
		return fmt.Errorf("websocket listener is not connected")
	}

	w.blockCallback = callback

	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_subscribe",
		"params":  []interface{}{"newHeads"},
	}

	if err := w.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	w.logger.Info("Subscribed to new blocks")
	return nil
}

// SubscribePendingTransactions subscribes to pending transaction events
func (w *WebSocketListener) SubscribePendingTransactions(ctx context.Context, callback func(string)) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isConnected {
		return fmt.Errorf("websocket listener is not connected")
	}

	w.txCallback = callback

	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "eth_subscribe",
		"params":  []interface{}{"newPendingTransactions"},
	}

	if err := w.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	w.logger.Info("Subscribed to pending transactions")
	return nil
}

// SubscribeLogs subscribes to contract log events
func (w *WebSocketListener) SubscribeLogs(ctx context.Context, callback func(interface{})) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isConnected {
		return fmt.Errorf("websocket listener is not connected")
	}

	w.logCallback = callback

	// Subscribe to all logs - you can customize this filter
	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "eth_subscribe",
		"params":  []interface{}{"logs", map[string]interface{}{}},
	}

	if err := w.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	w.logger.Info("Subscribed to contract logs")
	return nil
}

// Unsubscribe unsubscribes from all subscriptions
func (w *WebSocketListener) Unsubscribe() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for subType, subID := range w.subscriptionIDs {
		w.unsubscribe(subType, subID)
	}

	w.subscriptionIDs = make(map[string]string)
	return nil
}

// GetConnectionStats returns connection statistics
func (w *WebSocketListener) GetConnectionStats() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return map[string]interface{}{
		"connected":         w.isConnected,
		"last_message_time": w.lastMessageTime,
		"messages_received": w.messagesReceived,
		"reconnect_count":   w.reconnectCount,
		"errors":            w.errors,
		"subscriptions":     len(w.subscriptionIDs),
	}
}

// connect establishes WebSocket connection
func (w *WebSocketListener) connect(ctx context.Context) error {
	if w.ethereumConfig.WSURL == "" {
		return fmt.Errorf("WebSocket URL is not configured")
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second

	conn, _, err := dialer.DialContext(ctx, w.ethereumConfig.WSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	// Set read and write timeouts
	conn.SetReadDeadline(time.Now().Add(w.config.ReadTimeout))
	conn.SetWriteDeadline(time.Now().Add(w.config.WriteTimeout))

	w.conn = conn
	w.logger.Info("Successfully connected to WebSocket")
	return nil
}

// messageListener listens for WebSocket messages
func (w *WebSocketListener) messageListener(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Message listener panic recovered", zap.Any("panic", r))
		}
		w.logger.Info("Message listener stopped")
	}()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			w.conn.SetReadDeadline(time.Now().Add(w.config.ReadTimeout))

			var message map[string]interface{}
			if err := w.conn.ReadJSON(&message); err != nil {
				w.mu.Lock()
				w.errors++
				w.mu.Unlock()

				w.logger.Error("Failed to read WebSocket message", zap.Error(err))

				// Trigger reconnection
				select {
				case w.reconnectCh <- struct{}{}:
				default:
				}
				continue
			}

			w.mu.Lock()
			w.messagesReceived++
			w.lastMessageTime = time.Now()
			w.mu.Unlock()

			w.handleMessage(message)
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (w *WebSocketListener) handleMessage(message map[string]interface{}) {
	// Handle subscription responses
	if id, ok := message["id"]; ok {
		if result, exists := message["result"]; exists {
			if subID, isString := result.(string); isString {
				// Store subscription ID based on request ID
				switch id {
				case float64(1): // newHeads subscription
					w.subscriptionIDs["blocks"] = subID
				case float64(2): // newPendingTransactions subscription
					w.subscriptionIDs["transactions"] = subID
				case float64(3): // logs subscription
					w.subscriptionIDs["logs"] = subID
				}
				w.logger.Debug("Subscription confirmed", zap.Any("id", id), zap.String("subscription_id", subID))
			}
		}
		return
	}

	// Handle subscription notifications
	if method, ok := message["method"].(string); ok && method == "eth_subscription" {
		if params, exists := message["params"].(map[string]interface{}); exists {
			if subscription, ok := params["subscription"].(string); ok {
				if result, resultExists := params["result"]; resultExists {
					w.handleSubscriptionResult(subscription, result)
				}
			}
		}
	}
}

// handleSubscriptionResult processes subscription results
func (w *WebSocketListener) handleSubscriptionResult(subscriptionID string, result interface{}) {
	// Determine subscription type
	var subType string
	for sType, sID := range w.subscriptionIDs {
		if sID == subscriptionID {
			subType = sType
			break
		}
	}

	switch subType {
	case "blocks":
		w.handleBlockResult(result)
	case "transactions":
		w.handleTransactionResult(result)
	case "logs":
		w.handleLogResult(result)
	default:
		w.logger.Debug("Unknown subscription result", zap.String("subscription_id", subscriptionID))
	}
}

// handleBlockResult processes new block results
func (w *WebSocketListener) handleBlockResult(result interface{}) {
	if w.blockCallback == nil {
		return
	}

	blockData, ok := result.(map[string]interface{})
	if !ok {
		w.logger.Error("Invalid block data format")
		return
	}

	if numberHex, exists := blockData["number"].(string); exists {
		if blockNumber, ok := new(big.Int).SetString(numberHex[2:], 16); ok {
			w.blockCallback(blockNumber)
		} else {
			w.logger.Error("Failed to parse block number", zap.String("number_hex", numberHex))
		}
	}
}

// handleTransactionResult processes pending transaction results
func (w *WebSocketListener) handleTransactionResult(result interface{}) {
	if w.txCallback == nil {
		return
	}

	if txHash, ok := result.(string); ok {
		w.txCallback(txHash)
	} else {
		w.logger.Error("Invalid transaction hash format")
	}
}

// handleLogResult processes log results
func (w *WebSocketListener) handleLogResult(result interface{}) {
	if w.logCallback == nil {
		return
	}

	w.logCallback(result)
}

// connectionMonitor monitors connection health and handles reconnections
func (w *WebSocketListener) connectionMonitor(ctx context.Context) {
	pingTicker := time.NewTicker(w.config.PingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-pingTicker.C:
			w.ping()
		case <-w.reconnectCh:
			w.handleReconnection(ctx)
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// ping sends a ping message to keep connection alive
func (w *WebSocketListener) ping() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn == nil {
		return
	}

	w.conn.SetWriteDeadline(time.Now().Add(w.config.WriteTimeout))
	if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		w.logger.Error("Failed to send ping", zap.Error(err))
		select {
		case w.reconnectCh <- struct{}{}:
		default:
		}
	}
}

// handleReconnection handles connection reconnection
func (w *WebSocketListener) handleReconnection(ctx context.Context) {
	w.mu.Lock()
	w.isConnected = false
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.reconnectCount++
	w.mu.Unlock()

	w.logger.Info("Attempting to reconnect WebSocket")

	for attempt := 0; attempt < w.config.ReconnectAttempts; attempt++ {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		case <-time.After(w.config.ReconnectDelay):
		}

		if err := w.connect(ctx); err != nil {
			w.logger.Error("Reconnection attempt failed",
				zap.Int("attempt", attempt+1),
				zap.Error(err))
			continue
		}

		w.mu.Lock()
		w.isConnected = true
		w.mu.Unlock()

		// Resubscribe to all previous subscriptions
		w.resubscribe()

		w.logger.Info("Successfully reconnected WebSocket")
		return
	}

	w.logger.Error("Failed to reconnect after all attempts")
}

// resubscribe resubscribes to all previous subscriptions after reconnection
func (w *WebSocketListener) resubscribe() {
	// Clear old subscription IDs
	w.subscriptionIDs = make(map[string]string)

	ctx := context.Background()

	// Resubscribe to blocks if callback exists
	if w.blockCallback != nil {
		w.SubscribeNewBlocks(ctx, w.blockCallback)
	}

	// Resubscribe to transactions if callback exists
	if w.txCallback != nil {
		w.SubscribePendingTransactions(ctx, w.txCallback)
	}

	// Resubscribe to logs if callback exists
	if w.logCallback != nil {
		w.SubscribeLogs(ctx, w.logCallback)
	}
}

// unsubscribe sends unsubscribe message for a specific subscription
func (w *WebSocketListener) unsubscribe(subType, subID string) error {
	if w.conn == nil || subID == "" {
		return nil
	}

	unsubscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "eth_unsubscribe",
		"params":  []interface{}{subID},
	}

	if err := w.conn.WriteJSON(unsubscribeMsg); err != nil {
		w.logger.Error("Failed to send unsubscribe message",
			zap.String("subscription_type", subType),
			zap.Error(err))
		return err
	}

	w.logger.Debug("Sent unsubscribe request", zap.String("subscription_type", subType))
	return nil
}
