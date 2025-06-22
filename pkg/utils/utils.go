package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// FormatDuration formats duration in human readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}

// FormatBytes formats bytes in human readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ValidateEthereumAddress validates Ethereum address format
func ValidateEthereumAddress(address string) bool {
	if !strings.HasPrefix(address, "0x") {
		return false
	}

	// Remove 0x prefix
	addr := address[2:]

	// Check length (20 bytes = 40 hex characters)
	if len(addr) != 40 {
		return false
	}

	// Check if all characters are valid hex
	_, err := hex.DecodeString(addr)
	return err == nil
}

// ValidateEthereumHash validates Ethereum hash format (32 bytes)
func ValidateEthereumHash(hash string) bool {
	if !strings.HasPrefix(hash, "0x") {
		return false
	}

	// Remove 0x prefix
	h := hash[2:]

	// Check length (32 bytes = 64 hex characters)
	if len(h) != 64 {
		return false
	}

	// Check if all characters are valid hex
	_, err := hex.DecodeString(h)
	return err == nil
}

// BigIntToString safely converts big.Int to string
func BigIntToString(bi *big.Int) string {
	if bi == nil {
		return "0"
	}
	return bi.String()
}

// StringToBigInt safely converts string to big.Int
func StringToBigInt(s string) *big.Int {
	bi, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return bi
}

// WeiToEther converts wei to ether
func WeiToEther(wei *big.Int) *big.Float {
	if wei == nil {
		return big.NewFloat(0)
	}

	weiFloat := new(big.Float).SetInt(wei)
	etherUnit := new(big.Float).SetInt(big.NewInt(1e18))

	return new(big.Float).Quo(weiFloat, etherUnit)
}

// EtherToWei converts ether to wei
func EtherToWei(ether *big.Float) *big.Int {
	if ether == nil {
		return big.NewInt(0)
	}

	etherUnit := new(big.Float).SetInt(big.NewInt(1e18))
	wei := new(big.Float).Mul(ether, etherUnit)

	result, _ := wei.Int(nil)
	return result
}

// CalculatePercentage calculates percentage
func CalculatePercentage(part, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return (float64(part) / float64(total)) * 100
}

// TruncateString truncates string to specified length
func TruncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// MinInt returns minimum of two integers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns maximum of two integers
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinUint64 returns minimum of two uint64
func MinUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// MaxUint64 returns maximum of two uint64
func MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
