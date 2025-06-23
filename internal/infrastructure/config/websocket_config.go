package config

import "time"

// WebSocketConfig represents WebSocket listener configuration
type WebSocketConfig struct {
	// Connection settings
	ReconnectAttempts int           `mapstructure:"reconnect_attempts"` // Max reconnection attempts
	ReconnectDelay    time.Duration `mapstructure:"reconnect_delay"`    // Delay between reconnection attempts
	PingInterval      time.Duration `mapstructure:"ping_interval"`      // Ping interval for connection health
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`       // Read timeout
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`      // Write timeout

	// Data processing settings
	BufferSize    int           `mapstructure:"buffer_size"`    // Message buffer size
	BatchSize     int           `mapstructure:"batch_size"`     // Batch size for database writes
	FlushInterval time.Duration `mapstructure:"flush_interval"` // Interval to flush buffered data
	MaxRetries    int           `mapstructure:"max_retries"`    // Max retries for failed operations
	RetryDelay    time.Duration `mapstructure:"retry_delay"`    // Delay between retries

	// Data filtering
	SubscribeToBlocks bool `mapstructure:"subscribe_to_blocks"` // Subscribe to new blocks
	SubscribeToTxs    bool `mapstructure:"subscribe_to_txs"`    // Subscribe to pending transactions
	SubscribeToLogs   bool `mapstructure:"subscribe_to_logs"`   // Subscribe to contract logs

	// Health check settings
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"` // Health check interval
}
