package bitquery

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/domain"
)

const moduleName = "bitquery"

var Module = fx.Module(moduleName,
	fx.Provide(
		fx.Annotate(
			func(cfg *config.Config, logger *zap.Logger) domain.Provider {
				return NewProvider(cfg.Bitquery, cfg.SupportedTokens, logger)
			},
			fx.ResultTags(`group:"providers"`),
		),
	),
)

func NewProvider(cfg config.BitqueryConfig, supportedTokens []string, logger *zap.Logger) *Provider {
	return &Provider{
		config:          cfg,
		supportedTokens: supportedTokens,
		logger:          logger.Named(moduleName),
		orderBookCache:  make(map[string]*domain.OrderBook),
		spreadPercent:   0.001, // 0.1% spread for synthetic order book
		depthLevels:     10,    // 10 levels of market depth
	}
}