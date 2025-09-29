package gate

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/domain"
	"go.uber.org/zap"
)

type Provider struct {
	config          config.GateConfig
	supportedTokens []string
	logger          *zap.Logger
	conn            *websocket.Conn
	mu              sync.RWMutex
	writeMu         sync.Mutex
	orderBookCache  map[string]*domain.OrderBook
	subscriptions   map[string]bool
}

func NewProvider(cfg config.GateConfig, supportedTokens []string, logger *zap.Logger) *Provider {
	return &Provider{
		config:          cfg,
		supportedTokens: supportedTokens,
		logger:          logger.Named("gate"),
		orderBookCache:  make(map[string]*domain.OrderBook),
		subscriptions:   make(map[string]bool),
	}
}

func (p *Provider) Name() string {
	return moduleName
}

func (p *Provider) Connect(ctx context.Context) error {
	p.logger.Info("Connecting to Gate exchange")

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(p.config.WsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Gate websocket: %w", err)
	}

	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	go p.messageHandler(ctx)

	p.logger.Info("Connected to Gate exchange")
	return nil
}

func (p *Provider) Disconnect() error {
	p.logger.Info("Disconnecting from Gate exchange")

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		return p.conn.Close()
	}

	return nil
}

func (p *Provider) FetchOrderBook(symbol string) (*domain.OrderBook, error) {
	p.logger.Info("Fetching order book", zap.String("symbol", symbol))

	if err := p.subscribeToOrderbook(symbol); err != nil {
		return nil, fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.RLock()
			orderBook, exists := p.orderBookCache[symbol]
			p.mu.RUnlock()

			if exists {
				return orderBook, nil
			}
		}
	}
}

func (p *Provider) FetchOrderBooks(symbols []string) (map[string]*domain.OrderBook, error) {
	p.logger.Info("Fetching multiple order books", zap.Strings("symbols", symbols))

	results := make(map[string]*domain.OrderBook)
	errors := make(map[string]error)

	for _, symbol := range symbols {
		if err := p.subscribeToOrderbook(symbol); err != nil {
			errors[symbol] = fmt.Errorf("failed to subscribe to orderbook: %w", err)
			continue
		}
	}

	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	pendingSymbols := make(map[string]bool)
	for _, symbol := range symbols {
		if _, hasError := errors[symbol]; !hasError {
			pendingSymbols[symbol] = true
		}
	}

	for len(pendingSymbols) > 0 {
		select {
		case <-timeout:
			for symbol := range pendingSymbols {
				errors[symbol] = fmt.Errorf("timeout waiting for orderbook data")
			}
			return results, fmt.Errorf("timeout fetching orderbooks for %d symbols", len(pendingSymbols))
		case <-ticker.C:
			p.mu.RLock()
			for symbol := range pendingSymbols {
				if orderbook, exists := p.orderBookCache[symbol]; exists {
					results[symbol] = orderbook
					delete(pendingSymbols, symbol)
				}
			}
			p.mu.RUnlock()
		}
	}

	p.logger.Info("Successfully fetched order books",
		zap.Int("successful", len(results)),
		zap.Int("failed", len(errors)))

	return results, nil
}

func (p *Provider) GetSupportedSymbols() []string {
	return p.supportedTokens
}

func (p *Provider) genSign(channel, event string, timestamp int64) map[string]string {
	signatureString := fmt.Sprintf("channel=%s&event=%s&time=%d", channel, event, timestamp)

	h := hmac.New(sha512.New, []byte(p.config.APISecret))
	h.Write([]byte(signatureString))
	signature := hex.EncodeToString(h.Sum(nil))

	return map[string]string{
		"method": "api_key",
		"KEY":    p.config.APIKey,
		"SIGN":   signature,
	}
}

func (p *Provider) subscribeToOrderbook(symbol string) error {
	// Check if already subscribed to avoid duplicate subscriptions
	p.mu.RLock()
	conn := p.conn
	alreadySubscribed := p.subscriptions[symbol]
	p.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	if alreadySubscribed {
		return nil // Already subscribed, no need to subscribe again
	}

	timestamp := time.Now().Unix()
	channel := "spot.order_book_update"
	event := "subscribe"
	payload := []string{symbol, "100ms"}

	request := map[string]interface{}{
		"time":    timestamp,
		"channel": channel,
		"event":   event,
		"payload": payload,
	}

	if p.config.APIKey != "" && p.config.APISecret != "" {
		request["auth"] = p.genSign(channel, event, timestamp)
	}

	msg, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription request: %w", err)
	}

	// Use write mutex to prevent concurrent writes to websocket
	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	p.logger.Info("Subscribing to orderbook", zap.String("symbol", symbol))
	if err = conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return fmt.Errorf("failed to send subscription request: %w", err)
	}

	// Mark as subscribed
	p.mu.Lock()
	p.subscriptions[symbol] = true
	p.mu.Unlock()

	return nil
}

type OrderBookResponse struct {
	Time    int64         `json:"time"`
	Channel string        `json:"channel"`
	Event   string        `json:"event"`
	Result  OrderbookData `json:"result"`
}

type OrderbookData struct {
	S string     `json:"s"`
	U int64      `json:"u"`
	B [][]string `json:"b"`
	A [][]string `json:"a"`
}

func (p *Provider) messageHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			p.mu.RLock()
			conn := p.conn
			p.mu.RUnlock()

			if conn == nil {
				return
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				p.logger.Error("Failed to read message", zap.Error(err))
				return
			}

			p.processMessage(message)
		}
	}
}

func (p *Provider) processMessage(message []byte) {
	// First try to parse as a general response to check for errors
	var genericResponse struct {
		Time    int64       `json:"time"`
		Channel string      `json:"channel"`
		Event   string      `json:"event"`
		Error   interface{} `json:"error"`
		Result  interface{} `json:"result"`
	}

	if err := json.Unmarshal(message, &genericResponse); err != nil {
		p.logger.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	// Log all messages for debugging
	p.logger.Info("Received WebSocket message",
		zap.String("channel", genericResponse.Channel),
		zap.String("event", genericResponse.Event),
		zap.Int64("time", genericResponse.Time),
		zap.Any("result", genericResponse.Result))

	// Check for subscription errors
	if genericResponse.Error != nil {
		p.logger.Warn("WebSocket subscription error",
			zap.String("channel", genericResponse.Channel),
			zap.String("event", genericResponse.Event),
			zap.Any("error", genericResponse.Error))
		return
	}

	// Process orderbook updates
	if genericResponse.Channel == "spot.order_book_update" && genericResponse.Event == "update" {
		var response OrderBookResponse
		if err := json.Unmarshal(message, &response); err != nil {
			p.logger.Error("Failed to unmarshal orderbook response", zap.Error(err))
			return
		}
		p.processOrderBookUpdate(response)
	} else if genericResponse.Event == "subscribe" {
		p.logger.Debug("Subscription confirmed",
			zap.String("channel", genericResponse.Channel),
			zap.Any("result", genericResponse.Result))
	}
}

func (p *Provider) processOrderBookUpdate(response OrderBookResponse) {
	symbol := response.Result.S

	orderBook := &domain.OrderBook{
		Exchange:  moduleName,
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids:      make([]domain.Order, 0),
		Asks:      make([]domain.Order, 0),
	}

	for _, bid := range response.Result.B {
		if len(bid) >= 2 {
			price := parseFloat(bid[0])
			volume := parseFloat(bid[1])
			if price > 0 && volume > 0 {
				orderBook.Bids = append(orderBook.Bids, domain.Order{
					Price:  price,
					Volume: volume,
				})
			}
		}
	}

	for _, ask := range response.Result.A {
		if len(ask) >= 2 {
			price := parseFloat(ask[0])
			volume := parseFloat(ask[1])
			if price > 0 && volume > 0 {
				orderBook.Asks = append(orderBook.Asks, domain.Order{
					Price:  price,
					Volume: volume,
				})
			}
		}
	}

	p.mu.Lock()
	p.orderBookCache[symbol] = orderBook
	p.mu.Unlock()

	p.logger.Debug("Updated orderbook",
		zap.String("symbol", symbol),
		zap.Int("bids", len(orderBook.Bids)),
		zap.Int("asks", len(orderBook.Asks)))
}

func parseFloat(s string) float64 {
	var f float64
	if err := json.Unmarshal([]byte(s), &f); err != nil {
		panic("failed to parse float")
	}
	return f
}
