package domain

import (
	"context"
)

type Provider interface {
	Name() string
	FetchOrderBook(symbol string) (*OrderBook, error)
	FetchOrderBooks(symbols []string) (map[string]*OrderBook, error)
	GetSupportedSymbols() []string
	Connect(ctx context.Context) error
	Disconnect() error
}
