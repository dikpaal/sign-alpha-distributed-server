package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB holds the database connection pool
type DB struct {
	pool *pgxpool.Pool
}

// Trade represents a trade record
type Trade struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// NewDB creates a new database connection
func NewDB() (*DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trading:trading123@localhost:5432/trading_pipeline?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}

	db := &DB{pool: pool}
	if err := db.initSchema(); err != nil {
		return nil, err
	}

	return db, nil
}

// initSchema creates the trades hypertable if it doesn't exist
func (db *DB) initSchema() error {
	ctx := context.Background()

	// Create trades table
	_, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS trades (
			time TIMESTAMPTZ NOT NULL,
			symbol TEXT NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Convert to hypertable (TimescaleDB)
	_, err = db.pool.Exec(ctx, `
		SELECT create_hypertable('trades', 'time', if_not_exists => TRUE)
	`)
	if err != nil {
		log.Printf("Hypertable creation note: %v", err)
	}

	// Create index for faster symbol queries
	_, err = db.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS trades_symbol_time_idx ON trades (symbol, time DESC)
	`)

	return err
}

// InsertTrade inserts a new trade record
func (db *DB) InsertTrade(symbol string, price float64) error {
	_, err := db.pool.Exec(context.Background(),
		"INSERT INTO trades (time, symbol, price) VALUES ($1, $2, $3)",
		time.Now(), symbol, price)
	return err
}

// GetHistory returns recent trades for a symbol
func (db *DB) GetHistory(symbol string, limit int) ([]Trade, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.pool.Query(context.Background(),
		`SELECT symbol, price, time FROM trades
		 WHERE symbol = $1
		 ORDER BY time DESC
		 LIMIT $2`,
		symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(&t.Symbol, &t.Price, &t.Timestamp); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	return trades, nil
}

// GetHistoricalStats returns aggregated stats for a time period
func (db *DB) GetHistoricalStats(symbol string, duration time.Duration) (map[string]float64, error) {
	since := time.Now().Add(-duration)

	var avg, high, low float64
	err := db.pool.QueryRow(context.Background(),
		`SELECT COALESCE(AVG(price), 0), COALESCE(MAX(price), 0), COALESCE(MIN(price), 0)
		 FROM trades
		 WHERE symbol = $1 AND time > $2`,
		symbol, since).Scan(&avg, &high, &low)

	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"average": avg,
		"high":    high,
		"low":     low,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	db.pool.Close()
}
