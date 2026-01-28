#include "process.h"
#include <vector>
#include <mutex>
#include <limits>

// Buffer size for moving average calculation
const int BUFFER_SIZE = 20;

// Thread-safe price processor
static std::mutex mtx;
static std::vector<double> price_buffer;
static double high_price = 0.0;
static double low_price = std::numeric_limits<double>::max();

extern "C" {

void add_price(double price) {
    std::lock_guard<std::mutex> lock(mtx);

    // Update high/low
    if (price > high_price) {
        high_price = price;
    }
    if (price < low_price) {
        low_price = price;
    }

    // Add to circular buffer
    if (price_buffer.size() >= BUFFER_SIZE) {
        price_buffer.erase(price_buffer.begin());
    }
    price_buffer.push_back(price);
}

double get_moving_average(void) {
    std::lock_guard<std::mutex> lock(mtx);

    if (price_buffer.empty()) {
        return 0.0;
    }

    double sum = 0.0;
    for (double p : price_buffer) {
        sum += p;
    }
    return sum / price_buffer.size();
}

double get_high(void) {
    std::lock_guard<std::mutex> lock(mtx);
    return high_price;
}

double get_low(void) {
    std::lock_guard<std::mutex> lock(mtx);
    // Return 0 if no prices have been added yet
    if (low_price == std::numeric_limits<double>::max()) {
        return 0.0;
    }
    return low_price;
}

void reset_processor(void) {
    std::lock_guard<std::mutex> lock(mtx);
    price_buffer.clear();
    high_price = 0.0;
    low_price = std::numeric_limits<double>::max();
}

} // extern "C"
