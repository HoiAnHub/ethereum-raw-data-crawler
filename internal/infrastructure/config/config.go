package config

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents application configuration
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Ethereum   EthereumConfig   `mapstructure:"ethereum"`
	MongoDB    MongoDBConfig    `mapstructure:"mongodb"`
	Crawler    CrawlerConfig    `mapstructure:"crawler"`
	Scheduler  SchedulerConfig  `mapstructure:"scheduler"`
	WebSocket  WebSocketConfig  `mapstructure:"websocket"`
	GraphQL    GraphQLConfig    `mapstructure:"graphql"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	NATS       NATSConfig       `mapstructure:"nats"`
}

// AppConfig represents application configuration
type AppConfig struct {
	Port     int    `mapstructure:"port"`
	Env      string `mapstructure:"env"`
	LogLevel string `mapstructure:"log_level"`
}

// EthereumConfig represents Ethereum network configuration
type EthereumConfig struct {
	RPCURL         string        `mapstructure:"rpc_url"`
	WSURL          string        `mapstructure:"ws_url"`
	StartBlock     uint64        `mapstructure:"start_block"`
	Network        string        `mapstructure:"network"`
	ChainID        int64         `mapstructure:"chain_id"`
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	RateLimit      time.Duration `mapstructure:"rate_limit"`
	SkipReceipts   bool          `mapstructure:"skip_receipts"`
}

// MongoDBConfig represents MongoDB configuration
type MongoDBConfig struct {
	URI            string        `mapstructure:"uri"`
	Database       string        `mapstructure:"database"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
	MaxPoolSize    uint64        `mapstructure:"max_pool_size"`
}

// CrawlerConfig represents crawler configuration
type CrawlerConfig struct {
	BatchSize         int           `mapstructure:"batch_size"`
	ConcurrentWorkers int           `mapstructure:"concurrent_workers"`
	RetryAttempts     int           `mapstructure:"retry_attempts"`
	RetryDelay        time.Duration `mapstructure:"retry_delay"`

	// Batch upsert configuration
	UseUpsert      bool `mapstructure:"use_upsert"`      // Enable batch upsert instead of insert
	UpsertFallback bool `mapstructure:"upsert_fallback"` // Fallback to insert if upsert fails
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	Mode              string        `mapstructure:"mode"`               // polling, realtime, hybrid
	EnableRealtime    bool          `mapstructure:"enable_realtime"`    // Enable WebSocket real-time scheduling
	EnablePolling     bool          `mapstructure:"enable_polling"`     // Enable polling fallback
	PollingInterval   time.Duration `mapstructure:"polling_interval"`   // Polling interval
	FallbackTimeout   time.Duration `mapstructure:"fallback_timeout"`   // Time to wait before fallback to polling
	ReconnectAttempts int           `mapstructure:"reconnect_attempts"` // Max WebSocket reconnection attempts
	ReconnectDelay    time.Duration `mapstructure:"reconnect_delay"`    // Delay between reconnection attempts
	MaxRetries        int           `mapstructure:"max_retries"`        // Max retries for failed blocks
	SkipDuration      time.Duration `mapstructure:"skip_duration"`      // Duration to skip failed blocks
}

// GraphQLConfig represents GraphQL configuration
type GraphQLConfig struct {
	Endpoint   string `mapstructure:"endpoint"`
	Playground bool   `mapstructure:"playground"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	MetricsEnabled      bool          `mapstructure:"metrics_enabled"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
}

// NATSConfig represents NATS JetStream configuration
type NATSConfig struct {
	URL                string        `mapstructure:"url"`
	StreamName         string        `mapstructure:"stream_name"`
	SubjectPrefix      string        `mapstructure:"subject_prefix"`
	ConnectTimeout     time.Duration `mapstructure:"connect_timeout"`
	ReconnectAttempts  int           `mapstructure:"reconnect_attempts"`
	ReconnectDelay     time.Duration `mapstructure:"reconnect_delay"`
	MaxPendingMessages int           `mapstructure:"max_pending_messages"`
	Enabled            bool          `mapstructure:"enabled"`
}

// loadEnvFile manually loads environment variables from .env file
func loadEnvFile() error {
	file, err := os.Open(".env")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// LoadConfig loads configuration from environment variables and config files
func LoadConfig() (*Config, error) {
	// Load .env file manually first
	if _, err := os.Stat(".env"); err == nil {
		if err := loadEnvFile(); err != nil {
			return nil, err
		}
	}

	// Set default values first
	setDefaults()

	// Bind environment variables
	bindEnvVars()

	// Read environment variables automatically
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.log_level", "info")

	// Ethereum defaults
	viper.SetDefault("ethereum.start_block", 1)
	viper.SetDefault("ethereum.network", "ethereum")
	viper.SetDefault("ethereum.chain_id", 1)
	viper.SetDefault("ethereum.request_timeout", "60s")
	viper.SetDefault("ethereum.rate_limit", "500ms")
	viper.SetDefault("ethereum.skip_receipts", false)

	// MongoDB defaults
	viper.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database", "ethereum_raw_data")
	viper.SetDefault("mongodb.connect_timeout", "10s")
	viper.SetDefault("mongodb.max_pool_size", 100)

	// Crawler defaults
	viper.SetDefault("crawler.batch_size", 100)
	viper.SetDefault("crawler.concurrent_workers", 10)
	viper.SetDefault("crawler.retry_attempts", 3)
	viper.SetDefault("crawler.retry_delay", "5s")
	viper.SetDefault("crawler.use_upsert", true)
	viper.SetDefault("crawler.upsert_fallback", true)

	// Scheduler defaults
	viper.SetDefault("scheduler.mode", "hybrid")
	viper.SetDefault("scheduler.enable_realtime", true)
	viper.SetDefault("scheduler.enable_polling", true)
	viper.SetDefault("scheduler.polling_interval", "3s")
	viper.SetDefault("scheduler.fallback_timeout", "30s")
	viper.SetDefault("scheduler.reconnect_attempts", 5)
	viper.SetDefault("scheduler.reconnect_delay", "5s")
	viper.SetDefault("scheduler.max_retries", 3)
	viper.SetDefault("scheduler.skip_duration", "30s")

	// GraphQL defaults
	viper.SetDefault("graphql.endpoint", "/graphql")
	viper.SetDefault("graphql.playground", true)

	// Monitoring defaults
	viper.SetDefault("monitoring.metrics_enabled", true)
	viper.SetDefault("monitoring.health_check_interval", "30s")

	// WebSocket defaults
	viper.SetDefault("websocket.reconnect_attempts", 5)
	viper.SetDefault("websocket.reconnect_delay", "5s")
	viper.SetDefault("websocket.ping_interval", "30s")
	viper.SetDefault("websocket.read_timeout", "60s")
	viper.SetDefault("websocket.write_timeout", "10s")
	viper.SetDefault("websocket.buffer_size", 100)
	viper.SetDefault("websocket.batch_size", 10)
	viper.SetDefault("websocket.flush_interval", "5s")
	viper.SetDefault("websocket.max_retries", 3)
	viper.SetDefault("websocket.retry_delay", "2s")
	viper.SetDefault("websocket.subscribe_to_blocks", true)
	viper.SetDefault("websocket.subscribe_to_txs", false)
	viper.SetDefault("websocket.subscribe_to_logs", false)
	viper.SetDefault("websocket.health_check_interval", "30s")

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.stream_name", "TRANSACTIONS")
	viper.SetDefault("nats.subject_prefix", "transactions")
	viper.SetDefault("nats.connect_timeout", "10s")
	viper.SetDefault("nats.reconnect_attempts", 5)
	viper.SetDefault("nats.reconnect_delay", "2s")
	viper.SetDefault("nats.max_pending_messages", 1000)
	viper.SetDefault("nats.enabled", false)
}

func bindEnvVars() {
	// App
	viper.BindEnv("app.port", "APP_PORT")
	viper.BindEnv("app.env", "APP_ENV")
	viper.BindEnv("app.log_level", "LOG_LEVEL")

	// Ethereum
	viper.BindEnv("ethereum.rpc_url", "ETHEREUM_RPC_URL")
	viper.BindEnv("ethereum.ws_url", "ETHEREUM_WS_URL")
	viper.BindEnv("ethereum.start_block", "START_BLOCK_NUMBER")
	viper.BindEnv("ethereum.request_timeout", "ETHEREUM_REQUEST_TIMEOUT")
	viper.BindEnv("ethereum.rate_limit", "ETHEREUM_RATE_LIMIT")
	viper.BindEnv("ethereum.skip_receipts", "ETHEREUM_SKIP_RECEIPTS")

	// MongoDB
	viper.BindEnv("mongodb.uri", "MONGO_URI")
	viper.BindEnv("mongodb.database", "MONGO_DATABASE")
	viper.BindEnv("mongodb.connect_timeout", "MONGO_CONNECT_TIMEOUT")
	viper.BindEnv("mongodb.max_pool_size", "MONGO_MAX_POOL_SIZE")

	// Crawler
	viper.BindEnv("crawler.batch_size", "BATCH_SIZE")
	viper.BindEnv("crawler.concurrent_workers", "CONCURRENT_WORKERS")
	viper.BindEnv("crawler.retry_attempts", "RETRY_ATTEMPTS")
	viper.BindEnv("crawler.retry_delay", "RETRY_DELAY")
	viper.BindEnv("crawler.use_upsert", "CRAWLER_USE_UPSERT")
	viper.BindEnv("crawler.upsert_fallback", "CRAWLER_UPSERT_FALLBACK")

	// Scheduler
	viper.BindEnv("scheduler.mode", "SCHEDULER_MODE")
	viper.BindEnv("scheduler.enable_realtime", "SCHEDULER_ENABLE_REALTIME")
	viper.BindEnv("scheduler.enable_polling", "SCHEDULER_ENABLE_POLLING")
	viper.BindEnv("scheduler.polling_interval", "SCHEDULER_POLLING_INTERVAL")
	viper.BindEnv("scheduler.fallback_timeout", "SCHEDULER_FALLBACK_TIMEOUT")
	viper.BindEnv("scheduler.reconnect_attempts", "SCHEDULER_RECONNECT_ATTEMPTS")
	viper.BindEnv("scheduler.reconnect_delay", "SCHEDULER_RECONNECT_DELAY")
	viper.BindEnv("scheduler.max_retries", "SCHEDULER_MAX_RETRIES")
	viper.BindEnv("scheduler.skip_duration", "SCHEDULER_SKIP_DURATION")

	// GraphQL
	viper.BindEnv("graphql.endpoint", "GRAPHQL_ENDPOINT")
	viper.BindEnv("graphql.playground", "GRAPHQL_PLAYGROUND")

	// Monitoring
	viper.BindEnv("monitoring.metrics_enabled", "METRICS_ENABLED")
	viper.BindEnv("monitoring.health_check_interval", "HEALTH_CHECK_INTERVAL")

	// WebSocket
	viper.BindEnv("websocket.reconnect_attempts", "WEBSOCKET_RECONNECT_ATTEMPTS")
	viper.BindEnv("websocket.reconnect_delay", "WEBSOCKET_RECONNECT_DELAY")
	viper.BindEnv("websocket.ping_interval", "WEBSOCKET_PING_INTERVAL")
	viper.BindEnv("websocket.read_timeout", "WEBSOCKET_READ_TIMEOUT")
	viper.BindEnv("websocket.write_timeout", "WEBSOCKET_WRITE_TIMEOUT")
	viper.BindEnv("websocket.buffer_size", "WEBSOCKET_BUFFER_SIZE")
	viper.BindEnv("websocket.batch_size", "WEBSOCKET_BATCH_SIZE")
	viper.BindEnv("websocket.flush_interval", "WEBSOCKET_FLUSH_INTERVAL")
	viper.BindEnv("websocket.max_retries", "WEBSOCKET_MAX_RETRIES")
	viper.BindEnv("websocket.retry_delay", "WEBSOCKET_RETRY_DELAY")
	viper.BindEnv("websocket.subscribe_to_blocks", "WEBSOCKET_SUBSCRIBE_TO_BLOCKS")
	viper.BindEnv("websocket.subscribe_to_txs", "WEBSOCKET_SUBSCRIBE_TO_TXS")
	viper.BindEnv("websocket.subscribe_to_logs", "WEBSOCKET_SUBSCRIBE_TO_LOGS")
	viper.BindEnv("websocket.health_check_interval", "WEBSOCKET_HEALTH_CHECK_INTERVAL")

	// NATS
	viper.BindEnv("nats.url", "NATS_URL")
	viper.BindEnv("nats.stream_name", "NATS_STREAM_NAME")
	viper.BindEnv("nats.subject_prefix", "NATS_SUBJECT_PREFIX")
	viper.BindEnv("nats.connect_timeout", "NATS_CONNECT_TIMEOUT")
	viper.BindEnv("nats.reconnect_attempts", "NATS_RECONNECT_ATTEMPTS")
	viper.BindEnv("nats.reconnect_delay", "NATS_RECONNECT_DELAY")
	viper.BindEnv("nats.max_pending_messages", "NATS_MAX_PENDING_MESSAGES")
	viper.BindEnv("nats.enabled", "NATS_ENABLED")
}
