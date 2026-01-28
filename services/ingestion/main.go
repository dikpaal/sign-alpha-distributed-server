package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

// TradeMessage is published to NATS
type TradeMessage struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Time   int64   `json:"time"`
}

// BinanceTrade represents a trade event from Binance
type BinanceTrade struct {
	Price string `json:"p"`
	Time  int64  `json:"T"`
}

func main() {
	symbol := os.Getenv("SYMBOL")
	if symbol == "" {
		symbol = "btcusdt"
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	log.Printf("Ingestion service starting for %s", symbol)

	// Connect to NATS with retry
	var nc *nats.Conn
	var err error
	for i := 0; i < 10; i++ {
		nc, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		log.Printf("NATS connection failed, retrying in 2s... (%v)", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Println("Connected to NATS")

	// Track current symbol for dynamic switching
	var mu sync.RWMutex
	currentSymbol := symbol

	// Subscribe to symbol change requests
	nc.Subscribe("control.symbol", func(msg *nats.Msg) {
		var req struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return
		}
		mu.Lock()
		currentSymbol = req.Symbol
		mu.Unlock()
		log.Printf("Symbol changed to %s", req.Symbol)
	})

	// Start Binance connection loop
	for {
		mu.RLock()
		sym := currentSymbol
		mu.RUnlock()

		connectToBinance(nc, sym, &mu, &currentSymbol)
		time.Sleep(2 * time.Second)
	}
}

func connectToBinance(nc *nats.Conn, symbol string, mu *sync.RWMutex, currentSymbol *string) {
	url := "wss://stream.binance.com:9443/ws/" + symbol + "@trade"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Binance connection error: %v", err)
		return
	}
	defer conn.Close()
	log.Printf("Connected to Binance for %s", symbol)

	for {
		// Check if symbol changed
		mu.RLock()
		newSymbol := *currentSymbol
		mu.RUnlock()
		if newSymbol != symbol {
			log.Printf("Symbol changed, reconnecting...")
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		var trade BinanceTrade
		if err := json.Unmarshal(message, &trade); err != nil {
			continue
		}

		var price float64
		if _, err := json.Number(trade.Price).Float64(); err == nil {
			json.Unmarshal([]byte(trade.Price), &price)
		}

		if price > 0 {
			msg := TradeMessage{
				Symbol: symbol,
				Price:  price,
				Time:   trade.Time,
			}
			data, _ := json.Marshal(msg)
			nc.Publish("trades.raw", data)
		}
	}
}
