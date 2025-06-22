package logger

import (
	"ethereum-raw-data-crawler/internal/infrastructure/config"

	"go.uber.org/zap"
)

// Logger wraps zap logger with additional functionality
type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// NewLogger creates a new logger instance
func NewLogger(cfg *config.Config) (*Logger, error) {
	var zapConfig zap.Config

	if cfg.App.Env == "production" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// Set log level
	level, err := zap.ParseAtomicLevel(cfg.App.LogLevel)
	if err != nil {
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	zapConfig.Level = level

	// Build logger
	zapLogger, err := zapConfig.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}, nil
}

// Info logs info level message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Infof logs info level formatted message
func (l *Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

// Debug logs debug level message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Debugf logs debug level formatted message
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

// Warn logs warning level message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Warnf logs warning level formatted message
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

// Error logs error level message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Errorf logs error level formatted message
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

// Fatal logs fatal level message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// Fatalf logs fatal level formatted message and exits
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

// With adds fields to logger
func (l *Logger) With(fields ...zap.Field) *Logger {
	newLogger := l.Logger.With(fields...)
	return &Logger{
		Logger: newLogger,
		sugar:  newLogger.Sugar(),
	}
}

// WithComponent adds component field to logger
func (l *Logger) WithComponent(name string) *Logger {
	return l.With(zap.String("component", name))
}

// WithTransaction adds transaction hash field to logger
func (l *Logger) WithTransaction(hash string) *Logger {
	return l.With(zap.String("tx_hash", hash))
}

// WithBlock adds block number field to logger
func (l *Logger) WithBlock(blockNumber uint64) *Logger {
	return l.With(zap.Uint64("block_number", blockNumber))
}

// WithError adds error field to logger
func (l *Logger) WithError(err error) *Logger {
	return l.With(zap.Error(err))
}

// Sync flushes buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
