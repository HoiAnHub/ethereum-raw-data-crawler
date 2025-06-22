package utils

import (
	"math/big"
	"testing"
	"time"
)

func TestValidateEthereumAddress(t *testing.T) {
	tests := []struct {
		address string
		valid   bool
	}{
		{"0x742d35Cc6e56A0e24C1D887FC9b50f08a2B6F4bC", true},
		{"0x0000000000000000000000000000000000000000", true},
		{"742d35Cc6e56A0e24C1D887FC9b50f08a2B6F4bC", false},    // No 0x prefix
		{"0x742d35Cc6e56A0e24C1D887FC9b50f08a2B6F4b", false},   // Too short
		{"0x742d35Cc6e56A0e24C1D887FC9b50f08a2B6F4bCC", false}, // Too long
		{"0xGGGd35Cc6e56A0e24C1D887FC9b50f08a2B6F4bC", false},  // Invalid hex
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			result := ValidateEthereumAddress(tt.address)
			if result != tt.valid {
				t.Errorf("ValidateEthereumAddress(%s) = %v, want %v", tt.address, result, tt.valid)
			}
		})
	}
}

func TestValidateEthereumHash(t *testing.T) {
	tests := []struct {
		hash  string
		valid bool
	}{
		{"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", true},
		{"0x0000000000000000000000000000000000000000000000000000000000000000", true},
		{"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},   // No 0x prefix
		{"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd", false},   // Too short
		{"0xGGGG567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false}, // Invalid hex
	}

	for _, tt := range tests {
		t.Run(tt.hash, func(t *testing.T) {
			result := ValidateEthereumHash(tt.hash)
			if result != tt.valid {
				t.Errorf("ValidateEthereumHash(%s) = %v, want %v", tt.hash, result, tt.valid)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1.5m"},
		{3 * time.Hour, "3.0h"},
		{25 * time.Hour, "1.0d"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestBigIntToString(t *testing.T) {
	tests := []struct {
		input    *big.Int
		expected string
	}{
		{big.NewInt(123), "123"},
		{big.NewInt(0), "0"},
		{nil, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := BigIntToString(tt.input)
			if result != tt.expected {
				t.Errorf("BigIntToString(%v) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringToBigInt(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"0", "0"},
		{"invalid", "0"},
		{"", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StringToBigInt(tt.input)
			if result.String() != tt.expected {
				t.Errorf("StringToBigInt(%s) = %s, want %s", tt.input, result.String(), tt.expected)
			}
		})
	}
}

func TestWeiToEther(t *testing.T) {
	tests := []struct {
		wei      *big.Int
		expected float64
	}{
		{big.NewInt(1000000000000000000), 1.0}, // 1 ether
		{big.NewInt(500000000000000000), 0.5},  // 0.5 ether
		{big.NewInt(0), 0.0},                   // 0 ether
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := WeiToEther(tt.wei)
			resultFloat, _ := result.Float64()
			if resultFloat != tt.expected {
				t.Errorf("WeiToEther(%v) = %f, want %f", tt.wei, resultFloat, tt.expected)
			}
		})
	}
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		part      uint64
		total     uint64
		expected  float64
		tolerance float64
	}{
		{50, 100, 50.0, 0.0001},
		{1, 3, 33.333333333333336, 0.0001},
		{0, 100, 0.0, 0.0001},
		{100, 0, 0.0, 0.0001}, // Division by zero case
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := CalculatePercentage(tt.part, tt.total)
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("CalculatePercentage(%d, %d) = %f, want %f (within %f)", tt.part, tt.total, result, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"hello world", 5, "hello..."},
		{"short", 10, "short"},
		{"exact", 5, "exact"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TruncateString(tt.input, tt.length)
			if result != tt.expected {
				t.Errorf("TruncateString(%s, %d) = %s, want %s", tt.input, tt.length, result, tt.expected)
			}
		})
	}
}

func TestMinMaxFunctions(t *testing.T) {
	// Test MinInt and MaxInt
	if MinInt(5, 3) != 3 {
		t.Errorf("MinInt(5, 3) = %d, want 3", MinInt(5, 3))
	}
	if MaxInt(5, 3) != 5 {
		t.Errorf("MaxInt(5, 3) = %d, want 5", MaxInt(5, 3))
	}

	// Test MinUint64 and MaxUint64
	if MinUint64(5, 3) != 3 {
		t.Errorf("MinUint64(5, 3) = %d, want 3", MinUint64(5, 3))
	}
	if MaxUint64(5, 3) != 5 {
		t.Errorf("MaxUint64(5, 3) = %d, want 5", MaxUint64(5, 3))
	}
}
