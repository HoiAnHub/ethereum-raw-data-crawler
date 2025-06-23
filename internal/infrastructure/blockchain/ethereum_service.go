package blockchain

import (
	"context"
	"encoding/hex"
	"errors"
	"ethereum-raw-data-crawler/internal/domain/entity"
	"ethereum-raw-data-crawler/internal/domain/service"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/logger"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"go.uber.org/zap"
)

// EthereumService implements BlockchainService for Ethereum
type EthereumService struct {
	client          *ethclient.Client
	config          *config.EthereumConfig
	logger          *logger.Logger
	isConnected     bool
	lastRequestTime time.Time
	minRequestDelay time.Duration
}

// NewEthereumService creates new Ethereum service
func NewEthereumService(cfg *config.EthereumConfig, logger *logger.Logger) service.BlockchainService {
	return &EthereumService{
		config:          cfg,
		logger:          logger.WithComponent("ethereum-service"),
		minRequestDelay: cfg.RateLimit, // Use configurable rate limit
	}
}

// Connect connects to Ethereum node
func (s *EthereumService) Connect(ctx context.Context) error {
	s.logger.Info("Connecting to Ethereum node", zap.String("rpc_url", s.config.RPCURL))

	client, err := ethclient.DialContext(ctx, s.config.RPCURL)
	if err != nil {
		s.logger.Error("Failed to connect to Ethereum node", zap.Error(err))
		return err
	}

	s.client = client
	s.isConnected = true

	// Verify connection
	if err := s.HealthCheck(ctx); err != nil {
		s.logger.Error("Health check failed after connection", zap.Error(err))
		s.isConnected = false
		return err
	}

	s.logger.Info("Successfully connected to Ethereum node")
	return nil
}

// Disconnect disconnects from Ethereum node
func (s *EthereumService) Disconnect() error {
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
	s.isConnected = false
	s.logger.Info("Disconnected from Ethereum node")
	return nil
}

// IsConnected checks if connected to Ethereum node
func (s *EthereumService) IsConnected() bool {
	return s.isConnected && s.client != nil
}

// reconnect attempts to reconnect to Ethereum node
func (s *EthereumService) reconnect(ctx context.Context) error {
	s.logger.Warn("Attempting to reconnect to Ethereum node")

	// Disconnect first
	s.Disconnect()

	// Try to reconnect
	if err := s.Connect(ctx); err != nil {
		s.logger.Error("Failed to reconnect to Ethereum node", zap.Error(err))
		return err
	}

	s.logger.Info("Successfully reconnected to Ethereum node")
	return nil
}

// isConnectionError checks if the error indicates a connection problem
func (s *EthereumService) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"EOF",
		"context deadline exceeded",
		"no such host",
		"network unreachable",
		"broken pipe",
		"connection timed out",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(strings.ToLower(errStr), connErr) {
			return true
		}
	}

	return false
}

// GetLatestBlockNumber gets latest block number
func (s *EthereumService) GetLatestBlockNumber(ctx context.Context) (*big.Int, error) {
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	blockNumber, err := s.client.BlockNumber(ctx)
	if err != nil {
		s.logger.Error("Failed to get latest block number", zap.Error(err))
		return nil, err
	}

	return new(big.Int).SetUint64(blockNumber), nil
}

// GetBlockByNumber gets block by number with rate limiting and retry logic
func (s *EthereumService) GetBlockByNumber(ctx context.Context, blockNumber *big.Int) (*entity.Block, error) {
	if !s.IsConnected() {
		if err := s.reconnect(ctx); err != nil {
			return nil, ErrNotConnected
		}
	}

	s.logger.Debug("Getting block by number", zap.String("block_number", blockNumber.String()))

	var block *types.Block
	var err error

	// Retry with exponential backoff for rate limiting
	for attempt := 1; attempt <= 3; attempt++ {
		s.rateLimit() // Apply rate limiting

		block, err = s.client.BlockByNumber(ctx, blockNumber)
		if err == nil {
			break
		}

		// Check if this is a connection error and try to reconnect
		if s.isConnectionError(err) && attempt == 1 {
			s.logger.Warn("Connection error detected, attempting reconnect", zap.Error(err))
			if reconnectErr := s.reconnect(ctx); reconnectErr == nil {
				continue // Retry after successful reconnect
			}
		}

		// Handle rate limiting
		if err = s.handleRateLimitError(err, attempt); err != nil && attempt < 3 {
			continue
		}

		if attempt == 3 {
			s.logger.Error("Failed to get block by number after retries",
				zap.String("block_number", blockNumber.String()),
				zap.Error(err))
			return nil, err
		}
	}

	return s.convertBlock(block), nil
}

// GetBlockByHash gets block by hash
func (s *EthereumService) GetBlockByHash(ctx context.Context, blockHash string) (*entity.Block, error) {
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	hash := common.HexToHash(blockHash)
	block, err := s.client.BlockByHash(ctx, hash)
	if err != nil {
		s.logger.Error("Failed to get block by hash",
			zap.String("block_hash", blockHash),
			zap.Error(err))
		return nil, err
	}

	return s.convertBlock(block), nil
}

// GetTransactionByHash gets transaction by hash
func (s *EthereumService) GetTransactionByHash(ctx context.Context, txHash string) (*entity.Transaction, error) {
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	hash := common.HexToHash(txHash)
	tx, isPending, err := s.client.TransactionByHash(ctx, hash)
	if err != nil {
		s.logger.Error("Failed to get transaction by hash",
			zap.String("tx_hash", txHash),
			zap.Error(err))
		return nil, err
	}

	if isPending {
		s.logger.Debug("Transaction is pending", zap.String("tx_hash", txHash))
		return nil, ErrTransactionPending
	}

	return s.convertTransaction(tx, nil, nil, 0), nil
}

// GetTransactionReceipt gets transaction receipt
func (s *EthereumService) GetTransactionReceipt(ctx context.Context, txHash string) (*entity.Transaction, error) {
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	hash := common.HexToHash(txHash)
	receipt, err := s.client.TransactionReceipt(ctx, hash)
	if err != nil {
		s.logger.Error("Failed to get transaction receipt",
			zap.String("tx_hash", txHash),
			zap.Error(err))
		return nil, err
	}

	// Get the transaction details as well
	tx, _, err := s.client.TransactionByHash(ctx, hash)
	if err != nil {
		s.logger.Error("Failed to get transaction details for receipt",
			zap.String("tx_hash", txHash),
			zap.Error(err))
		return nil, err
	}

	return s.convertTransaction(tx, receipt, nil, 0), nil
}

// GetTransactionsByBlock gets all transactions in a block
func (s *EthereumService) GetTransactionsByBlock(ctx context.Context, blockNumber *big.Int) ([]*entity.Transaction, error) {
	if !s.IsConnected() {
		if err := s.reconnect(ctx); err != nil {
			return nil, ErrNotConnected
		}
	}

	block, err := s.client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	transactions := make([]*entity.Transaction, 0, len(block.Transactions()))

	s.logger.Info("Processing transactions in block",
		zap.String("block_number", blockNumber.String()),
		zap.Int("tx_count", len(block.Transactions())))

	for i, tx := range block.Transactions() {
		var receipt *types.Receipt
		var err error

		// Skip receipt fetching if configured to do so (for faster testing)
		if !s.config.SkipReceipts {
			// Apply rate limiting before each transaction receipt call
			s.rateLimit()

			// Create context with configurable timeout for individual transaction
			txCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)

			// Get receipt for each transaction with retry logic
			receipt, err = s.getTransactionReceiptWithRetry(txCtx, tx.Hash())
			cancel()

			if err != nil {
				s.logger.Warn("Failed to get receipt for transaction",
					zap.String("tx_hash", tx.Hash().Hex()),
					zap.Int("tx_index", i),
					zap.Error(err))
				// Continue without receipt but include the transaction
			}
		}

		transactions = append(transactions, s.convertTransaction(tx, receipt, block, uint(i)))

		// Log progress for blocks with transactions
		if len(block.Transactions()) > 10 && (i+1)%10 == 0 {
			s.logger.Info("Transaction processing progress",
				zap.String("block_number", blockNumber.String()),
				zap.Int("processed", i+1),
				zap.Int("total", len(block.Transactions())))
		}
	}

	s.logger.Info("Completed processing transactions in block",
		zap.String("block_number", blockNumber.String()),
		zap.Int("successful", len(transactions)))

	return transactions, nil
}

// sanitizeData converts raw bytes to a safe UTF-8 string for MongoDB storage
func (s *EthereumService) sanitizeData(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// Convert to hex string to ensure valid UTF-8
	return "0x" + hex.EncodeToString(data)
}

// getTransactionReceiptWithRetry gets transaction receipt with retry logic
func (s *EthereumService) getTransactionReceiptWithRetry(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	var err error

	// Retry with exponential backoff for rate limiting and timeouts
	maxRetries := 5 // Increased retries for better reliability
	for attempt := 1; attempt <= maxRetries; attempt++ {
		receipt, err = s.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		// Don't retry if context is cancelled or deadline exceeded from parent
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Handle rate limiting and timeouts with backoff
		if attempt < maxRetries {
			s.handleRateLimitError(err, attempt)
			// Additional progressive delay for retries
			additionalDelay := time.Duration(attempt) * 500 * time.Millisecond
			time.Sleep(additionalDelay)
			continue
		}

		// Final attempt failed
		return nil, err
	}

	return receipt, err
}

// GetBlocksInRange gets blocks in range
func (s *EthereumService) GetBlocksInRange(ctx context.Context, startBlock, endBlock *big.Int) ([]*entity.Block, error) {
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	var blocks []*entity.Block

	for i := new(big.Int).Set(startBlock); i.Cmp(endBlock) <= 0; i.Add(i, big.NewInt(1)) {
		block, err := s.GetBlockByNumber(ctx, i)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// GetNetworkID gets network ID
func (s *EthereumService) GetNetworkID(ctx context.Context) (*big.Int, error) {
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	return s.client.NetworkID(ctx)
}

// GetGasPrice gets current gas price
func (s *EthereumService) GetGasPrice(ctx context.Context) (*big.Int, error) {
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	return s.client.SuggestGasPrice(ctx)
}

// GetPeerCount gets peer count
func (s *EthereumService) GetPeerCount(ctx context.Context) (uint64, error) {
	if !s.IsConnected() {
		return 0, ErrNotConnected
	}

	// Note: This requires admin access to the node
	// For now, return 0 as it's not critical for crawler functionality
	return 0, nil
}

// HealthCheck performs health check
func (s *EthereumService) HealthCheck(ctx context.Context) error {
	if !s.IsConnected() {
		return ErrNotConnected
	}

	// Try to get latest block number
	_, err := s.client.BlockNumber(ctx)
	return err
}

// convertBlock converts go-ethereum Block to entity.Block
func (s *EthereumService) convertBlock(block *types.Block) *entity.Block {
	txHashes := make([]string, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		txHashes[i] = tx.Hash().Hex()
	}

	uncles := make([]string, len(block.Uncles()))
	for i, uncle := range block.Uncles() {
		uncles[i] = uncle.Hash().Hex()
	}

	return &entity.Block{
		Number:            block.Number().String(),
		Hash:              block.Hash().Hex(),
		ParentHash:        block.ParentHash().Hex(),
		Nonce:             block.Nonce(),
		Sha3Uncles:        block.UncleHash().Hex(),
		LogsBloom:         "0x" + hex.EncodeToString(block.Bloom().Bytes()),
		TransactionsRoot:  block.TxHash().Hex(),
		StateRoot:         block.Root().Hex(),
		ReceiptsRoot:      block.ReceiptHash().Hex(),
		Miner:             block.Coinbase().Hex(),
		Difficulty:        block.Difficulty().String(),
		TotalDifficulty:   "0", // Note: This requires separate call
		ExtraData:         s.sanitizeData(block.Extra()),
		Size:              block.Size(),
		GasLimit:          block.GasLimit(),
		GasUsed:           block.GasUsed(),
		Timestamp:         time.Unix(int64(block.Time()), 0),
		TransactionHashes: txHashes,
		Uncles:            uncles,
		CrawledAt:         time.Now(),
		Network:           s.config.Network,
		Status:            entity.BlockStatusPending,
	}
}

// convertTransaction converts go-ethereum Transaction to entity.Transaction
func (s *EthereumService) convertTransaction(tx *types.Transaction, receipt *types.Receipt, block *types.Block, txIndex uint) *entity.Transaction {
	var to *string
	if tx.To() != nil {
		toAddr := tx.To().Hex()
		to = &toAddr
	}

	var contractAddress *string
	var status uint64 = 1 // Default success
	var gasUsed uint64
	var cumulativeGasUsed uint64
	var blockHash string
	var blockNumber *big.Int
	var transactionIndex uint = txIndex

	if receipt != nil {
		if receipt.ContractAddress != (common.Address{}) {
			addr := receipt.ContractAddress.Hex()
			contractAddress = &addr
		}
		status = receipt.Status
		gasUsed = receipt.GasUsed
		cumulativeGasUsed = receipt.CumulativeGasUsed
		blockHash = receipt.BlockHash.Hex()
		blockNumber = receipt.BlockNumber
		transactionIndex = receipt.TransactionIndex
	} else if block != nil {
		// Use block context when receipt is not available
		blockHash = block.Hash().Hex()
		blockNumber = block.Number()
		transactionIndex = txIndex
	}

	// Get from address with multiple approaches
	var fromAddr string

	// First try: Use latest signer with chain config (should handle most transaction types)
	if from, err := types.Sender(types.LatestSigner(params.MainnetChainConfig), tx); err == nil {
		fromAddr = from.Hex()
	} else {
		// Fallback: Try different signers to handle various transaction types
		signers := []types.Signer{
			types.NewLondonSigner(tx.ChainId()), // For EIP-1559 transactions (type 2)
			types.NewEIP155Signer(tx.ChainId()), // For EIP-155 transactions (type 0)
			types.HomesteadSigner{},             // For pre-EIP155 transactions
			types.FrontierSigner{},              // For frontier transactions
		}

		for _, signer := range signers {
			if from, err := types.Sender(signer, tx); err == nil {
				fromAddr = from.Hex()
				break
			}
		}
	}

	// Log if we couldn't extract sender (only for non-type-3 transactions)
	if fromAddr == "" {
		if tx.Type() == 3 {
			// EIP-4844 blob transactions - known limitation with current go-ethereum version
			s.logger.Debug("Cannot extract sender for EIP-4844 blob transaction (known limitation)",
				zap.String("tx_hash", tx.Hash().Hex()),
				zap.String("tx_type", "3"))
		} else {
			// This is unexpected for other transaction types
			s.logger.Warn("Failed to extract sender address",
				zap.String("tx_hash", tx.Hash().Hex()),
				zap.String("chain_id", tx.ChainId().String()),
				zap.String("tx_type", fmt.Sprintf("%d", tx.Type())))
		}
	}

	// Handle blockNumber conversion safely
	var blockNumberStr string
	if blockNumber != nil {
		blockNumberStr = blockNumber.String()
	}

	// Determine transaction status based on context
	var txStatus entity.TransactionStatus
	if receipt != nil {
		// If we have receipt, transaction is processed
		if receipt.Status == 1 {
			txStatus = entity.TransactionStatusProcessed
		} else {
			txStatus = entity.TransactionStatusFailed
		}
	} else if block != nil && blockHash != "" {
		// If transaction is in a block, it's processed (we just don't have receipt)
		txStatus = entity.TransactionStatusProcessed
	} else {
		// Otherwise it's pending
		txStatus = entity.TransactionStatusPending
	}

	return &entity.Transaction{
		Hash:                 tx.Hash().Hex(),
		BlockHash:            blockHash,
		BlockNumber:          blockNumberStr,
		TransactionIndex:     transactionIndex,
		From:                 fromAddr,
		To:                   to,
		Value:                tx.Value().String(),
		Gas:                  tx.Gas(),
		GasPrice:             tx.GasPrice().String(),
		GasUsed:              gasUsed,
		CumulativeGasUsed:    cumulativeGasUsed,
		Data:                 s.sanitizeData(tx.Data()),
		Nonce:                tx.Nonce(),
		Status:               status,
		MaxFeePerGas:         tx.GasFeeCap().String(),
		MaxPriorityFeePerGas: tx.GasTipCap().String(),
		ContractAddress:      contractAddress,
		CrawledAt:            time.Now(),
		Network:              s.config.Network,
		TxStatus:             txStatus,
	}
}

// Common errors
var (
	ErrNotConnected       = errors.New("not connected to blockchain node")
	ErrTransactionPending = errors.New("transaction is pending")
)

// rateLimit ensures minimum delay between requests
func (s *EthereumService) rateLimit() {
	elapsed := time.Since(s.lastRequestTime)
	if elapsed < s.minRequestDelay {
		time.Sleep(s.minRequestDelay - elapsed)
	}
	s.lastRequestTime = time.Now()
}

// handleRateLimitError checks if error is rate limit or timeout and implements exponential backoff
func (s *EthereumService) handleRateLimitError(err error, attempt int) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check if it's a rate limit error
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "Too Many Requests") {
		backoffDuration := time.Duration(attempt*attempt) * time.Second // Exponential backoff
		s.logger.Warn("Rate limit hit, backing off",
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoffDuration),
			zap.Error(err))
		time.Sleep(backoffDuration)
		return err
	}

	// Check if it's a timeout error
	if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "timeout") {
		backoffDuration := time.Duration(attempt) * 2 * time.Second // Linear backoff for timeouts
		s.logger.Warn("Request timeout, backing off",
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoffDuration),
			zap.Error(err))
		time.Sleep(backoffDuration)
		return err
	}

	return err
}
