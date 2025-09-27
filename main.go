package main

import (
	"go.uber.org/fx"

	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/providers/gate"
	"github.com/igefined/zero-delta-screener/internal/screener"
	"github.com/igefined/zero-delta-screener/pkg/logger"
)

func main() {
	fx.New(
		config.Module,
		logger.Module,
		// Provider modules
		gate.Module,
		// bybit.Module,
		// Business logic modules
		screener.Module,
	).Run()
}
