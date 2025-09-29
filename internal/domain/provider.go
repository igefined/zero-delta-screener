package domain

type Provider interface {
	Name() string
	FetchOrderBook(symbol string) (*OrderBook, error)
	FetchOrderBooks(symbols []string) (map[string]*OrderBook, error)
	GetSupportedSymbols() []string
	Connect(closeCh <-chan struct{}) error
	Disconnect() error
}
