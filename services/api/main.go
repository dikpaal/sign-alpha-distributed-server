package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// ProcessedMessage from processing service
type ProcessedMessage struct {
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	MovingAverage float64 `json:"moving_average"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Time          int64   `json:"time"`
}

// Trade for history endpoint
type Trade struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// Server holds application state
type Server struct {
	mu       sync.RWMutex
	current  ProcessedMessage
	symbol   string
	coinName string

	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex

	db *pgxpool.Pool
	nc *nats.Conn
}

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

func getCoinName(symbol string) string {
	for _, c := range coins {
		if c.symbol == symbol {
			return c.name
		}
	}
	return symbol
}

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trading:trading123@localhost:5432/trading_pipeline?sslmode=disable"
	}

	log.Println("API service starting...")

	// Connect to NATS
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
	log.Println("Connected to NATS")

	// Connect to database
	var db *pgxpool.Pool
	for i := 0; i < 10; i++ {
		db, err = pgxpool.New(context.Background(), dbURL)
		if err == nil {
			break
		}
		log.Printf("DB connection failed, retrying in 2s... (%v)", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("Warning: Database not available: %v", err)
	} else {
		log.Println("Connected to TimescaleDB")
		initSchema(db)
	}

	server := &Server{
		symbol:   "btcusdt",
		coinName: "Bitcoin (BTC)",
		clients:  make(map[*websocket.Conn]bool),
		db:       db,
		nc:       nc,
	}

	// Subscribe to processed trades
	nc.Subscribe("trades.processed", func(msg *nats.Msg) {
		var processed ProcessedMessage
		if err := json.Unmarshal(msg.Data, &processed); err != nil {
			return
		}

		server.mu.Lock()
		server.current = processed
		server.mu.Unlock()

		// Write to database
		if db != nil {
			go func() {
				_, err := db.Exec(context.Background(),
					"INSERT INTO trades (time, symbol, price) VALUES ($1, $2, $3)",
					time.Now(), processed.Symbol, processed.Price)
				if err != nil {
					log.Printf("DB write error: %v", err)
				}
			}()
		}

		// Broadcast to WebSocket clients
		server.broadcast(processed.Price)
	})

	// HTTP routes
	http.HandleFunc("/api/price", server.handlePrice)
	http.HandleFunc("/api/stats", server.handleStats)
	http.HandleFunc("/api/history", server.handleHistory)
	http.HandleFunc("/api/symbol", server.handleSymbol)
	http.HandleFunc("/api/coins", server.handleCoins)
	http.HandleFunc("/ws", server.handleWebSocket)

	log.Println("Server running on http://localhost:8080")
	log.Println("Endpoints:")
	log.Println("  GET  /api/price   - Current price")
	log.Println("  GET  /api/stats   - Moving average, high, low")
	log.Println("  GET  /api/history - Historical trades")
	log.Println("  GET  /api/symbol  - Current symbol")
	log.Println("  POST /api/symbol  - Change symbol")
	log.Println("  GET  /api/coins   - Available coins")
	log.Println("  WS   /ws          - Real-time prices")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func initSchema(db *pgxpool.Pool) {
	ctx := context.Background()
	db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS trades (
			time TIMESTAMPTZ NOT NULL,
			symbol TEXT NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)
	`)
	db.Exec(ctx, `SELECT create_hypertable('trades', 'time', if_not_exists => TRUE)`)
	db.Exec(ctx, `CREATE INDEX IF NOT EXISTS trades_symbol_time_idx ON trades (symbol, time DESC)`)
}

func (s *Server) handlePrice(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	price := s.current.Price
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]float64{"price": price})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	stats := map[string]float64{
		"moving_average": s.current.MovingAverage,
		"high":           s.current.High,
		"low":            s.current.Low,
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	s.mu.RLock()
	symbol := s.symbol
	s.mu.RUnlock()

	rows, err := s.db.Query(context.Background(),
		`SELECT symbol, price, time FROM trades WHERE symbol = $1 ORDER BY time DESC LIMIT 100`,
		symbol)
	if err != nil {
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(&t.Symbol, &t.Price, &t.Timestamp); err != nil {
			continue
		}
		trades = append(trades, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trades)
}

func (s *Server) handleSymbol(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Symbol string `json:"symbol"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		newName := getCoinName(req.Symbol)
		if newName == req.Symbol {
			http.Error(w, "Unknown symbol", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.symbol = req.Symbol
		s.coinName = newName
		s.current = ProcessedMessage{}
		s.mu.Unlock()

		// Notify other services via NATS
		msg, _ := json.Marshal(map[string]string{"symbol": req.Symbol})
		s.nc.Publish("control.symbol", msg)

		log.Printf("Changed to %s", newName)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"symbol": req.Symbol, "name": newName})
		return
	}

	s.mu.RLock()
	symbol := s.symbol
	name := s.coinName
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"symbol": symbol, "name": name})
}

func (s *Server) handleCoins(w http.ResponseWriter, r *http.Request) {
	list := []map[string]string{
		{"symbol": "btcusdt", "name": "Bitcoin (BTC)"},
		{"symbol": "ethusdt", "name": "Ethereum (ETH)"},
		{"symbol": "solusdt", "name": "Solana (SOL)"},
		{"symbol": "bnbusdt", "name": "Binance Coin (BNB)"},
		{"symbol": "xrpusdt", "name": "Ripple (XRP)"},
		{"symbol": "dogeusdt", "name": "Dogecoin (DOGE)"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	log.Printf("Client connected. Total: %d", len(s.clients))

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			s.clientsMu.Lock()
			delete(s.clients, conn)
			s.clientsMu.Unlock()
			log.Printf("Client disconnected. Total: %d", len(s.clients))
			return
		}
	}
}

func (s *Server) broadcast(price float64) {
	msg, _ := json.Marshal(map[string]float64{"price": price})

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
			client.Close()
			go func(c *websocket.Conn) {
				s.clientsMu.Lock()
				delete(s.clients, c)
				s.clientsMu.Unlock()
			}(client)
		}
	}
}
