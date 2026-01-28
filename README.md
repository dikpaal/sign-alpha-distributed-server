# Real-Time Crypto Data Pipeline

A distributed, event-driven data pipeline that streams cryptocurrency market data from Binance, processes it through a C++ signal processing module, persists to TimescaleDB, and serves it via HTTP/WebSocket with a terminal-based dashboard.

## Features

- **Microservices architecture** with NATS message queue
- **Real-time price streaming** from Binance WebSocket API
- **C++ signal processing** with moving averages and high/low tracking
- **TimescaleDB persistence** for historical trade data
- **Thread-safe REST API** with WebSocket broadcasts
- **Interactive TUI dashboard** with live price updates and sparkline charts
- **Dynamic coin switching** propagated across all services

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Binance   │────▶│  Ingestion  │────▶│    NATS     │────▶│  Processing │
│  WebSocket  │     │   Service   │     │   Message   │     │   Service   │
└─────────────┘     └─────────────┘     │    Queue    │     │    (C++)    │
                                        └──────┬──────┘     └──────┬──────┘
                                               │                   │
                                               │ trades.processed  │
                                               ▼                   │
┌─────────────┐     ┌─────────────┐     ┌─────────────┐◀───────────┘
│ TUI Client  │◀───▶│ API Service │◀───▶│ TimescaleDB │
│             │     │  (HTTP/WS)  │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
```

**Data Flow:**
1. **Ingestion** pulls trades from Binance → publishes to `trades.raw`
2. **Processing** subscribes, runs C++ analysis → publishes to `trades.processed`
3. **API** subscribes, stores in DB, serves HTTP/WS
4. **Symbol changes** propagate via NATS `control.symbol` topic

## Project Structure

```
TRADING-PIPELINE/
├── README.md
├── Makefile                 # Build and run commands
├── docker-compose.yml       # Service orchestration
├── services/
│   ├── ingestion/           # Binance WebSocket → NATS
│   │   ├── main.go
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── processing/          # NATS → C++ processing → NATS
│   │   ├── main.go
│   │   ├── process.cpp
│   │   ├── process.h
│   │   ├── Dockerfile
│   │   └── go.mod
│   └── api/                 # NATS + TimescaleDB → HTTP/WS
│       ├── main.go
│       ├── Dockerfile
│       └── go.mod
├── tui/                     # Terminal UI client
│   ├── main.go
│   └── go.mod
└── scripts/
    └── test.sh
```

## Tech Stack

### Languages
| Language | Version | Usage |
|----------|---------|-------|
| Go | 1.23+ | All services, HTTP API, WebSocket |
| C++ | C++11 | Signal processing (SMA, high/low) |

### Infrastructure
| Component | Technology | Purpose |
|-----------|------------|---------|
| Message Queue | NATS | Event-driven communication between services |
| Database | TimescaleDB | Time-series storage for trade history |
| Containers | Docker Compose | Service orchestration |

### Go Packages
| Package | Purpose |
|---------|---------|
| `gorilla/websocket` | WebSocket client/server |
| `nats-io/nats.go` | NATS messaging |
| `jackc/pgx/v5` | PostgreSQL/TimescaleDB driver |
| `bubbletea` | Terminal UI framework |
| `lipgloss` | Terminal styling |

### External APIs
| API | Protocol | Purpose |
|-----|----------|---------|
| Binance WebSocket | `wss://stream.binance.com:9443` | Real-time trade data |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/price` | Current cryptocurrency price |
| GET | `/api/stats` | Moving average, session high/low |
| GET | `/api/history` | Historical trades from database |
| GET | `/api/symbol` | Current trading pair info |
| POST | `/api/symbol` | Change trading pair |
| GET | `/api/coins` | List available cryptocurrencies |
| WS | `/ws` | Real-time price stream |

## Prerequisites

- **Docker** and **Docker Compose**
- **Go** 1.23+ (only for TUI client)

## Quick Start

```bash
# Clone the repository
git clone https://github.com/yourusername/TRADING-PIPELINE.git
cd TRADING-PIPELINE

# Start the distributed system
make run

# In another terminal, launch TUI
make tui

# Stop all services
make stop
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| `timescaledb` | 5433 | PostgreSQL with time-series extension |
| `nats` | 4222, 8222 | Message queue (8222 for monitoring) |
| `ingestion` | - | Binance WebSocket client |
| `processing` | - | C++ signal processing |
| `api` | 8080 | HTTP/WebSocket server |

## TUI Controls

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate / scroll |
| `Enter` | Select coin |
| `c` | Change coin (from dashboard) |
| `h` | View trade history from TimescaleDB |
| `r` | Refresh history (in history view) |
| `esc` | Back to dashboard |
| `q` | Quit |

## API Testing

```bash
# Get current price
curl http://localhost:8080/api/price

# Get stats
curl http://localhost:8080/api/stats

# Get historical trades
curl http://localhost:8080/api/history

# Change to Ethereum
curl -X POST http://localhost:8080/api/symbol \
  -H "Content-Type: application/json" \
  -d '{"symbol":"ethusdt"}'

# List available coins
curl http://localhost:8080/api/coins
```

## Supported Cryptocurrencies

| Symbol | Name |
|--------|------|
| `btcusdt` | Bitcoin (BTC) |
| `ethusdt` | Ethereum (ETH) |
| `solusdt` | Solana (SOL) |
| `bnbusdt` | Binance Coin (BNB) |
| `xrpusdt` | Ripple (XRP) |
| `dogeusdt` | Dogecoin (DOGE) |

## Make Commands

| Command | Description |
|---------|-------------|
| `make run` | Start all services |
| `make stop` | Stop all services |
| `make build` | Build Docker images |
| `make tui` | Build and run TUI client |
| `make logs` | View all service logs |
| `make logs-ingestion` | View ingestion logs |
| `make logs-processing` | View processing logs |
| `make logs-api` | View API logs |
| `make clean` | Remove images and artifacts |

## License

MIT
