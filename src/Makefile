.PHONY: all clean run

all: libprocess.so trading-pipeline

# Compile C++ to shared library
libprocess.so: process.cpp process.h
	g++ -shared -fPIC -o libprocess.so process.cpp -lpthread

# Build Go binary (links to C++ library)
trading-pipeline: libprocess.so main.go handlers.go binance.go
	CGO_ENABLED=1 go build -o trading-pipeline .

# Run the application
run: all
	LD_LIBRARY_PATH=. ./trading-pipeline

# Clean build artifacts
clean:
	rm -f libprocess.so trading-pipeline
