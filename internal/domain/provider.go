package domain

import (
	"context"
)

type Provider interface {
	Name() string
	FetchOrderBook(ctx context.Context, symbol string) (*OrderBook, error)
	GetSupportedSymbols() []string
	Connect(ctx context.Context) error
	Disconnect() error
}
