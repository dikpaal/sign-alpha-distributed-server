package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Parse command line flags
	symbol := flag.String("symbol", "", "Trading pair symbol (e.g., btcusdt)")
	flag.Parse()

	// If no symbol provided, run TUI to select
	selectedSymbol := *symbol
	if selectedSymbol == "" {
		var err error
		selectedSymbol, err = RunTUI()
		if err != nil {
			fmt.Println("Cancelled.")
			os.Exit(0)
		}
	}

	coinName := GetCoinName(selectedSymbol)
	fmt.Printf("\nStarting Trading Pipeline Server for %s...\n\n", coinName)

	// Create server
	server := NewServer()

	// Price channel for Binance updates
	priceChan := make(chan PriceUpdate, 100)

	// Start Binance WebSocket connection in background
	go ConnectBinance(selectedSymbol, priceChan)

	// Process incoming prices
	go func() {
		for update := range priceChan {
			server.UpdatePrice(update.Price)
		}
	}()

	// Setup HTTP routes
	http.HandleFunc("/api/price", server.HandlePrice)
	http.HandleFunc("/api/stats", server.HandleStats)
	http.HandleFunc("/api/symbol", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"symbol":"%s","name":"%s"}`, selectedSymbol, coinName)
	})
	http.HandleFunc("/ws", server.HandleWebSocket)

	// Start HTTP server
	log.Printf("Tracking: %s", coinName)
	log.Println("Server running on http://localhost:8080")
	log.Println("Endpoints:")
	log.Printf("  GET  /api/price   - Current %s price", selectedSymbol[:len(selectedSymbol)-4])
	log.Println("  GET  /api/stats   - Moving average, high, low")
	log.Println("  GET  /api/symbol  - Current symbol info")
	log.Println("  WS   /ws          - Real-time price stream")
	log.Println("")
	log.Println("Run 'make tui' in another terminal to view dashboard")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
