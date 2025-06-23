package service

import (
	"context"
	"math/big"
)

// WebSocketListenerService defines the interface for websocket listeners
type WebSocketListenerService interface {
	// Connection management
	Start(ctx context.Context) error
	Stop() error
	IsConnected() bool

	// Subscription management
	SubscribeNewBlocks(ctx context.Context, callback func(*big.Int)) error
	SubscribePendingTransactions(ctx context.Context, callback func(string)) error
	SubscribeLogs(ctx context.Context, callback func(interface{})) error
	Unsubscribe() error

	// Health checks
	GetConnectionStats() map[string]interface{}
}
