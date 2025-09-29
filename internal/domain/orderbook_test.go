package domain

import (
	"testing"
)

func TestNormalizeSymbol(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized underscore format",
			input:    "WFLI_USDT",
			expected: "WFLI_USDT",
		},
		{
			name:     "concatenated format with USDT",
			input:    "WFLIUSDT",
			expected: "WFLI_USDT",
		},
		{
			name:     "concatenated format with USDC",
			input:    "BTCUSDC",
			expected: "BTC_USDC",
		},
		{
			name:     "concatenated format with BTC",
			input:    "ETHBTC",
			expected: "ETH_BTC",
		},
		{
			name:     "lowercase input",
			input:    "wfliusdt",
			expected: "WFLI_USDT",
		},
		{
			name:     "mixed case underscore format",
			input:    "wfli_usdt",
			expected: "WFLI_USDT",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "unknown quote currency",
			input:    "ABCXYZ",
			expected: "ABCXYZ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSymbol(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSymbol(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToExchangeFormat(t *testing.T) {
	tests := []struct {
		name           string
		normalizedSymbol string
		exchange       string
		expected       string
	}{
		{
			name:           "bybit format",
			normalizedSymbol: "WFLI_USDT",
			exchange:       "bybit",
			expected:       "WFLIUSDT",
		},
		{
			name:           "gate format",
			normalizedSymbol: "WFLI_USDT",
			exchange:       "gate",
			expected:       "WFLI_USDT",
		},
		{
			name:           "unknown exchange defaults to normalized",
			normalizedSymbol: "WFLI_USDT",
			exchange:       "unknown",
			expected:       "WFLI_USDT",
		},
		{
			name:           "case insensitive exchange",
			normalizedSymbol: "BTC_USDT",
			exchange:       "BYBIT",
			expected:       "BTCUSDT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToExchangeFormat(tt.normalizedSymbol, tt.exchange)
			if result != tt.expected {
				t.Errorf("ToExchangeFormat(%q, %q) = %q, expected %q", 
					tt.normalizedSymbol, tt.exchange, result, tt.expected)
			}
		})
	}
}