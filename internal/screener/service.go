package screener

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/igefined/zero-delta-screener/internal/domain"
)

const bufferSize = 100

type Service struct {
	providers        []domain.Provider
	logger           *zap.Logger
	interval         time.Duration
	stopCh           chan struct{}
	wg               sync.WaitGroup
	orderBookCache   map[string]map[string]*domain.OrderBook // symbol -> exchange -> orderbook
	cacheMutex       sync.RWMutex
	minProfitPercent float64
	executeSwaps     bool // Flag to enable/disable actual swap execution
}

type Params struct {
	fx.In

	Providers []domain.Provider `group:"providers"`
	Logger    *zap.Logger
}

func NewService(params Params) *Service {
	return &Service{
		providers:        params.Providers,
		logger:           params.Logger.Named("screener"),
		interval:         time.Second,
		stopCh:           make(chan struct{}),
		orderBookCache:   make(map[string]map[string]*domain.OrderBook),
		minProfitPercent: 0.5,   // 0.5% minimum profit threshold
		executeSwaps:     false, // Set to true to enable actual swap execution
	}
}

func (s *Service) Start() error {
	s.logger.Info("Starting screener service", zap.Int("providers_count", len(s.providers)))

	for _, provider := range s.providers {
		if err := provider.Connect(s.stopCh); err != nil {
			s.logger.Error("Failed to connect to provider",
				zap.String("provider", provider.Name()),
				zap.Error(err))
			return err
		}
		s.logger.Info("Connected to provider", zap.String("provider", provider.Name()))
	}

	go s.runContinuousScreening()

	return nil
}

func (s *Service) Stop() error {
	s.logger.Info("Stopping screener service")

	close(s.stopCh)
	s.wg.Wait()

	for _, provider := range s.providers {
		if err := provider.Disconnect(); err != nil {
			s.logger.Error("Failed to disconnect from provider",
				zap.String("provider", provider.Name()),
				zap.Error(err))
		}
	}

	return nil
}

func (s *Service) runContinuousScreening() {
	s.wg.Add(1)
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			s.logger.Info("Stopping continuous screening")
			return
		case <-ticker.C:
			s.screen()
		}
	}
}

func (s *Service) screen() {
	s.logger.Debug("Starting screening cycle")

	var wg sync.WaitGroup
	resultsCh := make(chan *domain.OrderBook, len(s.providers)*bufferSize)

	for _, provider := range s.providers {
		symbols := provider.GetSupportedSymbols()
		for _, symbol := range symbols {
			wg.Add(1)
			go func(p domain.Provider, sym string) {
				defer wg.Done()

				orderBook, err := p.FetchOrderBook(sym)
				if err != nil {
					s.logger.Error("Failed to fetch order book",
						zap.String("provider", p.Name()),
						zap.String("symbol", sym),
						zap.Error(err))
					return
				}

				if orderBook != nil {
					select {
					case resultsCh <- orderBook:
					default:
						s.logger.Debug("Skipped orderBook")
					}
				}
			}(provider, symbol)
		}
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for orderBook := range resultsCh {
		s.processOrderBook(orderBook)
	}

	s.logger.Debug("Screening cycle completed")
}

func (s *Service) processOrderBook(orderBook *domain.OrderBook) {
	s.updateCache(orderBook)
	s.checkArbitrageOpportunities(orderBook.Symbol)
}

func (s *Service) updateCache(orderBook *domain.OrderBook) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if s.orderBookCache[orderBook.Symbol] == nil {
		s.orderBookCache[orderBook.Symbol] = make(map[string]*domain.OrderBook)
	}
	s.orderBookCache[orderBook.Symbol][orderBook.Exchange] = orderBook
}

func (s *Service) checkArbitrageOpportunities(symbol string) {
	s.cacheMutex.RLock()
	exchangeBooks := s.orderBookCache[symbol]
	s.cacheMutex.RUnlock()

	if len(exchangeBooks) < 2 {
		return
	}

	var orderBooks []*domain.OrderBook
	for _, book := range exchangeBooks {
		if len(book.Bids) > 0 && len(book.Asks) > 0 {
			orderBooks = append(orderBooks, book)
		}
	}

	for i := 0; i < len(orderBooks); i++ {
		for j := i + 1; j < len(orderBooks); j++ {
			s.compareOrderBooks(orderBooks[i], orderBooks[j])
		}
	}
}

func (s *Service) compareOrderBooks(book1, book2 *domain.OrderBook) {
	// Only process CEX-DEX opportunities
	if opportunity := s.calculateOpportunity(book1, book2, true); opportunity != nil && opportunity.OpportunityType == "CEX-DEX" {
		s.logArbitrageOpportunity(opportunity)
	}

	if opportunity := s.calculateOpportunity(book2, book1, true); opportunity != nil && opportunity.OpportunityType == "CEX-DEX" {
		s.logArbitrageOpportunity(opportunity)
	}
}

func (s *Service) calculateOpportunity(buyBook, sellBook *domain.OrderBook, checkProfit bool) *domain.ArbitrageOpportunity {
	if len(buyBook.Asks) == 0 || len(sellBook.Bids) == 0 {
		return nil
	}

	bestAsk := buyBook.Asks[0]
	bestBid := sellBook.Bids[0]

	priceDiff := bestBid.Price - bestAsk.Price
	if priceDiff <= 0 {
		return nil
	}

	profitPercent := (priceDiff / bestAsk.Price) * 100

	if checkProfit && profitPercent < s.minProfitPercent {
		return nil
	}

	volume := bestAsk.Volume
	if bestBid.Volume < volume {
		volume = bestBid.Volume
	}

	return &domain.ArbitrageOpportunity{
		Symbol:          buyBook.Symbol,
		BuyExchange:     buyBook.Exchange,
		SellExchange:    sellBook.Exchange,
		BuyPrice:        bestAsk.Price,
		SellPrice:       bestBid.Price,
		PriceDiff:       priceDiff,
		ProfitPercent:   profitPercent,
		Volume:          volume,
		Timestamp:       time.Now(),
		OpportunityType: domain.GetOpportunityType(buyBook.Exchange, sellBook.Exchange),
	}
}

func (s *Service) logArbitrageOpportunity(opportunity *domain.ArbitrageOpportunity) {
	// Use different emojis and log levels based on opportunity type
	var emoji, message string
	logLevel := zap.InfoLevel

	switch opportunity.OpportunityType {
	case "CEX-DEX":
		emoji = "🌊"
		message = "CEX-DEX ARBITRAGE OPPORTUNITY FOUND!"
		logLevel = zap.WarnLevel // Higher priority for CEX-DEX opportunities
	case "DEX-DEX":
		emoji = "🔄"
		message = "DEX-DEX ARBITRAGE OPPORTUNITY FOUND!"
	default: // CEX-CEX
		emoji = "🚀"
		message = "CEX-CEX ARBITRAGE OPPORTUNITY FOUND!"
	}

	fields := []zap.Field{
		zap.String("opportunity_type", opportunity.OpportunityType),
		zap.String("symbol", opportunity.Symbol),
		zap.String("buy_from", opportunity.BuyExchange),
		zap.String("sell_to", opportunity.SellExchange),
		zap.Float64("buy_price", opportunity.BuyPrice),
		zap.Float64("sell_price", opportunity.SellPrice),
		zap.Float64("price_diff", opportunity.PriceDiff),
		zap.Float64("profit_percent", opportunity.ProfitPercent),
		zap.Float64("volume", opportunity.Volume),
		zap.Time("timestamp", opportunity.Timestamp),
	}

	// Add special considerations for CEX-DEX opportunities
	if opportunity.OpportunityType == "CEX-DEX" {
		fields = append(fields,
			zap.String("note", "Consider gas fees, slippage, and transaction time for DEX trades"),
			zap.Bool("cross_infrastructure", true))
	}

	s.logger.Log(logLevel, emoji+" "+message, fields...)

	// Execute swap if enabled
	if s.executeSwaps {
		s.executeArbitrageSwap(opportunity)
	}
}

// executeArbitrageSwap executes the arbitrage opportunity with timing measurement
func (s *Service) executeArbitrageSwap(opportunity *domain.ArbitrageOpportunity) {
	requestID := fmt.Sprintf("arb_%d", time.Now().UnixNano())
	requestTime := time.Now()

	// Create buy swap request
	buyRequest := &domain.SwapRequest{
		Symbol:      opportunity.Symbol,
		Side:        "buy",
		Amount:      opportunity.Volume,
		Price:       opportunity.BuyPrice,
		OrderType:   "limit",
		RequestID:   requestID + "_buy",
		RequestTime: requestTime,
	}

	// Create sell swap request
	sellRequest := &domain.SwapRequest{
		Symbol:      opportunity.Symbol,
		Side:        "sell",
		Amount:      opportunity.Volume,
		Price:       opportunity.SellPrice,
		OrderType:   "limit",
		RequestID:   requestID + "_sell",
		RequestTime: requestTime,
	}

	s.logger.Info("Executing arbitrage swap",
		zap.String("opportunity_type", opportunity.OpportunityType),
		zap.String("buy_request_id", buyRequest.RequestID),
		zap.String("sell_request_id", sellRequest.RequestID),
		zap.String("symbol", opportunity.Symbol),
		zap.Float64("expected_profit_percent", opportunity.ProfitPercent))

}

// EnableSwapExecution enables or disables actual swap execution
func (s *Service) EnableSwapExecution(enable bool) {
	s.executeSwaps = enable
	s.logger.Info("Swap execution status changed", zap.Bool("enabled", enable))
}
