package bybit

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/domain"
)

type Provider struct {
	config config.ByBitConfig
	logger *zap.Logger
}

func NewProvider(cfg config.ByBitConfig, logger *zap.Logger) *Provider {
	return &Provider{
		config: cfg,
		logger: logger.Named("bybit"),
	}
}

func (p *Provider) Name() string {
	return "Bybit"
}

func (p *Provider) Connect(ctx context.Context) error {
	p.logger.Info("Connecting to Bybit exchange")
	// TODO: Implement actual WebSocket/REST connection logic
	return nil
}

func (p *Provider) Disconnect() error {
	p.logger.Info("Disconnecting from Bybit exchange")
	// TODO: Implement actual disconnection logic
	return nil
}

func (p *Provider) FetchOrderBook(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	p.logger.Info("Fetching order book", zap.String("symbol", symbol))

	// TODO: Implement actual API call
	// This is a placeholder implementation
	return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetSupportedSymbols() []string {
	// TODO: Fetch from API or config
	return []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
}
