package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("Starting Trading Pipeline...")

	// Create server
	server := NewServer()

	// Price channel for Binance updates
	priceChan := make(chan PriceUpdate, 100)

	// Start Binance WebSocket connection in background
	go ConnectBinance(priceChan)

	// Process incoming prices
	go func() {
		for update := range priceChan {
			server.UpdatePrice(update.Price)
		}
	}()

	// Setup HTTP routes
	http.HandleFunc("/api/price", server.HandlePrice)
	http.HandleFunc("/api/stats", server.HandleStats)
	http.HandleFunc("/ws", server.HandleWebSocket)

	// Start HTTP server
	log.Println("Server running on http://localhost:8080")
	log.Println("Endpoints:")
	log.Println("  GET  /api/price  - Current BTC price")
	log.Println("  GET  /api/stats  - Moving average, high, low")
	log.Println("  WS   /ws         - Real-time price stream")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
