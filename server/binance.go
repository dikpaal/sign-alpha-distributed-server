package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// BinanceTrade represents a trade event from Binance
type BinanceTrade struct {
	Price string `json:"p"`
	Time  int64  `json:"T"`
}

// PriceUpdate is sent through the channel when new price arrives
type PriceUpdate struct {
	Price float64
	Time  time.Time
}

// BinanceClient manages the Binance WebSocket connection
type BinanceClient struct {
	mu        sync.RWMutex
	symbol    string
	conn      *websocket.Conn
	priceChan chan<- PriceUpdate
	restart   chan struct{}
}

// NewBinanceClient creates a new Binance client
func NewBinanceClient(symbol string, priceChan chan<- PriceUpdate) *BinanceClient {
	return &BinanceClient{
		symbol:    symbol,
		priceChan: priceChan,
		restart:   make(chan struct{}, 1),
	}
}

// GetSymbol returns the current symbol
func (b *BinanceClient) GetSymbol() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.symbol
}

// ChangeSymbol changes the trading pair and reconnects
func (b *BinanceClient) ChangeSymbol(newSymbol string) {
	b.mu.Lock()
	b.symbol = newSymbol
	if b.conn != nil {
		b.conn.Close()
	}
	b.mu.Unlock()

	// Signal restart
	select {
	case b.restart <- struct{}{}:
	default:
	}

	log.Printf("Switching to %s", newSymbol)
}

// Run starts the Binance WebSocket connection loop
func (b *BinanceClient) Run() {
	for {
		b.mu.RLock()
		symbol := b.symbol
		b.mu.RUnlock()

		url := "wss://stream.binance.com:9443/ws/" + symbol + "@trade"

		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Printf("Binance connection error: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		b.mu.Lock()
		b.conn = conn
		b.mu.Unlock()

		log.Printf("Connected to Binance WebSocket for %s", symbol)
		b.readMessages(conn)
		conn.Close()

		// Check if we're restarting for symbol change
		select {
		case <-b.restart:
			log.Println("Reconnecting with new symbol...")
		default:
			log.Println("Connection lost, reconnecting...")
			time.Sleep(2 * time.Second)
		}
	}
}

// readMessages reads messages from Binance WebSocket
func (b *BinanceClient) readMessages(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var trade BinanceTrade
		if err := json.Unmarshal(message, &trade); err != nil {
			continue
		}

		// Parse price string to float
		var price float64
		if _, err := json.Number(trade.Price).Float64(); err == nil {
			json.Unmarshal([]byte(trade.Price), &price)
		}

		if price > 0 {
			b.priceChan <- PriceUpdate{
				Price: price,
				Time:  time.UnixMilli(trade.Time),
			}
		}
	}
}
