package repository

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"math/big"
)

// BlockRepository interface for block data operations
type BlockRepository interface {
	// Create operations
	CreateBlock(ctx context.Context, block *entity.Block) error
	CreateBlocks(ctx context.Context, blocks []*entity.Block) error

	// Read operations
	GetBlockByNumber(ctx context.Context, blockNumber *big.Int) (*entity.Block, error)
	GetBlockByHash(ctx context.Context, hash string) (*entity.Block, error)
	GetBlocksInRange(ctx context.Context, startBlock, endBlock *big.Int) ([]*entity.Block, error)
	GetLastProcessedBlock(ctx context.Context, network string) (*entity.Block, error)
	GetBlocksByStatus(ctx context.Context, status entity.BlockStatus, limit int) ([]*entity.Block, error)

	// Update operations
	UpdateBlockStatus(ctx context.Context, blockHash string, status entity.BlockStatus) error
	MarkBlockAsProcessed(ctx context.Context, blockHash string) error

	// Delete operations
	DeleteBlock(ctx context.Context, blockHash string) error

	// Utility operations
	BlockExists(ctx context.Context, blockHash string) (bool, error)
	GetBlockCount(ctx context.Context, network string) (int64, error)
	GetBlockCountByStatus(ctx context.Context, status entity.BlockStatus, network string) (int64, error)
}
