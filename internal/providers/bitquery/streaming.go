package bitquery

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Connect to Bitquery streaming API
func (p *Provider) connectToStreaming(ctx context.Context) error {
	p.logger.Info("Creating mock DEX data for Bitquery provider")
	
	// Create mock data immediately (synchronous)
	p.createMockOrderBooks()
	
	// Start background updates
	go p.updateMockDataPeriodically(ctx)
	return nil
}

// Update mock data periodically
func (p *Provider) updateMockDataPeriodically(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.updateMockOrderBooks()
		}
	}
}

// Create mock order books for supported tokens
func (p *Provider) createMockOrderBooks() {
	mockPrices := map[string]float64{
		"BONK_USDT": 0.00002451,
		"PUMP_USDT": 0.8234,
		"TRUMP_USDT": 12.45,
		"JUP_USDT": 0.9876,
		"WLFI_USDT": 15.67,
	}


	p.tradeMutex.Lock()
	defer p.tradeMutex.Unlock()

	for _, symbol := range p.supportedTokens {
		if price, exists := mockPrices[symbol]; exists {
			orderBook := p.createSyntheticOrderBook(symbol, price)
			p.orderBookCache[symbol] = orderBook
			
			p.logger.Info("Created mock order book", 
				zap.String("symbol", symbol),
				zap.Float64("price", price))
		} else {
			p.logger.Warn("No mock price for symbol", zap.String("symbol", symbol))
		}
	}
	
	p.logger.Info("Mock order book creation completed", 
		zap.Int("total_books", len(p.orderBookCache)))
}

// Update mock order books with small price variations
func (p *Provider) updateMockOrderBooks() {
	p.tradeMutex.Lock()
	defer p.tradeMutex.Unlock()

	for symbol, orderBook := range p.orderBookCache {
		if len(orderBook.Bids) > 0 && len(orderBook.Asks) > 0 {
			// Get current mid price
			midPrice := (orderBook.Bids[0].Price + orderBook.Asks[0].Price) / 2
			
			// Add small random variation (-1% to +1%)
			variation := (float64(time.Now().UnixNano()%200) - 100) / 10000 // -1% to +1%
			newPrice := midPrice * (1 + variation)
			
			// Update order book
			newOrderBook := p.createSyntheticOrderBook(symbol, newPrice)
			orderBook.Bids = newOrderBook.Bids
			orderBook.Asks = newOrderBook.Asks
			orderBook.Timestamp = newOrderBook.Timestamp
			
			p.logger.Debug("Updated mock order book",
				zap.String("symbol", symbol),
				zap.Float64("old_price", midPrice),
				zap.Float64("new_price", newPrice),
				zap.Float64("variation", variation*100))
		}
	}
}