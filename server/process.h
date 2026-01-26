#ifndef PROCESS_H
#define PROCESS_H

#ifdef __cplusplus
extern "C" {
#endif

// Add a new price to the buffer
void add_price(double price);

// Get the simple moving average of buffered prices
double get_moving_average(void);

// Get the highest price seen
double get_high(void);

// Get the lowest price seen
double get_low(void);

// Reset all data
void reset_processor(void);

#ifdef __cplusplus
}
#endif

#endif // PROCESS_H
