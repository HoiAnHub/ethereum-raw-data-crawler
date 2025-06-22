package database

import (
	"context"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB represents MongoDB database connection
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
	config   *config.MongoDBConfig
}

// NewMongoDB creates new MongoDB connection
func NewMongoDB(cfg *config.MongoDBConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	// Configure client options
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMaxConnIdleTime(30 * time.Second).
		SetConnectTimeout(cfg.ConnectTimeout)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	database := client.Database(cfg.Database)

	return &MongoDB{
		Client:   client,
		Database: database,
		config:   cfg,
	}, nil
}

// Close closes MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

// GetCollection returns a collection
func (m *MongoDB) GetCollection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// CreateIndexes creates necessary indexes for collections
func (m *MongoDB) CreateIndexes(ctx context.Context) error {
	// Blocks collection indexes
	blocksCollection := m.GetCollection("blocks")

	blocksIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "number", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "hash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "network", Value: 1}, {Key: "number", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "timestamp", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
	}

	if _, err := blocksCollection.Indexes().CreateMany(ctx, blocksIndexes); err != nil {
		return err
	}

	// Transactions collection indexes
	transactionsCollection := m.GetCollection("transactions")

	transactionsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "hash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "block_hash", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "block_number", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "from", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "to", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "network", Value: 1}, {Key: "block_number", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "tx_status", Value: 1}},
		},
	}

	if _, err := transactionsCollection.Indexes().CreateMany(ctx, transactionsIndexes); err != nil {
		return err
	}

	// Crawler metrics collection indexes
	metricsCollection := m.GetCollection("crawler_metrics")

	metricsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "timestamp", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "network", Value: 1}, {Key: "timestamp", Value: 1}},
		},
	}

	if _, err := metricsCollection.Indexes().CreateMany(ctx, metricsIndexes); err != nil {
		return err
	}

	// System health collection indexes
	healthCollection := m.GetCollection("system_health")

	healthIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "timestamp", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "network", Value: 1}, {Key: "timestamp", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
	}

	if _, err := healthCollection.Indexes().CreateMany(ctx, healthIndexes); err != nil {
		return err
	}

	return nil
}

// HealthCheck performs MongoDB health check
func (m *MongoDB) HealthCheck(ctx context.Context) error {
	return m.Client.Ping(ctx, nil)
}
