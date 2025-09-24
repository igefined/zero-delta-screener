package screener

import (
	"context"
	"sync"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/igefined/zero-delta-screener/internal/domain"
)

type Service struct {
	providers []domain.Provider
	logger    *zap.Logger
	interval  time.Duration
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

type Params struct {
	fx.In

	Providers []domain.Provider `group:"providers"`
	Logger    *zap.Logger
}

func NewService(params Params) *Service {
	return &Service{
		providers: params.Providers,
		logger:    params.Logger.Named("screener"),
		interval:  10 * time.Second, // Default interval
		stopCh:    make(chan struct{}),
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("Starting screener service",
		zap.Int("providers_count", len(s.providers)))

	// Connect to all providers
	for _, provider := range s.providers {
		if err := provider.Connect(ctx); err != nil {
			s.logger.Error("Failed to connect to provider",
				zap.String("provider", provider.Name()),
				zap.Error(err))
			return err
		}
		s.logger.Info("Connected to provider",
			zap.String("provider", provider.Name()))
	}

	// Start screening loop
	s.wg.Add(1)
	go s.screeningLoop(ctx)

	return nil
}

func (s *Service) Stop() error {
	s.logger.Info("Stopping screener service")

	close(s.stopCh)
	s.wg.Wait()

	// Disconnect from all providers
	for _, provider := range s.providers {
		if err := provider.Disconnect(); err != nil {
			s.logger.Error("Failed to disconnect from provider",
				zap.String("provider", provider.Name()),
				zap.Error(err))
		}
	}

	return nil
}

func (s *Service) screeningLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.screen(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Context cancelled, stopping screening loop")
			return
		case <-s.stopCh:
			s.logger.Info("Stop signal received, stopping screening loop")
			return
		case <-ticker.C:
			s.screen(ctx)
		}
	}
}

func (s *Service) screen(ctx context.Context) {
	s.logger.Debug("Starting screening cycle")

	var wg sync.WaitGroup
	resultsCh := make(chan *domain.OrderBook, len(s.providers)*10)

	for _, provider := range s.providers {
		symbols := provider.GetSupportedSymbols()
		for _, symbol := range symbols {
			wg.Add(1)
			go func(p domain.Provider, sym string) {
				defer wg.Done()

				orderBook, err := p.FetchOrderBook(ctx, sym)
				if err != nil {
					s.logger.Error("Failed to fetch order book",
						zap.String("provider", p.Name()),
						zap.String("symbol", sym),
						zap.Error(err))
					return
				}

				if orderBook != nil {
					resultsCh <- orderBook
				}
			}(provider, symbol)
		}
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Process results
	for orderBook := range resultsCh {
		s.processOrderBook(orderBook)
	}

	s.logger.Debug("Screening cycle completed")
}

func (s *Service) processOrderBook(orderBook *domain.OrderBook) {
	// TODO: Implement your screening logic here
	// For now, just log the order book
	s.logger.Info("Processing order book",
		zap.String("exchange", orderBook.Exchange),
		zap.String("symbol", orderBook.Symbol),
		zap.Time("timestamp", orderBook.Timestamp),
		zap.Int("bids", len(orderBook.Bids)),
		zap.Int("asks", len(orderBook.Asks)))
}
