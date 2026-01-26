package main

/*
#cgo LDFLAGS: -L. -lprocess -lpthread -lstdc++
#include "process.h"
*/
import "C"

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Server holds the state for our HTTP/WebSocket server
type Server struct {
	mu           sync.RWMutex
	currentPrice float64
	clients      map[*websocket.Conn]bool
	clientsMu    sync.RWMutex
}

// NewServer creates a new server instance
func NewServer() *Server {
	return &Server{
		clients: make(map[*websocket.Conn]bool),
	}
}

// UpdatePrice updates the current price and notifies all WebSocket clients
func (s *Server) UpdatePrice(price float64) {
	// Update current price (thread-safe)
	s.mu.Lock()
	s.currentPrice = price
	s.mu.Unlock()

	// Send to C++ processor
	C.add_price(C.double(price))

	// Broadcast to all WebSocket clients
	s.broadcast(price)
}

// broadcast sends price to all connected WebSocket clients
func (s *Server) broadcast(price float64) {
	msg, _ := json.Marshal(map[string]float64{"price": price})

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for client := range s.clients {
		err := client.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			client.Close()
			go s.removeClient(client)
		}
	}
}

// removeClient removes a client from the broadcast list
func (s *Server) removeClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	delete(s.clients, conn)
	s.clientsMu.Unlock()
}

// GetPrice returns the current price (thread-safe)
func (s *Server) GetPrice() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentPrice
}

// HandlePrice returns the current BTC price
func (s *Server) HandlePrice(w http.ResponseWriter, r *http.Request) {
	price := s.GetPrice()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]float64{"price": price})
}

// HandleStats returns stats from the C++ processor
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]float64{
		"moving_average": float64(C.get_moving_average()),
		"high":           float64(C.get_high()),
		"low":            float64(C.get_low()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Add client to broadcast list
	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	log.Printf("Client connected. Total clients: %d", len(s.clients))

	// Keep connection open, remove on disconnect
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			s.removeClient(conn)
			log.Printf("Client disconnected. Total clients: %d", len(s.clients))
			return
		}
	}
}

// ResetPrice resets the current price (used when changing coins)
func (s *Server) ResetPrice() {
	s.mu.Lock()
	s.currentPrice = 0
	s.mu.Unlock()
}

// C++ wrapper functions for TUI access

func GetMovingAverage() float64 {
	return float64(C.get_moving_average())
}

func GetHigh() float64 {
	return float64(C.get_high())
}

func GetLow() float64 {
	return float64(C.get_low())
}

func ResetProcessor() {
	C.reset_processor()
}
