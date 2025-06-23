package service

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"math/big"
)

// BlockchainService interface for blockchain interactions
type BlockchainService interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool

	// Block operations
	GetLatestBlockNumber(ctx context.Context) (*big.Int, error)
	GetBlockByNumber(ctx context.Context, blockNumber *big.Int) (*entity.Block, error)
	GetBlockByHash(ctx context.Context, blockHash string) (*entity.Block, error)

	// Transaction operations
	GetTransactionByHash(ctx context.Context, txHash string) (*entity.Transaction, error)
	GetTransactionReceipt(ctx context.Context, txHash string) (*entity.Transaction, error)
	GetTransactionsByBlock(ctx context.Context, blockNumber *big.Int) ([]*entity.Transaction, error)

	// Batch operations
	GetBlocksInRange(ctx context.Context, startBlock, endBlock *big.Int) ([]*entity.Block, error)

	// Network information
	GetNetworkID(ctx context.Context) (*big.Int, error)
	GetGasPrice(ctx context.Context) (*big.Int, error)
	GetPeerCount(ctx context.Context) (uint64, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// BlockSchedulerService defines the interface for real-time block scheduling
type BlockSchedulerService interface {
	// Start the scheduler to listen for new blocks
	Start(ctx context.Context) error

	// Stop the scheduler
	Stop() error

	// Check if scheduler is running
	IsRunning() bool

	// Subscribe to new block events
	SubscribeNewBlocks(ctx context.Context, callback func(*big.Int)) error

	// Unsubscribe from new block events
	Unsubscribe() error
}
