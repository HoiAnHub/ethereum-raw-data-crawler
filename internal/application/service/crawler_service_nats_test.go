package service

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockMessagingService is a mock implementation of MessagingService
type MockMessagingService struct {
	mock.Mock
	connected bool
}

func (m *MockMessagingService) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.connected = true
	}
	return args.Error(0)
}

func (m *MockMessagingService) Disconnect() error {
	args := m.Called()
	m.connected = false
	return args.Error(0)
}

func (m *MockMessagingService) IsConnected() bool {
	return m.connected
}

func (m *MockMessagingService) PublishTransaction(ctx context.Context, tx *entity.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockMessagingService) PublishTransactions(ctx context.Context, transactions []*entity.Transaction) error {
	args := m.Called(ctx, transactions)
	return args.Error(0)
}

func (m *MockMessagingService) GetStreamInfo() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

// MockBlockchainService for testing
type MockBlockchainService struct {
	mock.Mock
}

func (m *MockBlockchainService) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBlockchainService) Disconnect() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBlockchainService) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockBlockchainService) GetLatestBlockNumber(ctx context.Context) (*big.Int, error) {
	args := m.Called(ctx)
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockBlockchainService) GetBlockByNumber(ctx context.Context, blockNumber *big.Int) (*entity.Block, error) {
	args := m.Called(ctx, blockNumber)
	return args.Get(0).(*entity.Block), args.Error(1)
}

func (m *MockBlockchainService) GetBlockByHash(ctx context.Context, blockHash string) (*entity.Block, error) {
	args := m.Called(ctx, blockHash)
	return args.Get(0).(*entity.Block), args.Error(1)
}

func (m *MockBlockchainService) GetTransactionByHash(ctx context.Context, txHash string) (*entity.Transaction, error) {
	args := m.Called(ctx, txHash)
	return args.Get(0).(*entity.Transaction), args.Error(1)
}

func (m *MockBlockchainService) GetTransactionReceipt(ctx context.Context, txHash string) (*entity.Transaction, error) {
	args := m.Called(ctx, txHash)
	return args.Get(0).(*entity.Transaction), args.Error(1)
}

func (m *MockBlockchainService) GetTransactionsByBlock(ctx context.Context, blockNumber *big.Int) ([]*entity.Transaction, error) {
	args := m.Called(ctx, blockNumber)
	return args.Get(0).([]*entity.Transaction), args.Error(1)
}

func (m *MockBlockchainService) GetBlocksInRange(ctx context.Context, startBlock, endBlock *big.Int) ([]*entity.Block, error) {
	args := m.Called(ctx, startBlock, endBlock)
	return args.Get(0).([]*entity.Block), args.Error(1)
}

func (m *MockBlockchainService) GetNetworkID(ctx context.Context) (*big.Int, error) {
	args := m.Called(ctx)
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockBlockchainService) GetGasPrice(ctx context.Context) (*big.Int, error) {
	args := m.Called(ctx)
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockBlockchainService) GetPeerCount(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockBlockchainService) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockTransactionRepository for testing
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) CreateTransaction(ctx context.Context, tx *entity.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) CreateTransactions(ctx context.Context, txs []*entity.Transaction) error {
	args := m.Called(ctx, txs)
	return args.Error(0)
}

func (m *MockTransactionRepository) UpsertTransactions(ctx context.Context, txs []*entity.Transaction) error {
	args := m.Called(ctx, txs)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTransactionByHash(ctx context.Context, hash string) (*entity.Transaction, error) {
	args := m.Called(ctx, hash)
	return args.Get(0).(*entity.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetTransactionsByBlockNumber(ctx context.Context, blockNumber *big.Int) ([]*entity.Transaction, error) {
	args := m.Called(ctx, blockNumber)
	return args.Get(0).([]*entity.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetTransactionsByAddress(ctx context.Context, address string, limit int, offset int) ([]*entity.Transaction, error) {
	args := m.Called(ctx, address, limit, offset)
	return args.Get(0).([]*entity.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetTransactionCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func TestCrawlerService_PublishTransactions_Success(t *testing.T) {
	// Setup mocks
	mockMessagingService := &MockMessagingService{}

	// Setup config
	cfg := &config.Config{
		Crawler: config.CrawlerConfig{
			UseUpsert: true,
		},
	}

	// Setup logger
	logger, _ := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	// Create test transactions
	transactions := createTestTransactions()

	// Setup expectations
	mockMessagingService.connected = true
	mockMessagingService.On("PublishTransactions", mock.Anything, transactions).Return(nil)

	// Test publishTransactions method directly
	ctx := context.Background()

	// Create a minimal service just for testing publishTransactions
	service := &CrawlerService{
		messagingService: mockMessagingService,
		config:           cfg,
		logger:           logger,
	}

	err := service.publishTransactions(ctx, transactions, logger)

	// Assertions
	assert.NoError(t, err)
	mockMessagingService.AssertExpectations(t)
}

func TestCrawlerService_PublishTransactions_MessagingFailure(t *testing.T) {
	// Setup mocks
	mockMessagingService := &MockMessagingService{}

	// Setup config
	cfg := &config.Config{
		Crawler: config.CrawlerConfig{
			UseUpsert: true,
		},
	}

	// Setup logger
	logger, _ := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	// Create test transactions
	transactions := createTestTransactions()

	// Setup expectations - messaging fails
	mockMessagingService.connected = true
	mockMessagingService.On("PublishTransactions", mock.Anything, transactions).Return(assert.AnError)

	// Test publishTransactions method directly
	ctx := context.Background()

	// Create a minimal service just for testing publishTransactions
	service := &CrawlerService{
		messagingService: mockMessagingService,
		config:           cfg,
		logger:           logger,
	}

	err := service.publishTransactions(ctx, transactions, logger)

	// Assertions - should return error when messaging fails
	assert.Error(t, err)
	mockMessagingService.AssertExpectations(t)
}

func TestCrawlerService_PublishTransactions_NotConnected(t *testing.T) {
	// Setup mocks
	mockMessagingService := &MockMessagingService{}

	// Setup config
	cfg := &config.Config{
		Crawler: config.CrawlerConfig{
			UseUpsert: true,
		},
	}

	// Setup logger
	logger, _ := logger.NewLogger(&config.Config{
		App: config.AppConfig{LogLevel: "info"},
	})

	// Create test transactions
	transactions := createTestTransactions()

	// Setup expectations - messaging service not connected
	mockMessagingService.connected = false

	// Test publishTransactions method directly
	ctx := context.Background()

	// Create a minimal service just for testing publishTransactions
	service := &CrawlerService{
		messagingService: mockMessagingService,
		config:           cfg,
		logger:           logger,
	}

	err := service.publishTransactions(ctx, transactions, logger)

	// Assertions - should succeed even if messaging not connected
	assert.NoError(t, err)
	// Messaging service should not be called
	mockMessagingService.AssertNotCalled(t, "PublishTransactions")
}

// Helper function to create test transactions
func createTestTransactions() []*entity.Transaction {
	toAddress := "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6"
	return []*entity.Transaction{
		{
			ID:                primitive.NewObjectID(),
			Hash:              "0x1234567890abcdef",
			BlockHash:         "0xabcdef1234567890",
			BlockNumber:       "12345",
			TransactionIndex:  0,
			From:              "0x1234567890123456789012345678901234567890",
			To:                &toAddress,
			Value:             "1000000000000000000",
			Gas:               21000,
			GasPrice:          "20000000000",
			GasUsed:           21000,
			CumulativeGasUsed: 21000,
			Data:              "0x",
			Nonce:             1,
			Status:            1,
			CrawledAt:         time.Now(),
			Network:           "mainnet",
			TxStatus:          entity.TransactionStatusProcessed,
		},
	}
}
