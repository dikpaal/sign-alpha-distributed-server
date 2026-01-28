.PHONY: all build run tui stop logs clean

# Default target - build and run
all: run

# Build all microservices
build:
	@echo "Building microservices..."
	docker-compose build

# Start the distributed pipeline
run:
	@echo "Starting distributed pipeline..."
	docker-compose up -d
	@echo ""
	@echo "Services running:"
	@echo "  - TimescaleDB: localhost:5433"
	@echo "  - NATS:        localhost:4222 (monitoring: localhost:8222)"
	@echo "  - API:         localhost:8080"
	@echo ""
	@echo "Run 'make tui' to view dashboard"
	@echo "Run 'make logs' to view service logs"

# Build and run TUI client
tui:
	@echo "Building TUI client..."
	cd tui && go build -o tui-client .
	@echo "Starting TUI..."
	cd tui && ./tui-client

# Stop all containers
stop:
	@echo "Stopping containers..."
	docker-compose down

# View all service logs
logs:
	docker-compose logs -f

# View specific service logs
logs-ingestion:
	docker-compose logs -f ingestion

logs-processing:
	docker-compose logs -f processing

logs-api:
	docker-compose logs -f api

# Clean build artifacts
clean:
	rm -f tui/tui-client
	rm -f services/processing/libprocess.so
	docker-compose down --rmi local 2>/dev/null || true
