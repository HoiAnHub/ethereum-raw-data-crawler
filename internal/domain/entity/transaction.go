package entity

import (
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Transaction represents an Ethereum transaction
type Transaction struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Hash              string             `bson:"hash" json:"hash"`
	BlockHash         string             `bson:"block_hash" json:"block_hash"`
	BlockNumber       *big.Int           `bson:"block_number" json:"block_number"`
	TransactionIndex  uint               `bson:"transaction_index" json:"transaction_index"`
	From              string             `bson:"from" json:"from"`
	To                *string            `bson:"to" json:"to"` // Can be nil for contract creation
	Value             *big.Int           `bson:"value" json:"value"`
	Gas               uint64             `bson:"gas" json:"gas"`
	GasPrice          *big.Int           `bson:"gas_price" json:"gas_price"`
	GasUsed           uint64             `bson:"gas_used" json:"gas_used"`
	CumulativeGasUsed uint64             `bson:"cumulative_gas_used" json:"cumulative_gas_used"`
	Data              string             `bson:"data" json:"data"`
	Nonce             uint64             `bson:"nonce" json:"nonce"`
	Status            uint64             `bson:"status" json:"status"` // 1 for success, 0 for failure

	// EIP-1559 fields
	MaxFeePerGas         *big.Int `bson:"max_fee_per_gas,omitempty" json:"max_fee_per_gas,omitempty"`
	MaxPriorityFeePerGas *big.Int `bson:"max_priority_fee_per_gas,omitempty" json:"max_priority_fee_per_gas,omitempty"`

	// Contract creation
	ContractAddress *string `bson:"contract_address,omitempty" json:"contract_address,omitempty"`

	// Metadata
	CrawledAt   time.Time         `bson:"crawled_at" json:"crawled_at"`
	Network     string            `bson:"network" json:"network"`
	ProcessedAt *time.Time        `bson:"processed_at,omitempty" json:"processed_at,omitempty"`
	TxStatus    TransactionStatus `bson:"tx_status" json:"tx_status"`
}

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusProcessed TransactionStatus = "processed"
	TransactionStatusFailed    TransactionStatus = "failed"
)
