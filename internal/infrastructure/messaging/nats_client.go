package messaging

import (
	"context"
	"encoding/json"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// TransactionEvent represents a transaction event to be published to NATS
type TransactionEvent struct {
	Hash        string    `json:"hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Value       string    `json:"value"`
	Data        string    `json:"data"`
	BlockNumber string    `json:"block_number"`
	BlockHash   string    `json:"block_hash"`
	Timestamp   time.Time `json:"timestamp"`
	GasUsed     string    `json:"gas_used"`
	GasPrice    string    `json:"gas_price"`
	Network     string    `json:"network"`
}

// NATSClient handles NATS JetStream operations and implements MessagingService interface
type NATSClient struct {
	conn      *nats.Conn
	js        nats.JetStreamContext
	config    *config.NATSConfig
	logger    *logger.Logger
	isRunning bool
}

// NewNATSClient creates a new NATS client
func NewNATSClient(cfg *config.NATSConfig, logger *logger.Logger) *NATSClient {
	return &NATSClient{
		config: cfg,
		logger: logger.WithComponent("nats-client"),
	}
}

// NewNATSMessagingService creates a new NATS messaging service from main config
func NewNATSMessagingService(cfg *config.Config, logger *logger.Logger) *NATSClient {
	return NewNATSClient(&cfg.NATS, logger)
}

// Connect connects to NATS server and sets up JetStream
func (n *NATSClient) Connect(ctx context.Context) error {
	if !n.config.Enabled {
		n.logger.Info("NATS is disabled, skipping connection")
		return nil
	}

	n.logger.Info("Connecting to NATS server", zap.String("url", n.config.URL))

	// Connect to NATS
	opts := []nats.Option{
		nats.Name("ethereum-crawler"),
		nats.Timeout(n.config.ConnectTimeout),
		nats.ReconnectWait(n.config.ReconnectDelay),
		nats.MaxReconnects(n.config.ReconnectAttempts),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			n.logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			n.logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			n.logger.Info("NATS connection closed")
		}),
	}

	conn, err := nats.Connect(n.config.URL, opts...)
	if err != nil {
		n.logger.Error("Failed to connect to NATS", zap.Error(err))
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	n.conn = conn

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		n.logger.Error("Failed to create JetStream context", zap.Error(err))
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	n.js = js

	// Create or update stream
	if err := n.setupStream(ctx); err != nil {
		return fmt.Errorf("failed to setup stream: %w", err)
	}

	n.isRunning = true
	n.logger.Info("Successfully connected to NATS and setup JetStream")

	return nil
}

// Disconnect disconnects from NATS server
func (n *NATSClient) Disconnect() error {
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
		n.js = nil
	}
	n.isRunning = false
	n.logger.Info("Disconnected from NATS")
	return nil
}

// IsConnected checks if connected to NATS
func (n *NATSClient) IsConnected() bool {
	return n.isRunning && n.conn != nil && n.conn.IsConnected()
}

// setupStream creates or updates the JetStream stream
func (n *NATSClient) setupStream(ctx context.Context) error {
	streamName := n.config.StreamName
	subject := fmt.Sprintf("%s.events", n.config.SubjectPrefix)

	// Check if stream exists
	stream, err := n.js.StreamInfo(streamName)
	if err != nil {
		// Stream doesn't exist, create it
		n.logger.Info("Creating JetStream stream",
			zap.String("stream", streamName),
			zap.String("subject", subject))

		streamConfig := &nats.StreamConfig{
			Name:       streamName,
			Subjects:   []string{subject},
			Storage:    nats.FileStorage,
			Retention:  nats.WorkQueuePolicy,
			MaxMsgs:    1000000,            // 1M messages
			MaxBytes:   1024 * 1024 * 1024, // 1GB
			MaxAge:     24 * time.Hour,     // 24 hours
			Duplicates: 5 * time.Minute,    // Duplicate detection window
		}

		_, err = n.js.AddStream(streamConfig)
		if err != nil {
			n.logger.Error("Failed to create stream", zap.Error(err))
			return err
		}

		n.logger.Info("Successfully created JetStream stream")
	} else {
		n.logger.Info("JetStream stream already exists",
			zap.String("stream", streamName),
			zap.Uint64("messages", stream.State.Msgs))
	}

	return nil
}

// PublishTransaction publishes a transaction event to NATS JetStream
func (n *NATSClient) PublishTransaction(ctx context.Context, tx *entity.Transaction) error {
	if !n.IsConnected() {
		// If NATS is disabled or not connected, just log and return
		if !n.config.Enabled {
			return nil
		}
		return fmt.Errorf("NATS client is not connected")
	}

	// Convert transaction to event
	var toAddress string
	if tx.To != nil {
		toAddress = *tx.To
	}

	event := &TransactionEvent{
		Hash:        tx.Hash,
		From:        tx.From,
		To:          toAddress,
		Value:       tx.Value,
		Data:        tx.Data,
		BlockNumber: tx.BlockNumber,
		BlockHash:   tx.BlockHash,
		Timestamp:   time.Now(),
		GasUsed:     fmt.Sprintf("%d", tx.GasUsed),
		GasPrice:    tx.GasPrice,
		Network:     tx.Network,
	}

	// Serialize to JSON
	data, err := json.Marshal(event)
	if err != nil {
		n.logger.Error("Failed to marshal transaction event", zap.Error(err))
		return fmt.Errorf("failed to marshal transaction event: %w", err)
	}

	// Publish to JetStream
	subject := fmt.Sprintf("%s.events", n.config.SubjectPrefix)

	// Use transaction hash as message ID for deduplication
	_, err = n.js.Publish(subject, data, nats.MsgId(tx.Hash))
	if err != nil {
		n.logger.Error("Failed to publish transaction event",
			zap.String("hash", tx.Hash),
			zap.Error(err))
		return fmt.Errorf("failed to publish transaction event: %w", err)
	}

	n.logger.Debug("Published transaction event",
		zap.String("hash", tx.Hash),
		zap.String("subject", subject))

	return nil
}

// PublishTransactions publishes multiple transaction events to NATS JetStream
func (n *NATSClient) PublishTransactions(ctx context.Context, transactions []*entity.Transaction) error {
	if !n.IsConnected() {
		if !n.config.Enabled {
			return nil
		}
		return fmt.Errorf("NATS client is not connected")
	}

	if len(transactions) == 0 {
		return nil
	}

	n.logger.Debug("Publishing transaction events", zap.Int("count", len(transactions)))

	// Publish each transaction individually
	// We could batch this, but individual publishing allows for better error handling
	// and deduplication per transaction
	var errors []error
	successCount := 0

	for _, tx := range transactions {
		if err := n.PublishTransaction(ctx, tx); err != nil {
			errors = append(errors, err)
			n.logger.Error("Failed to publish transaction",
				zap.String("hash", tx.Hash),
				zap.Error(err))
		} else {
			successCount++
		}
	}

	n.logger.Info("Finished publishing transaction events",
		zap.Int("total", len(transactions)),
		zap.Int("success", successCount),
		zap.Int("errors", len(errors)))

	// Return error if any transactions failed to publish
	if len(errors) > 0 {
		return fmt.Errorf("failed to publish %d out of %d transactions", len(errors), len(transactions))
	}

	return nil
}

// GetStreamInfo returns information about the JetStream stream
func (n *NATSClient) GetStreamInfo() (interface{}, error) {
	if !n.IsConnected() {
		return nil, fmt.Errorf("NATS client is not connected")
	}

	return n.js.StreamInfo(n.config.StreamName)
}
