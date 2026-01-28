.PHONY: all build clean run tui test docker docker-run docker-stop services

# Default target
all: build

# Build monolithic server and TUI (for local dev without Docker)
build: build-server build-tui

# Build server with C++ library
build-server:
	@echo "Building server..."
	cd server && g++ -shared -fPIC -o libprocess.so process.cpp -lpthread
	cd server && CGO_ENABLED=1 go build -o trading-pipeline .

# Build TUI client
build-tui:
	@echo "Building TUI client..."
	cd tui && go build -o tui-client .

# Run the monolithic server (for local dev)
run: build-server
	@echo "Starting server..."
	cd server && LD_LIBRARY_PATH=. ./trading-pipeline

# Run the TUI client
tui: build-tui
	@echo "Starting TUI..."
	cd tui && ./tui-client

# Run test script
test: build
	@echo "Running tests..."
	cd server && LD_LIBRARY_PATH=. ../scripts/test.sh

# Clean build artifacts
clean:
	rm -f server/libprocess.so server/trading-pipeline
	rm -f tui/tui-client
	rm -f services/processing/libprocess.so

# Build all microservices Docker images
services:
	@echo "Building microservices..."
	docker-compose build

# Run full distributed system with Docker
docker-run:
	@echo "Starting distributed pipeline..."
	docker-compose up -d
	@echo ""
	@echo "Services running:"
	@echo "  - TimescaleDB: localhost:5432"
	@echo "  - NATS:        localhost:4222 (monitoring: localhost:8222)"
	@echo "  - API:         localhost:8080"
	@echo ""
	@echo "Run 'make tui' to view dashboard"
	@echo "Run 'make logs' to view service logs"

# Stop all containers
docker-stop:
	@echo "Stopping containers..."
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# View specific service logs
logs-ingestion:
	docker-compose logs -f ingestion

logs-processing:
	docker-compose logs -f processing

logs-api:
	docker-compose logs -f api
