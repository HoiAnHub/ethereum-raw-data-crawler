package repository

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"math/big"
)

// TransactionRepository interface for transaction data operations
type TransactionRepository interface {
	// Create operations
	CreateTransaction(ctx context.Context, tx *entity.Transaction) error
	CreateTransactions(ctx context.Context, txs []*entity.Transaction) error

	// Read operations
	GetTransactionByHash(ctx context.Context, hash string) (*entity.Transaction, error)
	GetTransactionsByBlockHash(ctx context.Context, blockHash string) ([]*entity.Transaction, error)
	GetTransactionsByBlockNumber(ctx context.Context, blockNumber *big.Int) ([]*entity.Transaction, error)
	GetTransactionsByAddress(ctx context.Context, address string, limit int, offset int) ([]*entity.Transaction, error)
	GetTransactionsByStatus(ctx context.Context, status entity.TransactionStatus, limit int) ([]*entity.Transaction, error)
	GetTransactionsByTimeRange(ctx context.Context, startTime, endTime *big.Int) ([]*entity.Transaction, error)

	// Update operations
	UpdateTransactionStatus(ctx context.Context, hash string, status entity.TransactionStatus) error
	MarkTransactionAsProcessed(ctx context.Context, hash string) error

	// Delete operations
	DeleteTransaction(ctx context.Context, hash string) error
	DeleteTransactionsByBlockHash(ctx context.Context, blockHash string) error

	// Utility operations
	TransactionExists(ctx context.Context, hash string) (bool, error)
	GetTransactionCount(ctx context.Context, network string) (int64, error)
	GetTransactionCountByStatus(ctx context.Context, status entity.TransactionStatus, network string) (int64, error)
	GetTransactionCountByAddress(ctx context.Context, address string) (int64, error)

	// Analytics operations
	GetTransactionVolumeByTimeRange(ctx context.Context, startTime, endTime *big.Int) (*big.Int, error)
	GetTopTransactionsByValue(ctx context.Context, limit int) ([]*entity.Transaction, error)
}
