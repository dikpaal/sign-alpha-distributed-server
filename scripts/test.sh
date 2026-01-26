#!/bin/bash

# ============================================
# Trading Pipeline - Comprehensive CLI Test
# ============================================
# Run from project root: ./scripts/test.sh
# ============================================

# Config
BASE_URL="http://localhost:8080"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
SERVER_DIR="$PROJECT_DIR/server"
SERVER_PID=""
REFRESH_RATE=1
SAMPLES_TO_SHOW=15

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Cleanup on exit
cleanup() {
    echo ""
    echo -e "${YELLOW}Shutting down...${NC}"
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null
        echo -e "${GREEN}Server stopped.${NC}"
    fi
    exit 0
}
trap cleanup SIGINT SIGTERM

# Start server if not running
start_server() {
    if curl -s "$BASE_URL/api/price" > /dev/null 2>&1; then
        echo -e "${GREEN}Server already running.${NC}"
        return
    fi

    echo -e "${YELLOW}Starting server...${NC}"

    # Build if needed
    if [ ! -f "$SERVER_DIR/trading-pipeline" ] || [ ! -f "$SERVER_DIR/libprocess.so" ]; then
        echo -e "${YELLOW}Building project...${NC}"
        cd "$PROJECT_DIR" && make build-server > /dev/null 2>&1
    fi

    # Start server in background
    cd "$SERVER_DIR"
    LD_LIBRARY_PATH="$SERVER_DIR" ./trading-pipeline > /dev/null 2>&1 &
    SERVER_PID=$!

    # Wait for server to be ready
    for i in {1..10}; do
        if curl -s "$BASE_URL/api/price" > /dev/null 2>&1; then
            echo -e "${GREEN}Server started (PID: $SERVER_PID)${NC}"
            sleep 2  # Let some prices accumulate
            return
        fi
        sleep 1
    done

    echo -e "${RED}Failed to start server${NC}"
    exit 1
}

# Test 1: Raw API responses
test_raw_api() {
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo -e "${CYAN}  TEST 1: Raw API Responses${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo ""

    echo -e "${BLUE}GET /api/price${NC}"
    curl -s "$BASE_URL/api/price" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/api/price"
    echo ""

    echo -e "${BLUE}GET /api/stats${NC}"
    curl -s "$BASE_URL/api/stats" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/api/stats"
    echo ""
}

# Test 2: Price change tracking
test_price_changes() {
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo -e "${CYAN}  TEST 2: Price Change Tracking (5 samples)${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo ""

    PREV_PRICE=0
    for i in {1..5}; do
        PRICE=$(curl -s "$BASE_URL/api/price" | grep -o '"price":[0-9.]*' | cut -d':' -f2)
        TIME=$(date +"%H:%M:%S")

        if [ "$PREV_PRICE" != "0" ]; then
            DIFF=$(echo "$PRICE - $PREV_PRICE" | bc 2>/dev/null || echo "0")
            if (( $(echo "$DIFF > 0" | bc -l 2>/dev/null || echo 0) )); then
                echo -e "[$TIME] \$${PRICE} ${GREEN}▲ +${DIFF}${NC}"
            elif (( $(echo "$DIFF < 0" | bc -l 2>/dev/null || echo 0) )); then
                echo -e "[$TIME] \$${PRICE} ${RED}▼ ${DIFF}${NC}"
            else
                echo -e "[$TIME] \$${PRICE} ━ 0.00"
            fi
        else
            echo -e "[$TIME] \$${PRICE}"
        fi

        PREV_PRICE=$PRICE
        sleep 1
    done
    echo ""
}

# Test 3: C++ processor stats
test_cpp_stats() {
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo -e "${CYAN}  TEST 3: C++ Signal Processing Stats${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo ""

    STATS=$(curl -s "$BASE_URL/api/stats")
    MA=$(echo "$STATS" | grep -o '"moving_average":[0-9.]*' | cut -d':' -f2)
    HIGH=$(echo "$STATS" | grep -o '"high":[0-9.]*' | cut -d':' -f2)
    LOW=$(echo "$STATS" | grep -o '"low":[0-9.]*' | cut -d':' -f2)
    PRICE=$(curl -s "$BASE_URL/api/price" | grep -o '"price":[0-9.]*' | cut -d':' -f2)

    echo -e "  Current Price:    ${GREEN}\$${PRICE}${NC}"
    echo -e "  Moving Average:   ${BLUE}\$${MA}${NC}"
    echo -e "  Session High:     ${GREEN}\$${HIGH}${NC}"
    echo -e "  Session Low:      ${RED}\$${LOW}${NC}"

    # Calculate spread
    if [ -n "$HIGH" ] && [ -n "$LOW" ]; then
        SPREAD=$(echo "$HIGH - $LOW" | bc 2>/dev/null || echo "N/A")
        echo -e "  High-Low Spread:  ${YELLOW}\$${SPREAD}${NC}"
    fi
    echo ""
}

# Test 4: Live streaming dashboard
test_live_stream() {
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo -e "${CYAN}  TEST 4: Live Streaming (${SAMPLES_TO_SHOW} updates)${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo ""
    echo "  TIME     | PRICE       | MA          | HIGH        | LOW"
    echo "  ---------|-------------|-------------|-------------|------------"

    for i in $(seq 1 $SAMPLES_TO_SHOW); do
        PRICE=$(curl -s "$BASE_URL/api/price" | grep -o '"price":[0-9.]*' | cut -d':' -f2)
        STATS=$(curl -s "$BASE_URL/api/stats")
        MA=$(echo "$STATS" | grep -o '"moving_average":[0-9.]*' | cut -d':' -f2)
        HIGH=$(echo "$STATS" | grep -o '"high":[0-9.]*' | cut -d':' -f2)
        LOW=$(echo "$STATS" | grep -o '"low":[0-9.]*' | cut -d':' -f2)
        TIME=$(date +"%H:%M:%S")

        printf "  %s | \$%-9.2f | \$%-9.2f | \$%-9.2f | \$%-9.2f\n" \
            "$TIME" "$PRICE" "$MA" "$HIGH" "$LOW"

        sleep $REFRESH_RATE
    done
    echo ""
}

# Test 5: API response times
test_latency() {
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo -e "${CYAN}  TEST 5: API Response Latency${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════${NC}"
    echo ""

    echo -e "  ${BLUE}/api/price:${NC}"
    for i in {1..3}; do
        TIME_MS=$(curl -s -o /dev/null -w "%{time_total}" "$BASE_URL/api/price")
        TIME_MS=$(echo "$TIME_MS * 1000" | bc 2>/dev/null || echo "N/A")
        echo "    Request $i: ${TIME_MS}ms"
    done

    echo ""
    echo -e "  ${BLUE}/api/stats:${NC}"
    for i in {1..3}; do
        TIME_MS=$(curl -s -o /dev/null -w "%{time_total}" "$BASE_URL/api/stats")
        TIME_MS=$(echo "$TIME_MS * 1000" | bc 2>/dev/null || echo "N/A")
        echo "    Request $i: ${TIME_MS}ms"
    done
    echo ""
}

# Main
main() {
    clear
    echo -e "${GREEN}"
    echo "  ╔═══════════════════════════════════════════╗"
    echo "  ║   TRADING PIPELINE - CLI TEST SUITE       ║"
    echo "  ║   Real-time Crypto Data Pipeline          ║"
    echo "  ╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    start_server

    test_raw_api
    test_price_changes
    test_cpp_stats
    test_live_stream
    test_latency

    echo -e "${GREEN}═══════════════════════════════════════════${NC}"
    echo -e "${GREEN}  All tests complete!${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════${NC}"

    cleanup
}

main
