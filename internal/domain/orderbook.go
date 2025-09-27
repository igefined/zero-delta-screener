package domain

import (
	"fmt"
	"time"
)

type OrderBook struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Bids      []Order   `json:"bids"`
	Asks      []Order   `json:"asks"`
}

type Order struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

func (o *Order) String() string {
	return fmt.Sprintf("Price: %.8f, Volume: %.8f", o.Price, o.Volume)
}

type Asset struct {
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
}
