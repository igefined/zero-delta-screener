package bitquery

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/domain"
)

type Provider struct {
	config          config.BitqueryConfig
	supportedTokens []string
	logger          *zap.Logger

	// gRPC connection
	conn *grpc.ClientConn

	// Order book construction from trades
	orderBookCache map[string]*domain.OrderBook
	tradeMutex     sync.RWMutex

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc

	// Market maker simulation parameters
	spreadPercent float64
	depthLevels   int
}

func (p *Provider) Name() string {
	return moduleName
}

func (p *Provider) Connect(closeCh <-chan struct{}) error {
	p.logger.Info("Connecting to Bitquery CoreCast gRPC")

	// Create context with cancellation
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Establish gRPC connection
	conn, err := grpc.NewClient(
		p.config.GrpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Bitquery gRPC: %w", err)
	}

	p.conn = conn

	// Start streaming DEX trades
	go p.streamDexTrades(closeCh)

	p.logger.Info("Connected to Bitquery CoreCast gRPC")
	return nil
}

func (p *Provider) Disconnect() error {
	p.logger.Info("Disconnecting from Bitquery CoreCast")

	if p.cancel != nil {
		p.cancel()
	}

	if p.conn != nil {
		return p.conn.Close()
	}

	return nil
}

func (p *Provider) FetchOrderBook(symbol string) (*domain.OrderBook, error) {
	start := time.Now()
	p.logger.Debug("Fetching order book", zap.String("symbol", symbol))

	normalizedSymbol := domain.NormalizeSymbol(symbol)

	p.tradeMutex.RLock()
	orderBook, exists := p.orderBookCache[normalizedSymbol]
	p.tradeMutex.RUnlock()

	if !exists {
		elapsed := time.Since(start)
		p.logger.Warn("No order book data available",
			zap.String("symbol", symbol),
			zap.Duration("execution_time", elapsed))
		return nil, fmt.Errorf("no order book data available for symbol %s", symbol)
	}

	// Return a copy to avoid race conditions
	result := p.copyOrderBook(orderBook)
	elapsed := time.Since(start)

	p.logger.Info("Order book fetched successfully",
		zap.String("symbol", symbol),
		zap.Duration("execution_time", elapsed),
		zap.Time("data_timestamp", orderBook.Timestamp),
		zap.Duration("data_age", time.Since(orderBook.Timestamp)))

	return result, nil
}

func (p *Provider) FetchOrderBooks(symbols []string) (map[string]*domain.OrderBook, error) {
	start := time.Now()
	p.logger.Info("Fetching multiple order books", zap.Strings("symbols", symbols))

	results := make(map[string]*domain.OrderBook)

	p.tradeMutex.RLock()
	for _, symbol := range symbols {
		normalizedSymbol := domain.NormalizeSymbol(symbol)
		if orderBook, exists := p.orderBookCache[normalizedSymbol]; exists {
			results[normalizedSymbol] = p.copyOrderBook(orderBook)
		}
	}
	p.tradeMutex.RUnlock()

	elapsed := time.Since(start)
	p.logger.Info("Successfully fetched order books",
		zap.Int("requested", len(symbols)),
		zap.Int("found", len(results)),
		zap.Duration("execution_time", elapsed))

	return results, nil
}

func (p *Provider) GetSupportedSymbols() []string {
	return p.supportedTokens
}

func (p *Provider) streamDexTrades(closeCh <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("Panic in DEX trade streaming", zap.Any("error", r))
		}
	}()

	// Create metadata with authentication
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + p.config.APIToken,
	})
	ctx := metadata.NewOutgoingContext(p.ctx, md)

	// Create streaming request
	req := &DexTradeRequest{
		Tokens:       p.supportedTokens,
		MinVolumeUSD: 100, // Minimum $100 volume to reduce noise
	}

	for {
		select {
		case <-closeCh:
			p.logger.Info("Stopping DEX trade streaming")
			return
		case <-p.ctx.Done():
			p.logger.Info("Context cancelled, stopping DEX trade streaming")
			return
		default:
			if err := p.connectAndStream(ctx, req); err != nil {
				p.logger.Error("DEX streaming error, retrying in 5s", zap.Error(err))

				select {
				case <-time.After(5 * time.Second):
					continue
				case <-closeCh:
					return
				case <-p.ctx.Done():
					return
				}
			}
		}
	}
}

func (p *Provider) connectAndStream(ctx context.Context, req *DexTradeRequest) error {
	// Note: Since we don't have the actual gRPC service implementation,
	// we'll simulate the streaming with a mock implementation
	// In a real implementation, this would be:
	// stream, err := p.client.StreamDexTrades(ctx, req)

	p.logger.Info("Starting DEX trade stream simulation")

	// Simulate streaming with periodic mock data
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Generate mock DEX trade data for testing
			if err := p.processMockTrade(); err != nil {
				p.logger.Error("Failed to process mock trade", zap.Error(err))
			}
		}
	}
}

func (p *Provider) processMockTrade() error {
	// Generate mock trade for testing purposes
	// In real implementation, this would process actual DexTrade messages
	for _, symbol := range p.supportedTokens {
		normalizedSymbol := domain.NormalizeSymbol(symbol)

		// Create synthetic order book based on mock trade data
		orderBook := p.createSyntheticOrderBook(normalizedSymbol, 1.0) // Mock price of 1.0

		p.tradeMutex.Lock()
		p.orderBookCache[normalizedSymbol] = orderBook
		p.tradeMutex.Unlock()

		p.logger.Debug("Updated synthetic order book",
			zap.String("symbol", normalizedSymbol),
			zap.Int("bids", len(orderBook.Bids)),
			zap.Int("asks", len(orderBook.Asks)))
	}

	return nil
}

func (p *Provider) createSyntheticOrderBook(symbol string, midPrice float64) *domain.OrderBook {
	orderBook := &domain.OrderBook{
		Exchange:  moduleName,
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids:      make([]domain.Order, 0, p.depthLevels),
		Asks:      make([]domain.Order, 0, p.depthLevels),
	}

	// Generate bid levels (below mid price)
	for i := 0; i < p.depthLevels; i++ {
		priceOffset := p.spreadPercent/2 + (float64(i) * p.spreadPercent * 0.1)
		price := midPrice * (1 - priceOffset)
		volume := 1000.0 / (1 + float64(i)*0.5) // Decreasing volume with distance

		orderBook.Bids = append(orderBook.Bids, domain.Order{
			Price:  price,
			Volume: volume,
		})
	}

	// Generate ask levels (above mid price)
	for i := 0; i < p.depthLevels; i++ {
		priceOffset := p.spreadPercent/2 + (float64(i) * p.spreadPercent * 0.1)
		price := midPrice * (1 + priceOffset)
		volume := 1000.0 / (1 + float64(i)*0.5) // Decreasing volume with distance

		orderBook.Asks = append(orderBook.Asks, domain.Order{
			Price:  price,
			Volume: volume,
		})
	}

	// Sort orders: bids descending (highest first), asks ascending (lowest first)
	sort.Slice(orderBook.Bids, func(i, j int) bool {
		return orderBook.Bids[i].Price > orderBook.Bids[j].Price
	})
	sort.Slice(orderBook.Asks, func(i, j int) bool {
		return orderBook.Asks[i].Price < orderBook.Asks[j].Price
	})

	return orderBook
}

func (p *Provider) copyOrderBook(original *domain.OrderBook) *domain.OrderBook {
	copy := &domain.OrderBook{
		Exchange:  original.Exchange,
		Symbol:    original.Symbol,
		Timestamp: original.Timestamp,
		Bids:      make([]domain.Order, len(original.Bids)),
		Asks:      make([]domain.Order, len(original.Asks)),
	}

	copy.Bids = append(copy.Bids, original.Bids...)
	copy.Asks = append(copy.Asks, original.Asks...)

	return copy
}
