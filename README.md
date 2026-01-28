# Real-Time Crypto Data Pipeline

A real-time data acquisition pipeline that streams cryptocurrency market data from Binance, processes it through a C++ signal processing module, and serves it via a Go HTTP/WebSocket server with a terminal-based dashboard.

## Features

- **Real-time price streaming** from Binance WebSocket API
- **C++ signal processing** with moving averages and high/low tracking
- **Thread-safe REST API** endpoints with mutex-protected state
- **WebSocket server** for real-time price broadcasts
- **Interactive TUI dashboard** with live price updates and sparkline charts
- **Dynamic coin switching** without server restart

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Binance      │────▶│   Go Server     │────▶│   TUI Client    │
│   WebSocket     │     │                 │     │                 │
└─────────────────┘     │  ┌───────────┐  │     │  ┌───────────┐  │
                        │  │  Binance  │  │     │  │ Dashboard │  │
                        │  │  Client   │  │     │  │   View    │  │
                        │  └─────┬─────┘  │     │  └───────────┘  │
                        │        │        │     │                 │
                        │  ┌─────▼─────┐  │     │  ┌───────────┐  │
                        │  │    C++    │  │     │  │   Coin    │  │
                        │  │ Processor │  │     │  │ Selector  │  │
                        │  └─────┬─────┘  │     │  └───────────┘  │
                        │        │        │     │                 │
                        │  ┌─────▼─────┐  │     └─────────────────┘
                        │  │  HTTP/WS  │  │
                        │  │  Server   │──┼──▶ REST API + WebSocket
                        │  └───────────┘  │
                        └─────────────────┘
```

## Project Structure

```
TRADING-PIPELINE/
├── README.md
├── Makefile              # Build and run commands
├── .gitignore
├── server/               # Go server with C++ integration
│   ├── main.go           # Server entry point, HTTP routes
│   ├── handlers.go       # HTTP/WebSocket handlers (thread-safe)
│   ├── binance.go        # Binance WebSocket client
│   ├── process.cpp       # C++ signal processing (SMA, high/low)
│   ├── process.h         # C++ header for CGO
│   ├── go.mod
│   └── go.sum
├── tui/                  # Terminal UI client
│   ├── main.go           # TUI dashboard with coin selection
│   ├── go.mod
│   └── go.sum
└── scripts/
    └── test.sh           # CLI test suite
```

## Tech Stack

### Languages
| Language | Version | Usage |
|----------|---------|-------|
| Go | 1.23+ | Server, WebSocket handling, HTTP API |
| C++ | C++11 | Signal processing (moving average, high/low) |
| Bash | - | Test scripts |

### Go Packages

**Server:**
| Package | Version | Purpose |
|---------|---------|---------|
| `gorilla/websocket` | v1.5.3 | WebSocket client (Binance) and server |
| Standard library (`net/http`, `sync`, `encoding/json`) | - | HTTP server, thread safety, JSON encoding |

**TUI Client:**
| Package | Version | Purpose |
|---------|---------|---------|
| `bubbletea` | v1.3.10 | Terminal UI framework (Elm architecture) |
| `lipgloss` | v1.1.0 | Terminal styling and colors |

### C++ Libraries
| Library | Purpose |
|---------|---------|
| `<mutex>` | Thread-safe price buffer access |
| `<vector>` | Circular buffer for price history |
| `<limits>` | Numeric limits for high/low initialization |

### External APIs
| API | Protocol | Purpose |
|-----|----------|---------|
| Binance WebSocket | `wss://stream.binance.com:9443` | Real-time trade data (free, no API key) |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/price` | Current cryptocurrency price |
| GET | `/api/stats` | Moving average, session high/low from C++ |
| GET | `/api/symbol` | Current trading pair info |
| POST | `/api/symbol` | Change trading pair |
| GET | `/api/coins` | List available cryptocurrencies |
| WS | `/ws` | Real-time price stream |

## Prerequisites

- **Go** 1.23 or higher
- **g++** (C++ compiler with C++11 support)
- **make**
- **Docker** (optional, for containerized deployment)

### macOS
```bash
# Install Go
brew install go

# g++ comes with Xcode Command Line Tools
xcode-select --install
```

### Ubuntu/Debian
```bash
sudo apt update
sudo apt install golang g++ make
```

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/TRADING-PIPELINE.git
cd TRADING-PIPELINE

# Build everything
make build
```

## Usage

### Quick Start

**Terminal 1 - Start the server:**
```bash
make run
```

**Terminal 2 - Launch the TUI:**
```bash
make tui
```

### Docker

```bash
# Build and run with Docker
make docker-run

# Stop container
make docker-stop
```

Or manually:
```bash
docker build -t trading-pipeline .
docker run -p 8080:8080 trading-pipeline
```

### TUI Controls

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate coin list |
| `Enter` | Select coin |
| `c` | Change coin (from dashboard) |
| `q` | Quit |

### API Testing

```bash
# Get current price
curl http://localhost:8080/api/price

# Get stats from C++ processor
curl http://localhost:8080/api/stats

# Get current symbol
curl http://localhost:8080/api/symbol

# Change to Ethereum
curl -X POST http://localhost:8080/api/symbol \
  -H "Content-Type: application/json" \
  -d '{"symbol":"ethusdt"}'

# List available coins
curl http://localhost:8080/api/coins
```

### Run Test Suite

```bash
./scripts/test.sh
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

## How It Works

1. **Server** connects to Binance WebSocket for real-time trade data
2. **Prices** are sent to the C++ processor via CGO
3. **C++ module** maintains a circular buffer and calculates:
   - Simple Moving Average (SMA) over last 20 prices
   - Session high/low prices
4. **HTTP API** serves current price and stats (thread-safe with `sync.RWMutex`)
5. **WebSocket** broadcasts price updates to connected clients
6. **TUI client** polls the API and renders a live dashboard

## Make Commands

| Command | Description |
|---------|-------------|
| `make build` | Build server and TUI |
| `make run` | Build and run server |
| `make tui` | Build and run TUI client |
| `make test` | Run test suite |
| `make clean` | Remove build artifacts |
| `make docker` | Build Docker image |
| `make docker-run` | Run server in Docker container |
| `make docker-stop` | Stop Docker container |

## License

MIT
