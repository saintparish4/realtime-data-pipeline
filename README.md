# Realtime Data Pipeline

A high-throughput, low-latency pipeline for streaming real-time price updates using Go, Kafka, and WebSockets.

## Key Features
- WebSocket multiplexing: subscribe to multiple symbols per connection
- Batch processing: efficient delivery with ≤100ms p95 latency
- Scalable: handles 100K+ price updates daily

## WebSocket API

**Endpoint:** `/ws`

### Subscribe/Unsubscribe
Send JSON messages to manage subscriptions:
```json
{ "action": "subscribe", "symbols": ["BTCUSDT", "ETHUSDT"] }
{ "action": "unsubscribe", "symbols": ["BTCUSDT"] }
```

### Receiving Updates
You will receive batches of price updates as JSON arrays:
```json
[
  { "symbol": "BTCUSDT", "price": 45000.0, "volume": 100.5, "timestamp": "2024-06-01T12:00:00Z", "source": "binance" },
  ...
]
```

## Performance
- Batching interval: 50ms (configurable)
- Non-blocking delivery to each client
- Designed for high throughput and low latency

## Quick Start
1. Start Kafka and dependencies (`docker-compose up`)
2. Run the Go services
3. Connect a WebSocket client to `/ws` and subscribe to symbols

## License
MIT