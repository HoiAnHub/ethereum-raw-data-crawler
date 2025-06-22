package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Block represents a blockchain block
type Block struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Number            string             `bson:"number" json:"number"`
	Hash              string             `bson:"hash" json:"hash"`
	ParentHash        string             `bson:"parent_hash" json:"parent_hash"`
	Nonce             uint64             `bson:"nonce" json:"nonce"`
	Sha3Uncles        string             `bson:"sha3_uncles" json:"sha3_uncles"`
	LogsBloom         string             `bson:"logs_bloom" json:"logs_bloom"`
	TransactionsRoot  string             `bson:"transactions_root" json:"transactions_root"`
	StateRoot         string             `bson:"state_root" json:"state_root"`
	ReceiptsRoot      string             `bson:"receipts_root" json:"receipts_root"`
	Miner             string             `bson:"miner" json:"miner"`
	Difficulty        string             `bson:"difficulty" json:"difficulty"`
	TotalDifficulty   string             `bson:"total_difficulty" json:"total_difficulty"`
	ExtraData         string             `bson:"extra_data" json:"extra_data"`
	Size              uint64             `bson:"size" json:"size"`
	GasLimit          uint64             `bson:"gas_limit" json:"gas_limit"`
	GasUsed           uint64             `bson:"gas_used" json:"gas_used"`
	Timestamp         time.Time          `bson:"timestamp" json:"timestamp"`
	TransactionHashes []string           `bson:"transaction_hashes" json:"transaction_hashes"`
	Uncles            []string           `bson:"uncles" json:"uncles"`

	// Metadata
	CrawledAt   time.Time   `bson:"crawled_at" json:"crawled_at"`
	Network     string      `bson:"network" json:"network"`
	ProcessedAt *time.Time  `bson:"processed_at,omitempty" json:"processed_at,omitempty"`
	Status      BlockStatus `bson:"status" json:"status"`
}

type BlockStatus string

const (
	BlockStatusPending   BlockStatus = "pending"
	BlockStatusProcessed BlockStatus = "processed"
	BlockStatusFailed    BlockStatus = "failed"
)
