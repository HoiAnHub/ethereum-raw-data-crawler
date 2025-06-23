package secondary

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/domain/repository"
	"ethereum-raw-data-crawler/internal/infrastructure/database"
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BlockRepositoryImpl implements BlockRepository interface
type BlockRepositoryImpl struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

// retryOperation executes an operation with retry logic for MongoDB connection issues
func (r *BlockRepositoryImpl) retryOperation(ctx context.Context, operation func() error) error {
	maxRetries := 3
	baseDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		// Check if it's a connection-related error
		if r.isConnectionError(err) && attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(attempt+1)
			time.Sleep(delay)
			continue
		}

		return err
	}

	return nil
}

// isConnectionError checks if the error is related to MongoDB connection issues
func (r *BlockRepositoryImpl) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	connectionErrors := []string{
		"connection",
		"network",
		"timeout",
		"server selection",
		"no reachable servers",
		"socket",
		"broken pipe",
		"connection reset",
	}

	for _, connErr := range connectionErrors {
		if len(errStr) > 0 && len(connErr) > 0 {
			// Simple contains check without strings package
			if containsSubstring(errStr, connErr) {
				return true
			}
		}
	}

	return false
}

// containsSubstring checks if s contains substr (simple implementation)
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// NewBlockRepository creates new block repository
func NewBlockRepository(db *database.MongoDB) repository.BlockRepository {
	return &BlockRepositoryImpl{
		db:         db,
		collection: db.GetCollection("blocks"),
	}
}

// CreateBlock creates a new block with retry logic
func (r *BlockRepositoryImpl) CreateBlock(ctx context.Context, block *entity.Block) error {
	return r.retryOperation(ctx, func() error {
		block.ID = primitive.NewObjectID()
		_, err := r.collection.InsertOne(ctx, block)
		return err
	})
}

// CreateBlocks creates multiple blocks
func (r *BlockRepositoryImpl) CreateBlocks(ctx context.Context, blocks []*entity.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	documents := make([]interface{}, len(blocks))
	for i, block := range blocks {
		block.ID = primitive.NewObjectID()
		documents[i] = block
	}

	_, err := r.collection.InsertMany(ctx, documents)
	return err
}

// GetBlockByNumber gets block by number
func (r *BlockRepositoryImpl) GetBlockByNumber(ctx context.Context, blockNumber *big.Int) (*entity.Block, error) {
	filter := bson.M{"number": blockNumber.String()}

	var block entity.Block
	err := r.collection.FindOne(ctx, filter).Decode(&block)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &block, nil
}

// GetBlockByHash gets block by hash
func (r *BlockRepositoryImpl) GetBlockByHash(ctx context.Context, hash string) (*entity.Block, error) {
	filter := bson.M{"hash": hash}

	var block entity.Block
	err := r.collection.FindOne(ctx, filter).Decode(&block)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &block, nil
}

// GetBlocksInRange gets blocks in range
func (r *BlockRepositoryImpl) GetBlocksInRange(ctx context.Context, startBlock, endBlock *big.Int) ([]*entity.Block, error) {
	filter := bson.M{
		"number": bson.M{
			"$gte": startBlock.String(),
			"$lte": endBlock.String(),
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "number", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var blocks []*entity.Block
	for cursor.Next(ctx) {
		var block entity.Block
		if err := cursor.Decode(&block); err != nil {
			return nil, err
		}
		blocks = append(blocks, &block)
	}

	return blocks, cursor.Err()
}

// GetLastProcessedBlock gets last processed block
func (r *BlockRepositoryImpl) GetLastProcessedBlock(ctx context.Context, network string) (*entity.Block, error) {
	filter := bson.M{
		"network": network,
		"status":  entity.BlockStatusProcessed,
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "number", Value: -1}})

	var block entity.Block
	err := r.collection.FindOne(ctx, filter, opts).Decode(&block)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &block, nil
}

// GetBlocksByStatus gets blocks by status
func (r *BlockRepositoryImpl) GetBlocksByStatus(ctx context.Context, status entity.BlockStatus, limit int) ([]*entity.Block, error) {
	filter := bson.M{"status": status}
	opts := options.Find().
		SetSort(bson.D{{Key: "number", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var blocks []*entity.Block
	for cursor.Next(ctx) {
		var block entity.Block
		if err := cursor.Decode(&block); err != nil {
			return nil, err
		}
		blocks = append(blocks, &block)
	}

	return blocks, cursor.Err()
}

// UpdateBlockStatus updates block status
func (r *BlockRepositoryImpl) UpdateBlockStatus(ctx context.Context, blockHash string, status entity.BlockStatus) error {
	filter := bson.M{"hash": blockHash}
	update := bson.M{"$set": bson.M{"status": status}}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// MarkBlockAsProcessed marks block as processed
func (r *BlockRepositoryImpl) MarkBlockAsProcessed(ctx context.Context, blockHash string) error {
	filter := bson.M{"hash": blockHash}
	update := bson.M{
		"$set": bson.M{
			"status":       entity.BlockStatusProcessed,
			"processed_at": primitive.NewDateTimeFromTime(time.Now()),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// DeleteBlock deletes block
func (r *BlockRepositoryImpl) DeleteBlock(ctx context.Context, blockHash string) error {
	filter := bson.M{"hash": blockHash}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// BlockExists checks if block exists
func (r *BlockRepositoryImpl) BlockExists(ctx context.Context, blockHash string) (bool, error) {
	filter := bson.M{"hash": blockHash}
	count, err := r.collection.CountDocuments(ctx, filter)
	return count > 0, err
}

// GetBlockCount gets total block count
func (r *BlockRepositoryImpl) GetBlockCount(ctx context.Context, network string) (int64, error) {
	filter := bson.M{"network": network}
	return r.collection.CountDocuments(ctx, filter)
}

// GetBlockCountByStatus gets block count by status
func (r *BlockRepositoryImpl) GetBlockCountByStatus(ctx context.Context, status entity.BlockStatus, network string) (int64, error) {
	filter := bson.M{
		"network": network,
		"status":  status,
	}
	return r.collection.CountDocuments(ctx, filter)
}
