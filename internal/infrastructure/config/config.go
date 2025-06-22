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
	GraphQL    GraphQLConfig    `mapstructure:"graphql"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
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

	// GraphQL defaults
	viper.SetDefault("graphql.endpoint", "/graphql")
	viper.SetDefault("graphql.playground", true)

	// Monitoring defaults
	viper.SetDefault("monitoring.metrics_enabled", true)
	viper.SetDefault("monitoring.health_check_interval", "30s")
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

	// GraphQL
	viper.BindEnv("graphql.endpoint", "GRAPHQL_ENDPOINT")
	viper.BindEnv("graphql.playground", "GRAPHQL_PLAYGROUND")

	// Monitoring
	viper.BindEnv("monitoring.metrics_enabled", "METRICS_ENABLED")
	viper.BindEnv("monitoring.health_check_interval", "HEALTH_CHECK_INTERVAL")
}
