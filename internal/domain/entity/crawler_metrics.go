package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CrawlerMetrics represents crawler monitoring metrics
type CrawlerMetrics struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`

	// Block metrics
	LastProcessedBlock uint64  `bson:"last_processed_block" json:"last_processed_block"`
	CurrentBlock       uint64  `bson:"current_block" json:"current_block"`
	BlocksProcessed    uint64  `bson:"blocks_processed" json:"blocks_processed"`
	BlocksPerSecond    float64 `bson:"blocks_per_second" json:"blocks_per_second"`

	// Transaction metrics
	TransactionsProcessed uint64  `bson:"transactions_processed" json:"transactions_processed"`
	TransactionsPerSecond float64 `bson:"transactions_per_second" json:"transactions_per_second"`

	// Error metrics
	ErrorCount       uint64     `bson:"error_count" json:"error_count"`
	LastErrorMessage string     `bson:"last_error_message" json:"last_error_message"`
	LastErrorTime    *time.Time `bson:"last_error_time,omitempty" json:"last_error_time,omitempty"`

	// Performance metrics
	AverageProcessingTime time.Duration `bson:"average_processing_time" json:"average_processing_time"`
	MemoryUsage           uint64        `bson:"memory_usage" json:"memory_usage"`
	GoroutineCount        int           `bson:"goroutine_count" json:"goroutine_count"`

	// Network metrics
	NetworkLatency time.Duration `bson:"network_latency" json:"network_latency"`
	RPCCallsCount  uint64        `bson:"rpc_calls_count" json:"rpc_calls_count"`

	// Database metrics
	DBConnectionCount int     `bson:"db_connection_count" json:"db_connection_count"`
	DBWritesPerSecond float64 `bson:"db_writes_per_second" json:"db_writes_per_second"`

	Network string `bson:"network" json:"network"`
}

// SystemHealth represents overall system health status
type SystemHealth struct {
	ID               primitive.ObjectID         `bson:"_id,omitempty" json:"id"`
	Timestamp        time.Time                  `bson:"timestamp" json:"timestamp"`
	Status           HealthStatus               `bson:"status" json:"status"`
	ComponentsHealth map[string]ComponentHealth `bson:"components_health" json:"components_health"`
	Message          string                     `bson:"message" json:"message"`
	Network          string                     `bson:"network" json:"network"`
}

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

type ComponentHealth struct {
	Status       HealthStatus  `bson:"status" json:"status"`
	LastChecked  time.Time     `bson:"last_checked" json:"last_checked"`
	Message      string        `bson:"message" json:"message"`
	ResponseTime time.Duration `bson:"response_time" json:"response_time"`
}
