package bybit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/igefined/zero-delta-screener/internal/config"
	"github.com/igefined/zero-delta-screener/internal/domain"
	"go.uber.org/zap"
)

type Provider struct {
	config          config.ByBitConfig
	supportedTokens []string
	logger          *zap.Logger
	conn            *websocket.Conn
	mu              sync.RWMutex
	writeMu         sync.Mutex
	orderBookCache  map[string]*domain.OrderBook
	subscriptions   map[string]bool
}

func NewProvider(cfg config.ByBitConfig, supportedTokens []string, logger *zap.Logger) *Provider {
	return &Provider{
		config:          cfg,
		supportedTokens: supportedTokens,
		logger:          logger.Named(moduleName),
		orderBookCache:  make(map[string]*domain.OrderBook),
		subscriptions:   make(map[string]bool),
	}
}

func (p *Provider) Name() string {
	return moduleName
}

func (p *Provider) Connect(ctx context.Context) error {
	p.logger.Info("Connecting to ByBit exchange")

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(p.config.WsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to ByBit websocket: %w", err)
	}

	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	go p.messageHandler(ctx)
	go p.pingHandler(ctx)

	p.logger.Info("Connected to ByBit exchange")
	return nil
}

func (p *Provider) Disconnect() error {
	p.logger.Info("Disconnecting from ByBit exchange")

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		return p.conn.Close()
	}

	return nil
}

func (p *Provider) FetchOrderBook(symbol string) (*domain.OrderBook, error) {
	p.logger.Info("Fetching order book", zap.String("symbol", symbol))

	before, after, _ := strings.Cut(symbol, "_")
	bbSymbol := strings.ToUpper(fmt.Sprintf("%s%s", before, after))

	if err := p.subscribeToOrderBook(bbSymbol); err != nil {
		return nil, fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			p.mu.RLock()
			orderBook, exists := p.orderBookCache[symbol]
			p.mu.RUnlock()

			if exists {
				return orderBook, nil
			}
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for orderbook data for symbol: %s", symbol)
		}
	}
}

func (p *Provider) FetchOrderBooks(symbols []string) (map[string]*domain.OrderBook, error) {
	p.logger.Info("Fetching multiple order books", zap.Strings("symbols", symbols))

	results := make(map[string]*domain.OrderBook)
	errors := make(map[string]error)

	for _, symbol := range symbols {
		if err := p.subscribeToOrderBook(symbol); err != nil {
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

func (p *Provider) genSign(timestamp int64) string {
	expires := strconv.FormatInt(timestamp, 10)
	signaturePayload := "GET/realtime" + expires

	h := hmac.New(sha256.New, []byte(p.config.APISecret))
	h.Write([]byte(signaturePayload))
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}

func (p *Provider) subscribeToOrderBook(symbol string) error {
	p.mu.RLock()
	conn := p.conn
	alreadySubscribed := p.subscriptions[symbol]
	p.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	if alreadySubscribed {
		return nil
	}

	var request map[string]interface{}

	if p.config.APIKey != "" && p.config.APISecret != "" {
		timestamp := time.Now().UnixMilli() + 5000
		signature := p.genSign(timestamp)

		authRequest := map[string]interface{}{
			"op": "auth",
			"args": []interface{}{
				p.config.APIKey,
				timestamp,
				signature,
			},
		}

		authMsg, err := json.Marshal(authRequest)
		if err != nil {
			return fmt.Errorf("failed to marshal auth request: %w", err)
		}

		p.writeMu.Lock()
		if err = conn.WriteMessage(websocket.TextMessage, authMsg); err != nil {
			p.writeMu.Unlock()
			return fmt.Errorf("failed to send auth request: %w", err)
		}
		p.writeMu.Unlock()

		time.Sleep(100 * time.Millisecond)
	}

	request = map[string]interface{}{
		"op": "subscribe",
		"args": []string{
			fmt.Sprintf("orderbook.50.%s", symbol),
		},
	}

	msg, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription request: %w", err)
	}

	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	p.logger.Info("Subscribing to orderbook", zap.String("symbol", symbol))
	if err = conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return fmt.Errorf("failed to send subscription request: %w", err)
	}

	p.mu.Lock()
	p.subscriptions[symbol] = true
	p.mu.Unlock()

	return nil
}

func (p *Provider) pingHandler(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.mu.RLock()
			conn := p.conn
			p.mu.RUnlock()

			if conn == nil {
				return
			}

			pingMsg := map[string]string{"op": "ping"}
			msg, err := json.Marshal(pingMsg)
			if err != nil {
				p.logger.Error("Failed to marshal ping message", zap.Error(err))
				continue
			}

			p.writeMu.Lock()
			if err = conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				p.logger.Error("Failed to send ping", zap.Error(err))
			}
			p.writeMu.Unlock()
		}
	}
}

type OrderBookMessage struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Ts    int64  `json:"ts"`
	Data  struct {
		S   string     `json:"s"`
		B   [][]string `json:"b"`
		A   [][]string `json:"a"`
		U   int64      `json:"u"`
		Seq int64      `json:"seq"`
	} `json:"data"`
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
	var genericResponse struct {
		Success bool        `json:"success"`
		RetMsg  string      `json:"ret_msg"`
		Op      string      `json:"op"`
		Topic   string      `json:"topic"`
		Type    string      `json:"type"`
		Data    interface{} `json:"data"`
	}

	if err := json.Unmarshal(message, &genericResponse); err != nil {
		p.logger.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	if genericResponse.Op == "pong" {
		p.logger.Debug("Received pong")
		return
	}

	if genericResponse.Op == "subscribe" {
		if genericResponse.Success {
			p.logger.Info("Subscription successful", zap.String("ret_msg", genericResponse.RetMsg))
		} else {
			p.logger.Error("Subscription failed", zap.String("ret_msg", genericResponse.RetMsg))
		}
		return
	}

	if genericResponse.Op == "auth" {
		if genericResponse.Success {
			p.logger.Info("Authentication successful")
		} else {
			p.logger.Error("Authentication failed", zap.String("ret_msg", genericResponse.RetMsg))
		}
		return
	}

	if genericResponse.Topic != "" && genericResponse.Type == "snapshot" {
		var orderBookMsg OrderBookMessage
		if err := json.Unmarshal(message, &orderBookMsg); err != nil {
			p.logger.Error("Failed to unmarshal orderbook message", zap.Error(err))
			return
		}
		p.processOrderBookUpdate(orderBookMsg)
	} else if genericResponse.Topic != "" && genericResponse.Type == "delta" {
		var orderBookMsg OrderBookMessage
		if err := json.Unmarshal(message, &orderBookMsg); err != nil {
			p.logger.Error("Failed to unmarshal orderbook delta message", zap.Error(err))
			return
		}
		p.processOrderBookUpdate(orderBookMsg)
	}
}

func (p *Provider) processOrderBookUpdate(msg OrderBookMessage) {
	symbol := msg.Data.S

	orderBook := &domain.OrderBook{
		Exchange:  moduleName,
		Symbol:    symbol,
		Timestamp: time.UnixMilli(msg.Ts),
		Bids:      make([]domain.Order, 0),
		Asks:      make([]domain.Order, 0),
	}

	for _, bid := range msg.Data.B {
		if len(bid) >= 2 {
			price, err := strconv.ParseFloat(bid[0], 64)
			if err != nil {
				p.logger.Error("Failed to parse bid price", zap.Error(err))
				continue
			}
			volume, err := strconv.ParseFloat(bid[1], 64)
			if err != nil {
				p.logger.Error("Failed to parse bid volume", zap.Error(err))
				continue
			}
			if price > 0 && volume > 0 {
				orderBook.Bids = append(orderBook.Bids, domain.Order{
					Price:  price,
					Volume: volume,
				})
			}
		}
	}

	for _, ask := range msg.Data.A {
		if len(ask) >= 2 {
			price, err := strconv.ParseFloat(ask[0], 64)
			if err != nil {
				p.logger.Error("Failed to parse ask price", zap.Error(err))
				continue
			}
			volume, err := strconv.ParseFloat(ask[1], 64)
			if err != nil {
				p.logger.Error("Failed to parse ask volume", zap.Error(err))
				continue
			}
			if price > 0 && volume > 0 {
				orderBook.Asks = append(orderBook.Asks, domain.Order{
					Price:  price,
					Volume: volume,
				})
			}
		}
	}

	p.mu.Lock()
	if msg.Type == "snapshot" {
		p.orderBookCache[symbol] = orderBook
	} else if msg.Type == "delta" {
		if existingBook, exists := p.orderBookCache[symbol]; exists {
			p.mergeOrderBookDelta(existingBook, orderBook)
		} else {
			p.orderBookCache[symbol] = orderBook
		}
	}
	p.mu.Unlock()

	p.logger.Debug("Updated orderbook",
		zap.String("symbol", symbol),
		zap.String("type", msg.Type),
		zap.Int("bids", len(orderBook.Bids)),
		zap.Int("asks", len(orderBook.Asks)))
}

func (p *Provider) mergeOrderBookDelta(existing, delta *domain.OrderBook) {
	existing.Timestamp = delta.Timestamp

	bidMap := make(map[float64]float64)
	for _, bid := range existing.Bids {
		bidMap[bid.Price] = bid.Volume
	}
	for _, bid := range delta.Bids {
		if bid.Volume == 0 {
			delete(bidMap, bid.Price)
		} else {
			bidMap[bid.Price] = bid.Volume
		}
	}

	askMap := make(map[float64]float64)
	for _, ask := range existing.Asks {
		askMap[ask.Price] = ask.Volume
	}
	for _, ask := range delta.Asks {
		if ask.Volume == 0 {
			delete(askMap, ask.Price)
		} else {
			askMap[ask.Price] = ask.Volume
		}
	}

	existing.Bids = make([]domain.Order, 0, len(bidMap))
	for price, volume := range bidMap {
		existing.Bids = append(existing.Bids, domain.Order{
			Price:  price,
			Volume: volume,
		})
	}

	existing.Asks = make([]domain.Order, 0, len(askMap))
	for price, volume := range askMap {
		existing.Asks = append(existing.Asks, domain.Order{
			Price:  price,
			Volume: volume,
		})
	}
}
