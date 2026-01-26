.PHONY: all build clean run tui test

# Default target
all: build

# Build both server and TUI
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

# Run the server
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
