package secondary

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/domain/repository"
	"ethereum-raw-data-crawler/internal/infrastructure/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MetricsRepositoryImpl implements MetricsRepository interface
type MetricsRepositoryImpl struct {
	db                *database.MongoDB
	metricsCollection *mongo.Collection
	healthCollection  *mongo.Collection
}

// NewMetricsRepository creates new metrics repository
func NewMetricsRepository(db *database.MongoDB) repository.MetricsRepository {
	return &MetricsRepositoryImpl{
		db:                db,
		metricsCollection: db.GetCollection("crawler_metrics"),
		healthCollection:  db.GetCollection("system_health"),
	}
}

// SaveCrawlerMetrics saves crawler metrics
func (r *MetricsRepositoryImpl) SaveCrawlerMetrics(ctx context.Context, metrics *entity.CrawlerMetrics) error {
	metrics.ID = primitive.NewObjectID()
	_, err := r.metricsCollection.InsertOne(ctx, metrics)
	return err
}

// GetLatestCrawlerMetrics gets latest crawler metrics
func (r *MetricsRepositoryImpl) GetLatestCrawlerMetrics(ctx context.Context, network string) (*entity.CrawlerMetrics, error) {
	filter := bson.M{"network": network}
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})

	var metrics entity.CrawlerMetrics
	err := r.metricsCollection.FindOne(ctx, filter, opts).Decode(&metrics)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &metrics, nil
}

// GetCrawlerMetricsByTimeRange gets crawler metrics by time range
func (r *MetricsRepositoryImpl) GetCrawlerMetricsByTimeRange(ctx context.Context, network string, startTime, endTime time.Time) ([]*entity.CrawlerMetrics, error) {
	filter := bson.M{
		"network": network,
		"timestamp": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	cursor, err := r.metricsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var metrics []*entity.CrawlerMetrics
	for cursor.Next(ctx) {
		var m entity.CrawlerMetrics
		if err := cursor.Decode(&m); err != nil {
			return nil, err
		}
		metrics = append(metrics, &m)
	}

	return metrics, cursor.Err()
}

// GetCrawlerMetricsHistory gets crawler metrics history
func (r *MetricsRepositoryImpl) GetCrawlerMetricsHistory(ctx context.Context, network string, limit int) ([]*entity.CrawlerMetrics, error) {
	filter := bson.M{"network": network}
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.metricsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var metrics []*entity.CrawlerMetrics
	for cursor.Next(ctx) {
		var m entity.CrawlerMetrics
		if err := cursor.Decode(&m); err != nil {
			return nil, err
		}
		metrics = append(metrics, &m)
	}

	return metrics, cursor.Err()
}

// SaveSystemHealth saves system health
func (r *MetricsRepositoryImpl) SaveSystemHealth(ctx context.Context, health *entity.SystemHealth) error {
	health.ID = primitive.NewObjectID()
	_, err := r.healthCollection.InsertOne(ctx, health)
	return err
}

// GetLatestSystemHealth gets latest system health
func (r *MetricsRepositoryImpl) GetLatestSystemHealth(ctx context.Context, network string) (*entity.SystemHealth, error) {
	filter := bson.M{"network": network}
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})

	var health entity.SystemHealth
	err := r.healthCollection.FindOne(ctx, filter, opts).Decode(&health)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &health, nil
}

// GetSystemHealthHistory gets system health history
func (r *MetricsRepositoryImpl) GetSystemHealthHistory(ctx context.Context, network string, limit int) ([]*entity.SystemHealth, error) {
	filter := bson.M{"network": network}
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.healthCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var healthRecords []*entity.SystemHealth
	for cursor.Next(ctx) {
		var h entity.SystemHealth
		if err := cursor.Decode(&h); err != nil {
			return nil, err
		}
		healthRecords = append(healthRecords, &h)
	}

	return healthRecords, cursor.Err()
}

// GetAverageProcessingTime gets average processing time
func (r *MetricsRepositoryImpl) GetAverageProcessingTime(ctx context.Context, network string, timeRange time.Duration) (time.Duration, error) {
	since := time.Now().Add(-timeRange)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"network": network,
				"timestamp": bson.M{
					"$gte": since,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":     nil,
				"average": bson.M{"$avg": "$average_processing_time"},
			},
		},
	}

	cursor, err := r.metricsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Average int64 `bson:"average"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return time.Duration(result.Average), nil
	}

	return 0, nil
}

// GetErrorRate gets error rate
func (r *MetricsRepositoryImpl) GetErrorRate(ctx context.Context, network string, timeRange time.Duration) (float64, error) {
	since := time.Now().Add(-timeRange)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"network": network,
				"timestamp": bson.M{
					"$gte": since,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":          nil,
				"total_blocks": bson.M{"$sum": "$blocks_processed"},
				"total_errors": bson.M{"$sum": "$error_count"},
			},
		},
		{
			"$project": bson.M{
				"error_rate": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$eq": []interface{}{"$total_blocks", 0}},
						"then": 0,
						"else": bson.M{"$divide": []interface{}{"$total_errors", "$total_blocks"}},
					},
				},
			},
		},
	}

	cursor, err := r.metricsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result struct {
		ErrorRate float64 `bson:"error_rate"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.ErrorRate, nil
	}

	return 0, nil
}

// GetThroughputStats gets throughput statistics
func (r *MetricsRepositoryImpl) GetThroughputStats(ctx context.Context, network string, timeRange time.Duration) (map[string]float64, error) {
	since := time.Now().Add(-timeRange)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"network": network,
				"timestamp": bson.M{
					"$gte": since,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":                         nil,
				"avg_blocks_per_second":       bson.M{"$avg": "$blocks_per_second"},
				"avg_transactions_per_second": bson.M{"$avg": "$transactions_per_second"},
				"max_blocks_per_second":       bson.M{"$max": "$blocks_per_second"},
				"max_transactions_per_second": bson.M{"$max": "$transactions_per_second"},
			},
		},
	}

	cursor, err := r.metricsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgBlocksPerSecond       float64 `bson:"avg_blocks_per_second"`
		AvgTransactionsPerSecond float64 `bson:"avg_transactions_per_second"`
		MaxBlocksPerSecond       float64 `bson:"max_blocks_per_second"`
		MaxTransactionsPerSecond float64 `bson:"max_transactions_per_second"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		return map[string]float64{
			"avg_blocks_per_second":       result.AvgBlocksPerSecond,
			"avg_transactions_per_second": result.AvgTransactionsPerSecond,
			"max_blocks_per_second":       result.MaxBlocksPerSecond,
			"max_transactions_per_second": result.MaxTransactionsPerSecond,
		}, nil
	}

	return make(map[string]float64), nil
}

// CleanupOldMetrics cleans up old metrics
func (r *MetricsRepositoryImpl) CleanupOldMetrics(ctx context.Context, olderThan time.Time) error {
	filter := bson.M{
		"timestamp": bson.M{
			"$lt": olderThan,
		},
	}

	// Delete old crawler metrics
	if _, err := r.metricsCollection.DeleteMany(ctx, filter); err != nil {
		return err
	}

	// Delete old health records
	if _, err := r.healthCollection.DeleteMany(ctx, filter); err != nil {
		return err
	}

	return nil
}
