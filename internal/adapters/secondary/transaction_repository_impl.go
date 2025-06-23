package secondary

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/domain/repository"
	"ethereum-raw-data-crawler/internal/infrastructure/database"
	"math/big"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TransactionRepositoryImpl implements TransactionRepository interface
type TransactionRepositoryImpl struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

// retryOperation executes an operation with retry logic for MongoDB connection issues
func (r *TransactionRepositoryImpl) retryOperation(ctx context.Context, operation func() error) error {
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
func (r *TransactionRepositoryImpl) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
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
		if strings.Contains(errStr, connErr) {
			return true
		}
	}

	return false
}

// NewTransactionRepository creates new transaction repository
func NewTransactionRepository(db *database.MongoDB) repository.TransactionRepository {
	return &TransactionRepositoryImpl{
		db:         db,
		collection: db.GetCollection("transactions"),
	}
}

// CreateTransaction creates a new transaction
func (r *TransactionRepositoryImpl) CreateTransaction(ctx context.Context, tx *entity.Transaction) error {
	tx.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, tx)
	return err
}

// CreateTransactions creates multiple transactions with retry logic
func (r *TransactionRepositoryImpl) CreateTransactions(ctx context.Context, txs []*entity.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	return r.retryOperation(ctx, func() error {
		documents := make([]interface{}, len(txs))
		for i, tx := range txs {
			tx.ID = primitive.NewObjectID()
			documents[i] = tx
		}

		_, err := r.collection.InsertMany(ctx, documents)
		return err
	})
}

// UpsertTransactions upserts multiple transactions using bulk operations with retry logic
func (r *TransactionRepositoryImpl) UpsertTransactions(ctx context.Context, txs []*entity.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	return r.retryOperation(ctx, func() error {
		// Create bulk write operations
		var operations []mongo.WriteModel

		for _, tx := range txs {
			// Set ID if not already set
			if tx.ID.IsZero() {
				tx.ID = primitive.NewObjectID()
			}

			// Create filter based on transaction hash (unique identifier)
			filter := bson.M{"hash": tx.Hash}

			// Create update document
			update := bson.M{"$set": tx}

			// Create upsert operation
			upsertOp := mongo.NewUpdateOneModel()
			upsertOp.SetFilter(filter)
			upsertOp.SetUpdate(update)
			upsertOp.SetUpsert(true)

			operations = append(operations, upsertOp)
		}

		// Execute bulk write
		opts := options.BulkWrite().SetOrdered(false) // Allow parallel execution
		_, err := r.collection.BulkWrite(ctx, operations, opts)

		return err
	})
}

// GetTransactionByHash gets transaction by hash
func (r *TransactionRepositoryImpl) GetTransactionByHash(ctx context.Context, hash string) (*entity.Transaction, error) {
	filter := bson.M{"hash": hash}

	var tx entity.Transaction
	err := r.collection.FindOne(ctx, filter).Decode(&tx)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &tx, nil
}

// GetTransactionsByBlockHash gets transactions by block hash
func (r *TransactionRepositoryImpl) GetTransactionsByBlockHash(ctx context.Context, blockHash string) ([]*entity.Transaction, error) {
	filter := bson.M{"block_hash": blockHash}
	opts := options.Find().SetSort(bson.D{{Key: "transaction_index", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*entity.Transaction
	for cursor.Next(ctx) {
		var tx entity.Transaction
		if err := cursor.Decode(&tx); err != nil {
			return nil, err
		}
		transactions = append(transactions, &tx)
	}

	return transactions, cursor.Err()
}

// GetTransactionsByBlockNumber gets transactions by block number
func (r *TransactionRepositoryImpl) GetTransactionsByBlockNumber(ctx context.Context, blockNumber *big.Int) ([]*entity.Transaction, error) {
	filter := bson.M{"block_number": blockNumber.String()}
	opts := options.Find().SetSort(bson.D{{Key: "transaction_index", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*entity.Transaction
	for cursor.Next(ctx) {
		var tx entity.Transaction
		if err := cursor.Decode(&tx); err != nil {
			return nil, err
		}
		transactions = append(transactions, &tx)
	}

	return transactions, cursor.Err()
}

// GetTransactionsByAddress gets transactions by address
func (r *TransactionRepositoryImpl) GetTransactionsByAddress(ctx context.Context, address string, limit int, offset int) ([]*entity.Transaction, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"from": address},
			{"to": address},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "block_number", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*entity.Transaction
	for cursor.Next(ctx) {
		var tx entity.Transaction
		if err := cursor.Decode(&tx); err != nil {
			return nil, err
		}
		transactions = append(transactions, &tx)
	}

	return transactions, cursor.Err()
}

// GetTransactionsByStatus gets transactions by status
func (r *TransactionRepositoryImpl) GetTransactionsByStatus(ctx context.Context, status entity.TransactionStatus, limit int) ([]*entity.Transaction, error) {
	filter := bson.M{"tx_status": status}
	opts := options.Find().
		SetSort(bson.D{{Key: "block_number", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*entity.Transaction
	for cursor.Next(ctx) {
		var tx entity.Transaction
		if err := cursor.Decode(&tx); err != nil {
			return nil, err
		}
		transactions = append(transactions, &tx)
	}

	return transactions, cursor.Err()
}

// GetTransactionsByTimeRange gets transactions by time range
func (r *TransactionRepositoryImpl) GetTransactionsByTimeRange(ctx context.Context, startTime, endTime *big.Int) ([]*entity.Transaction, error) {
	filter := bson.M{
		"block_number": bson.M{
			"$gte": startTime.String(),
			"$lte": endTime.String(),
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "block_number", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*entity.Transaction
	for cursor.Next(ctx) {
		var tx entity.Transaction
		if err := cursor.Decode(&tx); err != nil {
			return nil, err
		}
		transactions = append(transactions, &tx)
	}

	return transactions, cursor.Err()
}

// UpdateTransactionStatus updates transaction status
func (r *TransactionRepositoryImpl) UpdateTransactionStatus(ctx context.Context, hash string, status entity.TransactionStatus) error {
	filter := bson.M{"hash": hash}
	update := bson.M{"$set": bson.M{"tx_status": status}}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// MarkTransactionAsProcessed marks transaction as processed
func (r *TransactionRepositoryImpl) MarkTransactionAsProcessed(ctx context.Context, hash string) error {
	filter := bson.M{"hash": hash}
	update := bson.M{
		"$set": bson.M{
			"tx_status":    entity.TransactionStatusProcessed,
			"processed_at": primitive.NewDateTimeFromTime(time.Now()),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// DeleteTransaction deletes transaction
func (r *TransactionRepositoryImpl) DeleteTransaction(ctx context.Context, hash string) error {
	filter := bson.M{"hash": hash}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// DeleteTransactionsByBlockHash deletes transactions by block hash
func (r *TransactionRepositoryImpl) DeleteTransactionsByBlockHash(ctx context.Context, blockHash string) error {
	filter := bson.M{"block_hash": blockHash}
	_, err := r.collection.DeleteMany(ctx, filter)
	return err
}

// TransactionExists checks if transaction exists
func (r *TransactionRepositoryImpl) TransactionExists(ctx context.Context, hash string) (bool, error) {
	filter := bson.M{"hash": hash}
	count, err := r.collection.CountDocuments(ctx, filter)
	return count > 0, err
}

// GetTransactionCount gets total transaction count
func (r *TransactionRepositoryImpl) GetTransactionCount(ctx context.Context, network string) (int64, error) {
	filter := bson.M{"network": network}
	return r.collection.CountDocuments(ctx, filter)
}

// GetTransactionCountByStatus gets transaction count by status
func (r *TransactionRepositoryImpl) GetTransactionCountByStatus(ctx context.Context, status entity.TransactionStatus, network string) (int64, error) {
	filter := bson.M{
		"network":   network,
		"tx_status": status,
	}
	return r.collection.CountDocuments(ctx, filter)
}

// GetTransactionCountByAddress gets transaction count by address
func (r *TransactionRepositoryImpl) GetTransactionCountByAddress(ctx context.Context, address string) (int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"from": address},
			{"to": address},
		},
	}
	return r.collection.CountDocuments(ctx, filter)
}

// GetTransactionVolumeByTimeRange gets transaction volume by time range
func (r *TransactionRepositoryImpl) GetTransactionVolumeByTimeRange(ctx context.Context, startTime, endTime *big.Int) (*big.Int, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"block_number": bson.M{
					"$gte": startTime.String(),
					"$lte": endTime.String(),
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": "$value"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return big.NewInt(0), err
	}
	defer cursor.Close(ctx)

	var result struct {
		Total string `bson:"total"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return big.NewInt(0), err
		}

		totalValue, ok := new(big.Int).SetString(result.Total, 10)
		if !ok {
			return big.NewInt(0), nil
		}
		return totalValue, nil
	}

	return big.NewInt(0), nil
}

// GetTopTransactionsByValue gets top transactions by value
func (r *TransactionRepositoryImpl) GetTopTransactionsByValue(ctx context.Context, limit int) ([]*entity.Transaction, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "value", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*entity.Transaction
	for cursor.Next(ctx) {
		var tx entity.Transaction
		if err := cursor.Decode(&tx); err != nil {
			return nil, err
		}
		transactions = append(transactions, &tx)
	}

	return transactions, cursor.Err()
}
