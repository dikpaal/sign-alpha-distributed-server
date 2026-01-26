package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync"
)

// AppState holds the application state
type AppState struct {
	mu       sync.RWMutex
	symbol   string
	coinName string
}

func (a *AppState) GetSymbol() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.symbol
}

func (a *AppState) GetCoinName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.coinName
}

func (a *AppState) SetSymbol(symbol, name string) {
	a.mu.Lock()
	a.symbol = symbol
	a.coinName = name
	a.mu.Unlock()
}

// Available coins
var coins = []struct {
	symbol string
	name   string
}{
	{"btcusdt", "Bitcoin (BTC)"},
	{"ethusdt", "Ethereum (ETH)"},
	{"solusdt", "Solana (SOL)"},
	{"bnbusdt", "Binance Coin (BNB)"},
	{"xrpusdt", "Ripple (XRP)"},
	{"dogeusdt", "Dogecoin (DOGE)"},
}

// GetCoinName returns the display name for a symbol
func GetCoinName(symbol string) string {
	for _, coin := range coins {
		if coin.symbol == symbol {
			return coin.name
		}
	}
	return symbol
}

func main() {
	// Parse command line flags
	symbol := flag.String("symbol", "btcusdt", "Trading pair symbol (e.g., btcusdt)")
	flag.Parse()

	selectedSymbol := *symbol
	coinName := GetCoinName(selectedSymbol)

	log.Printf("Starting Trading Pipeline Server for %s...\n", coinName)

	// App state
	appState := &AppState{
		symbol:   selectedSymbol,
		coinName: coinName,
	}

	// Create server
	server := NewServer()

	// Price channel for Binance updates
	priceChan := make(chan PriceUpdate, 100)

	// Create and start Binance client
	binanceClient := NewBinanceClient(selectedSymbol, priceChan)
	go binanceClient.Run()

	// Process incoming prices
	go func() {
		for update := range priceChan {
			server.UpdatePrice(update.Price)
		}
	}()

	// Setup HTTP routes
	http.HandleFunc("/api/price", server.HandlePrice)
	http.HandleFunc("/api/stats", server.HandleStats)

	// GET and POST /api/symbol
	http.HandleFunc("/api/symbol", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Change symbol
			var req struct {
				Symbol string `json:"symbol"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			newName := GetCoinName(req.Symbol)
			if newName == req.Symbol {
				http.Error(w, "Unknown symbol", http.StatusBadRequest)
				return
			}

			// Update state
			appState.SetSymbol(req.Symbol, newName)

			// Reset processor and price
			ResetProcessor()
			server.ResetPrice()

			// Change Binance connection
			binanceClient.ChangeSymbol(req.Symbol)

			log.Printf("Changed to %s", newName)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"symbol": req.Symbol,
				"name":   newName,
			})
			return
		}

		// GET - return current symbol
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"symbol": appState.GetSymbol(),
			"name":   appState.GetCoinName(),
		})
	})

	// Available coins endpoint
	http.HandleFunc("/api/coins", func(w http.ResponseWriter, r *http.Request) {
		coins := []map[string]string{
			{"symbol": "btcusdt", "name": "Bitcoin (BTC)"},
			{"symbol": "ethusdt", "name": "Ethereum (ETH)"},
			{"symbol": "solusdt", "name": "Solana (SOL)"},
			{"symbol": "bnbusdt", "name": "Binance Coin (BNB)"},
			{"symbol": "xrpusdt", "name": "Ripple (XRP)"},
			{"symbol": "dogeusdt", "name": "Dogecoin (DOGE)"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(coins)
	})

	http.HandleFunc("/ws", server.HandleWebSocket)

	// Start HTTP server
	log.Printf("Tracking: %s", coinName)
	log.Println("Server running on http://localhost:8080")
	log.Println("Endpoints:")
	log.Println("  GET  /api/price   - Current price")
	log.Println("  GET  /api/stats   - Moving average, high, low")
	log.Println("  GET  /api/symbol  - Current symbol info")
	log.Println("  POST /api/symbol  - Change symbol")
	log.Println("  GET  /api/coins   - Available coins")
	log.Println("  WS   /ws          - Real-time price stream")
	log.Println("")
	log.Println("Run 'make tui' in another terminal to view dashboard")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
