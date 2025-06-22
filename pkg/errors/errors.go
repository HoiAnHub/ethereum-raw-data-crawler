package errors

import (
	"fmt"
)

// CrawlerError represents base crawler error
type CrawlerError struct {
	Code    string
	Message string
	Cause   error
}

func (e *CrawlerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *CrawlerError) Unwrap() error {
	return e.Cause
}

// Error codes
const (
	ErrCodeBlockchainConnection  = "BLOCKCHAIN_CONNECTION"
	ErrCodeDatabaseConnection    = "DATABASE_CONNECTION"
	ErrCodeBlockProcessing       = "BLOCK_PROCESSING"
	ErrCodeTransactionProcessing = "TRANSACTION_PROCESSING"
	ErrCodeConfiguration         = "CONFIGURATION"
	ErrCodeValidation            = "VALIDATION"
)

// NewBlockchainConnectionError creates blockchain connection error
func NewBlockchainConnectionError(message string, cause error) *CrawlerError {
	return &CrawlerError{
		Code:    ErrCodeBlockchainConnection,
		Message: message,
		Cause:   cause,
	}
}

// NewDatabaseConnectionError creates database connection error
func NewDatabaseConnectionError(message string, cause error) *CrawlerError {
	return &CrawlerError{
		Code:    ErrCodeDatabaseConnection,
		Message: message,
		Cause:   cause,
	}
}

// NewBlockProcessingError creates block processing error
func NewBlockProcessingError(blockNumber string, cause error) *CrawlerError {
	return &CrawlerError{
		Code:    ErrCodeBlockProcessing,
		Message: fmt.Sprintf("failed to process block %s", blockNumber),
		Cause:   cause,
	}
}

// NewTransactionProcessingError creates transaction processing error
func NewTransactionProcessingError(txHash string, cause error) *CrawlerError {
	return &CrawlerError{
		Code:    ErrCodeTransactionProcessing,
		Message: fmt.Sprintf("failed to process transaction %s", txHash),
		Cause:   cause,
	}
}

// NewConfigurationError creates configuration error
func NewConfigurationError(message string, cause error) *CrawlerError {
	return &CrawlerError{
		Code:    ErrCodeConfiguration,
		Message: message,
		Cause:   cause,
	}
}

// NewValidationError creates validation error
func NewValidationError(message string, cause error) *CrawlerError {
	return &CrawlerError{
		Code:    ErrCodeValidation,
		Message: message,
		Cause:   cause,
	}
}
