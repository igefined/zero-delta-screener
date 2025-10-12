package domain

import (
	"fmt"
	"strings"
	"time"
)

type OrderBook struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Bids      []Order   `json:"bids"`
	Asks      []Order   `json:"asks"`
}

type Order struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

func (o *Order) String() string {
	return fmt.Sprintf("Price: %.8f, Volume: %.8f", o.Price, o.Volume)
}

type Asset struct {
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
}

type ArbitrageOpportunity struct {
	Symbol         string    `json:"symbol"`
	BuyExchange    string    `json:"buy_exchange"`
	SellExchange   string    `json:"sell_exchange"`
	BuyPrice       float64   `json:"buy_price"`
	SellPrice      float64   `json:"sell_price"`
	PriceDiff      float64   `json:"price_diff"`
	ProfitPercent  float64   `json:"profit_percent"`
	Volume         float64   `json:"volume"`
	Timestamp      time.Time `json:"timestamp"`
	OpportunityType string   `json:"opportunity_type"` // "CEX-CEX", "CEX-DEX", "DEX-DEX"
}

// NormalizeSymbol converts various symbol formats to a standardized underscore format.
// Examples: "WFLIUSDT" -> "WFLI_USDT", "WFLI_USDT" -> "WFLI_USDT"
func NormalizeSymbol(symbol string) string {
	if symbol == "" {
		return symbol
	}
	
	// If already in underscore format, return as-is
	if strings.Contains(symbol, "_") {
		return strings.ToUpper(symbol)
	}
	
	// Convert from concatenated format to underscore format
	// Look for common quote currencies (USDT, USDC, BTC, ETH, etc.)
	quoteCurrencies := []string{"USDT", "USDC", "BTC", "ETH", "BNB", "DAI", "BUSD"}
	
	symbolUpper := strings.ToUpper(symbol)
	for _, quote := range quoteCurrencies {
		if strings.HasSuffix(symbolUpper, quote) {
			base := symbolUpper[:len(symbolUpper)-len(quote)]
			if base != "" {
				return base + "_" + quote
			}
		}
	}
	
	// If no known quote currency found, return as-is
	return symbolUpper
}

// ToExchangeFormat converts a normalized symbol to exchange-specific format
func ToExchangeFormat(normalizedSymbol, exchange string) string {
	switch strings.ToLower(exchange) {
	case "bybit":
		// ByBit uses concatenated format: WFLI_USDT -> WFLIUSDT
		return strings.ReplaceAll(normalizedSymbol, "_", "")
	case "gate":
		// Gate uses underscore format: WFLI_USDT -> WFLI_USDT
		return normalizedSymbol
	default:
		// Default to normalized format
		return normalizedSymbol
	}
}

// IsDEX returns true if the exchange is a decentralized exchange
func IsDEX(exchange string) bool {
	dexExchanges := []string{"bitquery", "uniswap", "sushiswap", "pancakeswap", "raydium", "orca", "jupiter"}
	exchangeLower := strings.ToLower(exchange)
	
	for _, dex := range dexExchanges {
		if exchangeLower == dex {
			return true
		}
	}
	return false
}

// GetOpportunityType determines the type of arbitrage opportunity
func GetOpportunityType(buyExchange, sellExchange string) string {
	buyIsDEX := IsDEX(buyExchange)
	sellIsDEX := IsDEX(sellExchange)
	
	if buyIsDEX && sellIsDEX {
		return "DEX-DEX"
	} else if !buyIsDEX && !sellIsDEX {
		return "CEX-CEX"
	} else {
		return "CEX-DEX"
	}
}
