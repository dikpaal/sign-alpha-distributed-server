FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY services/ingestion/ ./
RUN go mod download
RUN CGO_ENABLED=0 go build -o ingestion .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/ingestion .
CMD ["./ingestion"]
