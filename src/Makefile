.PHONY: all clean run tui

all: libprocess.so trading-pipeline tui-client

# Compile C++ to shared library
libprocess.so: process.cpp process.h
	g++ -shared -fPIC -o libprocess.so process.cpp -lpthread

# Build server (links to C++ library)
trading-pipeline: libprocess.so main.go handlers.go binance.go tui.go
	CGO_ENABLED=1 go build -o trading-pipeline .

# Build TUI client (standalone, no CGO)
tui-client:
	cd cmd/tui && go build -o ../../tui-client .

# Run the server
run: trading-pipeline
	LD_LIBRARY_PATH=. ./trading-pipeline

# Run the TUI client
tui: tui-client
	./tui-client

# Clean build artifacts
clean:
	rm -f libprocess.so trading-pipeline tui-client
