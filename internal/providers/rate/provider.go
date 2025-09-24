package rate

import (
	"context"
	"fmt"

	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/domain"
	"go.uber.org/zap"
)

type Provider struct {
	config config.RateConfig
	logger *zap.Logger
}

func NewProvider(cfg config.RateConfig, logger *zap.Logger) *Provider {
	return &Provider{
		config: cfg,
		logger: logger.Named("rate"),
	}
}

func (p *Provider) Name() string {
	return "Rate"
}

func (p *Provider) Connect(ctx context.Context) error {
	p.logger.Info("Connecting to Rate exchange")
	// TODO: Implement actual connection logic
	return nil
}

func (p *Provider) Disconnect() error {
	p.logger.Info("Disconnecting from Rate exchange")
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
	return []string{"BTC/USD", "ETH/USD", "SOL/USD"}
}
