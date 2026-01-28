FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc g++ musl-dev

WORKDIR /app
COPY services/processing/ ./

# Build C++ shared library
RUN g++ -shared -fPIC -o libprocess.so process.cpp -lpthread

# Build Go binary with CGO
RUN CGO_ENABLED=1 go build -o processing .

FROM alpine:latest
RUN apk add --no-cache libstdc++ libgcc
WORKDIR /app
COPY --from=builder /app/processing .
COPY --from=builder /app/libprocess.so .
ENV LD_LIBRARY_PATH=/app
CMD ["./processing"]
