package main

import (
	"encoding/json"
	"log"
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

// ConnectBinance connects to Binance WebSocket and streams BTC prices
func ConnectBinance(priceChan chan<- PriceUpdate) {
	url := "wss://stream.binance.com:9443/ws/btcusdt@trade"

	for {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Printf("Binance connection error: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Connected to Binance WebSocket")
		readBinanceMessages(conn, priceChan)
		conn.Close()

		log.Println("Binance connection lost, reconnecting...")
		time.Sleep(2 * time.Second)
	}
}

// readBinanceMessages reads messages from Binance and sends price updates
func readBinanceMessages(conn *websocket.Conn, priceChan chan<- PriceUpdate) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
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
			priceChan <- PriceUpdate{
				Price: price,
				Time:  time.UnixMilli(trade.Time),
			}
		}
	}
}
