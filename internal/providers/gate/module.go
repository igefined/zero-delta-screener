package gate

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/domain"
)

const moduleName = "Gate"

var Module = fx.Module(moduleName,
	fx.Provide(
		fx.Annotate(
			func(cfg *config.Config, logger *zap.Logger) domain.Provider {
				return NewProvider(cfg.Gate, cfg.SupportedTokens, logger)
			},
			fx.ResultTags(`group:"providers"`),
		),
	),
)
