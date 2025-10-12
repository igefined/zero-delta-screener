package bitquery

import (
	"time"
)

// DexTrade represents a DEX trade from Bitquery CoreCast
type DexTrade struct {
	Signature     string    `json:"signature"`
	BlockSlot     int64     `json:"block_slot"`
	BlockTime     time.Time `json:"block_time"`
	ProgramID     string    `json:"program_id"`
	BaseToken     TokenInfo `json:"base_token"`
	QuoteToken    TokenInfo `json:"quote_token"`
	BaseAmount    float64   `json:"base_amount"`
	QuoteAmount   float64   `json:"quote_amount"`
	Price         float64   `json:"price"`
	Trader        string    `json:"trader"`
	Market        string    `json:"market"`
	Side          string    `json:"side"` // "buy" or "sell"
	DexName       string    `json:"dex_name"`
}

// TokenInfo represents token metadata
type TokenInfo struct {
	Mint     string `json:"mint"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
}

// DexTradeRequest represents the subscription request
type DexTradeRequest struct {
	Markets      []string `json:"markets,omitempty"`
	Tokens       []string `json:"tokens,omitempty"`
	Programs     []string `json:"programs,omitempty"`
	MinVolumeUSD float64  `json:"min_volume_usd,omitempty"`
}

// CoreCastResponse represents the gRPC response wrapper
type CoreCastResponse struct {
	Data   *DexTrade `json:"data"`
	Error  string    `json:"error,omitempty"`
	Status string    `json:"status"`
}