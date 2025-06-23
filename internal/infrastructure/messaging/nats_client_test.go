package messaging

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockNATSClient is a mock implementation of MessagingService for testing
type MockNATSClient struct {
	mock.Mock
	connected bool
}

func (m *MockNATSClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.connected = true
	}
	return args.Error(0)
}

func (m *MockNATSClient) Disconnect() error {
	args := m.Called()
	m.connected = false
	return args.Error(0)
}

func (m *MockNATSClient) IsConnected() bool {
	return m.connected
}

func (m *MockNATSClient) PublishTransaction(ctx context.Context, tx *entity.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockNATSClient) PublishTransactions(ctx context.Context, transactions []*entity.Transaction) error {
	args := m.Called(ctx, transactions)
	return args.Error(0)
}

func (m *MockNATSClient) GetStreamInfo() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func TestNewNATSClient(t *testing.T) {
	cfg := &config.NATSConfig{
		URL:                "nats://localhost:4222",
		StreamName:         "TEST_STREAM",
		SubjectPrefix:      "test",
		ConnectTimeout:     10 * time.Second,
		ReconnectAttempts:  5,
		ReconnectDelay:     2 * time.Second,
		MaxPendingMessages: 1000,
		Enabled:            true,
	}

	logger, _ := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	client := NewNATSClient(cfg, logger)

	assert.NotNil(t, client)
	assert.Equal(t, cfg, client.config)
	assert.NotNil(t, client.logger)
	assert.False(t, client.isRunning)
}

func TestNewNATSMessagingService(t *testing.T) {
	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:                "nats://localhost:4222",
			StreamName:         "TEST_STREAM",
			SubjectPrefix:      "test",
			ConnectTimeout:     10 * time.Second,
			ReconnectAttempts:  5,
			ReconnectDelay:     2 * time.Second,
			MaxPendingMessages: 1000,
			Enabled:            true,
		},
	}

	logger, _ := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	client := NewNATSMessagingService(cfg, logger)

	assert.NotNil(t, client)
	assert.Equal(t, &cfg.NATS, client.config)
	assert.NotNil(t, client.logger)
	assert.False(t, client.isRunning)
}

func TestNATSClient_DisabledConfig(t *testing.T) {
	cfg := &config.NATSConfig{
		URL:     "nats://localhost:4222",
		Enabled: false, // Disabled
	}

	logger, _ := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	client := NewNATSClient(cfg, logger)
	ctx := context.Background()

	// Connect should succeed but do nothing when disabled
	err := client.Connect(ctx)
	assert.NoError(t, err)
	assert.False(t, client.IsConnected())

	// PublishTransaction should succeed but do nothing when disabled
	tx := createTestTransaction()
	err = client.PublishTransaction(ctx, tx)
	assert.NoError(t, err)

	// PublishTransactions should succeed but do nothing when disabled
	transactions := []*entity.Transaction{tx}
	err = client.PublishTransactions(ctx, transactions)
	assert.NoError(t, err)
}

func TestNATSClient_NotConnected(t *testing.T) {
	cfg := &config.NATSConfig{
		URL:     "nats://localhost:4222",
		Enabled: true,
	}

	logger, _ := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	client := NewNATSClient(cfg, logger)
	ctx := context.Background()

	// Should not be connected initially
	assert.False(t, client.IsConnected())

	// PublishTransaction should fail when not connected
	tx := createTestTransaction()
	err := client.PublishTransaction(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	// PublishTransactions should fail when not connected
	transactions := []*entity.Transaction{tx}
	err = client.PublishTransactions(ctx, transactions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	// GetStreamInfo should fail when not connected
	_, err = client.GetStreamInfo()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestMockNATSClient(t *testing.T) {
	mockClient := &MockNATSClient{}
	ctx := context.Background()

	// Test Connect
	mockClient.On("Connect", ctx).Return(nil)
	err := mockClient.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, mockClient.IsConnected())

	// Test PublishTransaction
	tx := createTestTransaction()
	mockClient.On("PublishTransaction", ctx, tx).Return(nil)
	err = mockClient.PublishTransaction(ctx, tx)
	assert.NoError(t, err)

	// Test PublishTransactions
	transactions := []*entity.Transaction{tx}
	mockClient.On("PublishTransactions", ctx, transactions).Return(nil)
	err = mockClient.PublishTransactions(ctx, transactions)
	assert.NoError(t, err)

	// Test Disconnect
	mockClient.On("Disconnect").Return(nil)
	err = mockClient.Disconnect()
	assert.NoError(t, err)
	assert.False(t, mockClient.IsConnected())

	// Verify all expectations
	mockClient.AssertExpectations(t)
}

// Helper function to create a test transaction
func createTestTransaction() *entity.Transaction {
	toAddress := "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6"
	return &entity.Transaction{
		ID:                primitive.NewObjectID(),
		Hash:              "0x1234567890abcdef",
		BlockHash:         "0xabcdef1234567890",
		BlockNumber:       "12345",
		TransactionIndex:  0,
		From:              "0x1234567890123456789012345678901234567890",
		To:                &toAddress,
		Value:             "1000000000000000000", // 1 ETH in wei
		Gas:               21000,
		GasPrice:          "20000000000", // 20 Gwei
		GasUsed:           21000,
		CumulativeGasUsed: 21000,
		Data:              "0x",
		Nonce:             1,
		Status:            1,
		CrawledAt:         time.Now(),
		Network:           "mainnet",
		TxStatus:          entity.TransactionStatusProcessed,
	}
}
