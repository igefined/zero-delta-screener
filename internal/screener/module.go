package screener

import (
	"context"

	"go.uber.org/fx"
)

const moduleName = "screener"

var Module = fx.Module(moduleName,
	fx.Provide(NewService),
	fx.Invoke(func(lc fx.Lifecycle, service *Service) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return service.Start()
			},
			OnStop: func(ctx context.Context) error {
				return service.Stop()
			},
		})
	}),
)
