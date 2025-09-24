package main

import (
	"go.uber.org/fx"

	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/providers/bybit"
	"github.com/igefined/zero-delta-screener/internal/providers/rate"
	"github.com/igefined/zero-delta-screener/internal/screener"
	"github.com/igefined/zero-delta-screener/pkg/logger"
)

func main() {
	fx.New(
		config.Module,
		logger.Module,
		// Provider modules
		rate.Module,
		bybit.Module,
		// Business logic modules
		screener.Module,
	).Run()
}
