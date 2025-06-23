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

// NewMongoDB creates new MongoDB connection with enhanced configuration
func NewMongoDB(cfg *config.MongoDBConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	// Configure client options with conservative settings for long-term stability
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(2).                           // Conservative minimum connections
		SetMaxConnIdleTime(60 * time.Second).        // Keep connections longer
		SetConnectTimeout(cfg.ConnectTimeout).       // Connection timeout
		SetSocketTimeout(60 * time.Second).          // Longer socket timeout
		SetServerSelectionTimeout(10 * time.Second). // Longer server selection timeout
		SetHeartbeatInterval(30 * time.Second).      // Less frequent heartbeats
		SetMaxConnecting(3).                         // Limit concurrent connections
		SetRetryWrites(true).                        // Enable retry writes
		SetRetryReads(true).                         // Enable retry reads
		SetCompressors([]string{"snappy"})           // Enable compression

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection with retry logic
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := client.Ping(ctx, nil); err != nil {
			if i == maxRetries-1 {
				client.Disconnect(ctx)
				return nil, err
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		break
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

// HealthCheck performs MongoDB health check with retry logic
func (m *MongoDB) HealthCheck(ctx context.Context) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := m.Client.Ping(ctx, nil); err != nil {
			if i == maxRetries-1 {
				return err
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		return nil
	}
	return nil
}

// Reconnect attempts to reconnect to MongoDB
func (m *MongoDB) Reconnect(ctx context.Context) error {
	// Close existing connection
	if err := m.Client.Disconnect(ctx); err != nil {
		// Log error but continue with reconnection attempt
	}

	// Configure client options with same conservative settings as NewMongoDB
	clientOptions := options.Client().
		ApplyURI(m.config.URI).
		SetMaxPoolSize(m.config.MaxPoolSize).
		SetMinPoolSize(2).
		SetMaxConnIdleTime(60 * time.Second).
		SetConnectTimeout(m.config.ConnectTimeout).
		SetSocketTimeout(60 * time.Second).
		SetServerSelectionTimeout(10 * time.Second).
		SetHeartbeatInterval(30 * time.Second).
		SetMaxConnecting(3).
		SetRetryWrites(true).
		SetRetryReads(true).
		SetCompressors([]string{"snappy"})

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return err
	}

	// Update client and database references
	m.Client = client
	m.Database = client.Database(m.config.Database)

	return nil
}

// IsConnected checks if MongoDB connection is active
func (m *MongoDB) IsConnected(ctx context.Context) bool {
	return m.Client.Ping(ctx, nil) == nil
}
