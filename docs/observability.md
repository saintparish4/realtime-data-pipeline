# Observability & Alerting

This document describes the observability and alerting capabilities of the real-time data pipeline, which provides comprehensive monitoring, metrics collection, and alert management.

## Overview

The observability layer is built on top of Redis and provides:

- **Time-series storage** for real-time metrics
- **Alert state management** with configurable rules
- **Rate limiting** for alert notifications
- **Performance monitoring** and error tracking
- **System health checks**

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Processing    │    │  Observability  │    │      Redis      │
│     Engine      │───▶│    Service      │───▶│   (Time-series  │
│                 │    │                 │    │    Storage)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   Alert Rules   │
                       │   & Notifications│
                       └─────────────────┘
```

## Features

### 1. Time-Series Metrics

Store and query time-series data for various metrics:

- **Price metrics**: Cryptocurrency price data
- **Volume metrics**: Trading volume data
- **Latency metrics**: Performance measurements
- **Error metrics**: Error counts and types
- **Custom metrics**: User-defined metrics

#### Example Usage

```go
// Record a price metric
metric := &models.TimeSeriesMetric{
    Symbol:    "BTCUSDT",
    Type:      models.MetricTypePrice,
    Value:     45000.0,
    Timestamp: time.Now(),
    Labels: map[string]string{
        "exchange": "binance",
        "pair":     "BTCUSDT",
    },
}

err := obsService.RecordMetric(ctx, metric)
```

#### Querying Metrics

```go
// Query price metrics for the last hour
query := &models.TimeSeriesQuery{
    Symbol:    "BTCUSDT",
    Type:      models.MetricTypePrice,
    StartTime: time.Now().Add(-1 * time.Hour),
    EndTime:   time.Now(),
    Limit:     100,
}

metrics, err := obsService.GetMetrics(ctx, query)
```

### 2. Alert Management

Configure and manage alert rules with different severity levels:

- **Info**: Informational alerts
- **Warning**: Warning-level alerts
- **Critical**: Critical alerts that affect system health

#### Alert Rules

```go
// Create a high price alert
rule := &observability.AlertRule{
    ID:          "high_price_btc",
    Name:        "BTC High Price Alert",
    Description: "Alert when BTC price exceeds $50,000",
    Severity:    models.AlertSeverityWarning,
    Symbol:      "BTCUSDT",
    Condition:   "price > threshold",
    Threshold:   50000,
    Enabled:     true,
}

err := obsService.AddAlertRule(rule)
```

#### Alert Status

Alerts can have the following statuses:

- **Active**: Alert is currently triggered
- **Resolved**: Alert condition has been resolved
- **Silenced**: Alert is temporarily silenced

#### Managing Alerts

```go
// Get all active alerts
activeAlerts, err := obsService.GetActiveAlerts(ctx)

// Resolve an alert
err := obsService.ResolveAlert(ctx, alertID)

// Silence an alert
err := obsService.SilenceAlert(ctx, alertID)
```

### 3. Rate Limiting

Prevent alert notification spam with configurable rate limiting:

```go
// Check rate limit before sending notification
exceeded, err := redisClient.CheckAndIncrementRateLimit(ctx, 
    "alert:high_price_btc:BTCUSDT", 1, 5*time.Minute)
if exceeded {
    // Skip notification
}
```

### 4. Performance Monitoring

Track system performance with latency metrics:

```go
// Record processing latency
startTime := time.Now()
// ... perform operation ...
duration := time.Since(startTime)

err := obsService.RecordLatencyMetric(ctx, "message_processing", duration, 
    map[string]string{"symbol": "BTCUSDT"})
```

### 5. Error Tracking

Monitor system errors and failures:

```go
// Record an error
err := obsService.RecordErrorMetric(ctx, "connection_timeout", 
    "Redis connection timeout", map[string]string{"component": "redis"})
```

### 6. System Health

Check overall system health:

```go
health, err := obsService.GetSystemHealth(ctx)
fmt.Printf("System status: %s\n", health["status"])
```

## Configuration

Configure observability settings in `cmd/config.yaml`:

```yaml
observability:
  metrics_retention: "30d"        # How long to keep metrics
  alert_check_interval: "30s"     # How often to check alerts
  notification_rate_limit: "5m"   # Rate limit for notifications
  cleanup_interval: "24h"         # How often to cleanup old data
```

## Redis Data Structure

The observability layer uses the following Redis data structures:

### Time-Series Metrics
- **Key pattern**: `metrics:{type}:{symbol}`
- **Data structure**: Sorted Set (ZSET)
- **Score**: Unix timestamp
- **Member**: JSON-encoded metric data

### Alerts
- **Key pattern**: `alerts:{alert_id}`
- **Data structure**: String (JSON)
- **Active alerts set**: `alerts:active` (Set)

### Rate Limits
- **Key pattern**: `rate_limit:{key}`
- **Data structure**: String (JSON)

### Notifications
- **Key pattern**: `notifications:{notification_id}`
- **Data structure**: String (JSON)

## Integration with Processing Engine

The observability service is integrated into the processing engine to automatically:

1. **Record metrics** for each processed message
2. **Track errors** during processing
3. **Monitor performance** with latency metrics
4. **Trigger alerts** based on configured rules

### Example Integration

```go
// In processing engine
func (e *Engine) handleMessage(ctx context.Context, key, value []byte) error {
    startTime := time.Now()
    
    // Process message...
    
    // Record metrics
    if e.obsService != nil {
        // Record price metric
        priceMetric := &models.TimeSeriesMetric{
            Symbol:    rawData.Symbol,
            Type:      models.MetricTypePrice,
            Value:     rawData.Price,
            Timestamp: rawData.Timestamp,
        }
        e.obsService.RecordMetric(ctx, priceMetric)
        
        // Record processing latency
        processingTime := time.Since(startTime)
        e.obsService.RecordLatencyMetric(ctx, "message_processing", 
            processingTime, map[string]string{"symbol": rawData.Symbol})
    }
    
    return nil
}
```

## Monitoring and Maintenance

### Data Cleanup

The system automatically cleans up old data:

```go
// Start cleanup routine
go obsService.StartCleanupRoutine(ctx)
```

### Metric Statistics

Get statistics about stored metrics:

```go
stats, err := obsService.GetMetricStats(ctx, "BTCUSDT", models.MetricTypePrice)
// Returns: count, oldest_timestamp, newest_timestamp
```

### Redis Commands for Debugging

```bash
# View all metric keys
redis-cli KEYS "metrics:*"

# View price metrics for BTCUSDT
redis-cli ZRANGE metrics:price:BTCUSDT 0 -1

# View active alerts
redis-cli SMEMBERS alerts:active

# View rate limits
redis-cli KEYS "rate_limit:*"

# View notifications
redis-cli KEYS "notifications:*"
```

## Best Practices

1. **Use meaningful labels** for metrics to enable better filtering and aggregation
2. **Set appropriate thresholds** for alerts to avoid false positives
3. **Configure rate limits** to prevent notification spam
4. **Monitor system health** regularly
5. **Clean up old data** to prevent Redis memory issues
6. **Use appropriate severity levels** for alerts

## Example Usage

See `examples/observability-example.go` for a complete example demonstrating all observability features.

## Future Enhancements

- **Prometheus integration** for metrics export
- **Grafana dashboards** for visualization
- **Webhook notifications** for alerts
- **Advanced alert conditions** with complex expressions
- **Metric aggregation** and rollup functions
- **Distributed tracing** support 