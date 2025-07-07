# Kafka Event Streaming for Observability

This document describes the updated Kafka infrastructure that has been transformed from raw data processing to a comprehensive observability event streaming platform.

## Overview

The Kafka infrastructure has been updated to focus on **Event Streaming for Observability** with the following key changes:

- **Raw data ingestion** → **Log/metric ingestion**
- **Processed data** → **Alert events**
- **Perfect for distributed tracing and correlation**

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │    │   Kafka Event   │    │   Observability │
│     Services    │───▶│    Streaming    │───▶│    Consumers    │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   Event Types   │
                       │   & Routing     │
                       └─────────────────┘
```

## Event Types

The new Kafka infrastructure supports structured observability events:

### 1. Log Events (`observability.logs`)
- Application logs with structured metadata
- Error tracking and debugging
- Performance monitoring logs

### 2. Metric Events (`observability.metrics`)
- Time-series metrics and measurements
- Performance indicators
- Business metrics

### 3. Alert Events (`observability.alerts`)
- System alerts and notifications
- Threshold violations
- Incident management

### 4. Trace Events (`observability.traces`)
- Distributed tracing spans
- Performance profiling
- Request flow tracking

### 5. Health Events (`observability.health`)
- Component health checks
- System status monitoring
- Availability tracking

## Event Structure

All events follow a consistent structure:

```go
type ObservabilityEvent struct {
    ID           string                 `json:"id"`
    Type         EventType              `json:"type"`
    Level        EventLevel             `json:"level"`
    Service      string                 `json:"service"`
    Component    string                 `json:"component"`
    Message      string                 `json:"message"`
    Timestamp    time.Time              `json:"timestamp"`
    CorrelationID string                `json:"correlation_id,omitempty"`
    TraceID      string                 `json:"trace_id,omitempty"`
    SpanID       string                 `json:"span_id,omitempty"`
    Labels       map[string]string      `json:"labels,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    Data         interface{}            `json:"data,omitempty"`
}
```

## Event Levels

Events are categorized by severity:

- **Debug**: Detailed debugging information
- **Info**: General information messages
- **Warning**: Warning conditions
- **Error**: Error conditions
- **Fatal**: Critical system failures

## Distributed Tracing

The infrastructure supports distributed tracing with:

- **Trace ID**: Unique identifier for a request flow
- **Span ID**: Individual operation within a trace
- **Correlation ID**: Links related events across services
- **Parent ID**: Links child spans to parent spans

### Example Trace Flow

```
Request → Service A → Service B → Service C
   ↓         ↓         ↓         ↓
TraceID   SpanID    SpanID    SpanID
   ↓         ↓         ↓         ↓
Correlation ID links all events
```

## Configuration

### Kafka Topics

```yaml
kafka:
  topics:
    observability_events: "observability.events"
    logs: "observability.logs"
    metrics: "observability.metrics"
    alerts: "observability.alerts"
    traces: "observability.traces"
    health_checks: "observability.health"
```

### Event Streaming Settings

```yaml
observability:
  event_streaming:
    correlation_id_header: "X-Correlation-ID"
    trace_id_header: "X-Trace-ID"
    span_id_header: "X-Span-ID"
    service_name: "realtime-data-pipeline"
    enable_distributed_tracing: true
    enable_correlation_tracking: true
    trace_sampling_rate: 0.1
```

## Usage Examples

### Publishing Events

```go
// Initialize producer
producer := kafka.NewProducer(brokers, topic, serviceName)

// Publish log event
err := producer.PublishLog(ctx, kafka.EventLevelInfo, "component", 
    "Processing data", labels, metadata)

// Publish metric event
err := producer.PublishMetric(ctx, "component", "latency", 150.5, 
    labels, metadata)

// Publish alert event
err := producer.PublishAlert(ctx, alertID, alertName, message, 
    kafka.EventLevelWarning, labels, metadata)

// Publish trace event
err := producer.PublishTrace(ctx, traceID, spanID, "operation", 
    duration, labels, metadata)
```

### Consuming Events

```go
// Initialize consumer
consumer := kafka.NewConsumer(brokers, topic, groupID)

// Consume all events
err := consumer.ConsumeEvents(ctx, func(ctx context.Context, event *kafka.ObservabilityEvent) error {
    // Process event
    return nil
})

// Consume specific event types
err := consumer.ConsumeLogs(ctx, logHandler)
err := consumer.ConsumeMetrics(ctx, metricHandler)
err := consumer.ConsumeAlerts(ctx, alertHandler)
err := consumer.ConsumeTraces(ctx, traceHandler)

// Consume with filtering
filter := &kafka.EventFilter{
    Types: []kafka.EventType{kafka.EventTypeLog, kafka.EventTypeError},
    Levels: []kafka.EventLevel{kafka.EventLevelError, kafka.EventLevelFatal},
}
err := consumer.ConsumeEventsWithFilter(ctx, handler, filter)
```

### Event Streaming Service

```go
// Initialize service
eventService, err := observability.NewEventStreamingService(cfg, redisClient)

// Start distributed trace
traceID, spanID := eventService.StartTrace(ctx, "operation", labels)

// Publish correlated events
err = eventService.PublishLogEvent(ctx, level, component, message, correlationID, labels)
err = eventService.PublishMetricEvent(ctx, component, metricName, value, correlationID, labels)
err = eventService.PublishAlertEvent(ctx, alert, correlationID)

// End trace
err = eventService.EndTrace(ctx, traceID, spanID, labels)

// Retrieve correlated events
events, err := eventService.GetCorrelatedEvents(ctx, correlationID)
```

## Event Correlation

Events can be correlated using:

1. **Correlation ID**: Links related events across services
2. **Trace ID**: Groups events within a request flow
3. **Span ID**: Identifies specific operations
4. **Labels**: Custom key-value pairs for filtering

### Correlation Example

```go
// Generate correlation ID
correlationID := uuid.New().String()

// Publish related events
eventService.PublishLogEvent(ctx, level, "service-a", "Request received", correlationID, labels)
eventService.PublishMetricEvent(ctx, "service-a", "request_count", 1, correlationID, labels)
eventService.PublishLogEvent(ctx, level, "service-b", "Processing request", correlationID, labels)
eventService.PublishAlertEvent(ctx, alert, correlationID)

// Retrieve all correlated events
events, err := eventService.GetCorrelatedEvents(ctx, correlationID)
```

## Event Filtering and Routing

The infrastructure supports sophisticated event filtering:

### By Event Type
```go
consumer.ConsumeLogs(ctx, handler)
consumer.ConsumeMetrics(ctx, handler)
consumer.ConsumeAlerts(ctx, handler)
consumer.ConsumeTraces(ctx, handler)
```

### By Event Level
```go
consumer.ConsumeErrors(ctx, handler) // Error and Fatal levels
```

### By Service/Component
```go
consumer.ConsumeByService(ctx, []string{"service-a", "service-b"}, handler)
consumer.ConsumeByComponent(ctx, []string{"processor", "websocket"}, handler)
```

### By Correlation ID
```go
consumer.ConsumeByCorrelationID(ctx, correlationID, handler)
```

## Event Storage and Retrieval

Events are stored in Redis for:

- **Correlation analysis**: Linking related events
- **Trace reconstruction**: Building complete request flows
- **Historical analysis**: Long-term event analysis
- **Debugging**: Investigating issues

### Storage Patterns

- **Events**: `events:{type}:{id}` (24-hour TTL)
- **Logs**: `logs:{level}:{id}` (24-hour TTL, errors 7 days)
- **Traces**: `traces:{traceID}:{spanID}` (7-day TTL)
- **Alerts**: `alert_events:{id}` (90-day TTL)

## Monitoring and Observability

The event streaming infrastructure provides:

### Health Monitoring
- Component health checks
- System status tracking
- Performance metrics

### Error Tracking
- Error event publishing
- Error correlation
- Error rate monitoring

### Performance Monitoring
- Latency metrics
- Throughput measurements
- Resource utilization

### Distributed Tracing
- Request flow visualization
- Performance profiling
- Bottleneck identification

## Migration from Legacy Infrastructure

The new infrastructure maintains backward compatibility:

### Legacy Topics
- `crypto.raw` → `observability.logs` (for ingestion logs)
- `crypto.processed` → `observability.alerts` (for alert events)

### Legacy Support
- Original producer/consumer interfaces still work
- Gradual migration path available
- Configuration supports both old and new topics

## Best Practices

### Event Publishing
1. **Use structured events**: Include all relevant metadata
2. **Set appropriate levels**: Use correct severity levels
3. **Include correlation IDs**: Link related events
4. **Add meaningful labels**: Enable filtering and analysis
5. **Validate events**: Ensure data quality

### Event Consumption
1. **Use appropriate filters**: Consume only needed events
2. **Handle errors gracefully**: Implement proper error handling
3. **Monitor consumer lag**: Track processing performance
4. **Scale consumers**: Use multiple consumer groups
5. **Implement retry logic**: Handle transient failures

### Distributed Tracing
1. **Start traces early**: Begin tracing at request entry
2. **Propagate context**: Pass trace IDs across services
3. **Add meaningful spans**: Create spans for significant operations
4. **Include metadata**: Add relevant context to spans
5. **Sample appropriately**: Use sampling for high-volume traces

### Event Correlation
1. **Use consistent correlation IDs**: Maintain ID across services
2. **Include context**: Add relevant labels and metadata
3. **Store events**: Persist events for analysis
4. **Query efficiently**: Use appropriate storage patterns
5. **Clean up old data**: Implement retention policies

## Example Implementation

See `examples/event-streaming/main.go` for a comprehensive example demonstrating:

- Distributed tracing
- Event correlation
- Structured event publishing
- Event filtering and routing
- Direct Kafka producer/consumer usage
- Event storage and retrieval
- Health monitoring and error tracking

## Future Enhancements

- **Event Schema Registry**: Centralized event schema management
- **Event Streaming Analytics**: Real-time event analysis
- **Advanced Filtering**: Complex event filtering expressions
- **Event Replay**: Historical event replay capabilities
- **Event Aggregation**: Automatic event aggregation and rollup
- **Integration with APM**: Integration with Application Performance Monitoring tools 