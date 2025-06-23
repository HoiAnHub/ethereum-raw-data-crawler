package service

import (
	"context"
	"ethereum-raw-data-crawler/internal/domain/entity"
)

// MessagingService defines the interface for publishing transaction events
type MessagingService interface {
	// Connect establishes connection to the messaging system
	Connect(ctx context.Context) error
	
	// Disconnect closes connection to the messaging system
	Disconnect() error
	
	// IsConnected checks if connected to the messaging system
	IsConnected() bool
	
	// PublishTransaction publishes a single transaction event
	PublishTransaction(ctx context.Context, tx *entity.Transaction) error
	
	// PublishTransactions publishes multiple transaction events
	PublishTransactions(ctx context.Context, transactions []*entity.Transaction) error
	
	// GetStreamInfo returns information about the message stream (if applicable)
	GetStreamInfo() (interface{}, error)
}
