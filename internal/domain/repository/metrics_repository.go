package repository

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"time"
)

// MetricsRepository interface for metrics data operations
type MetricsRepository interface {
	// Crawler metrics operations
	SaveCrawlerMetrics(ctx context.Context, metrics *entity.CrawlerMetrics) error
	GetLatestCrawlerMetrics(ctx context.Context, network string) (*entity.CrawlerMetrics, error)
	GetCrawlerMetricsByTimeRange(ctx context.Context, network string, startTime, endTime time.Time) ([]*entity.CrawlerMetrics, error)
	GetCrawlerMetricsHistory(ctx context.Context, network string, limit int) ([]*entity.CrawlerMetrics, error)

	// System health operations
	SaveSystemHealth(ctx context.Context, health *entity.SystemHealth) error
	GetLatestSystemHealth(ctx context.Context, network string) (*entity.SystemHealth, error)
	GetSystemHealthHistory(ctx context.Context, network string, limit int) ([]*entity.SystemHealth, error)

	// Aggregation operations
	GetAverageProcessingTime(ctx context.Context, network string, timeRange time.Duration) (time.Duration, error)
	GetErrorRate(ctx context.Context, network string, timeRange time.Duration) (float64, error)
	GetThroughputStats(ctx context.Context, network string, timeRange time.Duration) (map[string]float64, error)

	// Cleanup operations
	CleanupOldMetrics(ctx context.Context, olderThan time.Time) error
}
