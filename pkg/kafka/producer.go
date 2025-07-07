package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// EventType represents the type of observability event
type EventType string

const (
	EventTypeLog    EventType = "log"
	EventTypeMetric EventType = "metric"
	EventTypeAlert  EventType = "alert"
	EventTypeTrace  EventType = "trace"
	EventTypeHealth EventType = "health"
)

// EventLevel represents the severity level of an event
type EventLevel string

const (
	EventLevelDebug   EventLevel = "debug"
	EventLevelInfo    EventLevel = "info"
	EventLevelWarning EventLevel = "warning"
	EventLevelError   EventLevel = "error"
	EventLevelFatal   EventLevel = "fatal"
)

// ObservabilityEvent represents a structured event for observability
type ObservabilityEvent struct {
	ID            string                 `json:"id"`
	Type          EventType              `json:"type"`
	Level         EventLevel             `json:"level"`
	Service       string                 `json:"service"`
	Component     string                 `json:"component"`
	Message       string                 `json:"message"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Data          interface{}            `json:"data,omitempty"`
}

// Producer handles publishing observability events to Kafka
type Producer struct {
	writer  *kafka.Writer
	service string
}

// NewProducer creates a new Kafka producer for observability events
func NewProducer(brokers []string, topic string, service string) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer:  writer,
		service: service,
	}
}

// PublishEvent publishes an observability event to Kafka
func (p *Producer) PublishEvent(ctx context.Context, event *ObservabilityEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Service == "" {
		event.Service = p.service
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Use event type as the key for partitioning
	key := []byte(event.Type)

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "event_level", Value: []byte(event.Level)},
			{Key: "service", Value: []byte(event.Service)},
			{Key: "correlation_id", Value: []byte(event.CorrelationID)},
		},
	})
}

// PublishLog publishes a log event
func (p *Producer) PublishLog(ctx context.Context, level EventLevel, component, message string, labels map[string]string, metadata map[string]interface{}) error {
	event := &ObservabilityEvent{
		Type:      EventTypeLog,
		Level:     level,
		Component: component,
		Message:   message,
		Labels:    labels,
		Metadata:  metadata,
	}
	return p.PublishEvent(ctx, event)
}

// PublishMetric publishes a metric event
func (p *Producer) PublishMetric(ctx context.Context, component, metricName string, value float64, labels map[string]string, metadata map[string]interface{}) error {
	event := &ObservabilityEvent{
		Type:      EventTypeMetric,
		Level:     EventLevelInfo,
		Component: component,
		Message:   fmt.Sprintf("Metric: %s = %f", metricName, value),
		Labels:    labels,
		Metadata:  metadata,
		Data: map[string]interface{}{
			"metric_name": metricName,
			"value":       value,
		},
	}
	return p.PublishEvent(ctx, event)
}

// PublishAlert publishes an alert event
func (p *Producer) PublishAlert(ctx context.Context, alertID, alertName, message string, severity EventLevel, labels map[string]string, metadata map[string]interface{}) error {
	event := &ObservabilityEvent{
		Type:      EventTypeAlert,
		Level:     severity,
		Component: "alerting",
		Message:   message,
		Labels:    labels,
		Metadata:  metadata,
		Data: map[string]interface{}{
			"alert_id":   alertID,
			"alert_name": alertName,
		},
	}
	return p.PublishEvent(ctx, event)
}

// PublishTrace publishes a trace event for distributed tracing
func (p *Producer) PublishTrace(ctx context.Context, traceID, spanID, operation string, duration time.Duration, labels map[string]string, metadata map[string]interface{}) error {
	event := &ObservabilityEvent{
		Type:      EventTypeTrace,
		Level:     EventLevelInfo,
		Component: "tracing",
		Message:   fmt.Sprintf("Trace: %s took %v", operation, duration),
		TraceID:   traceID,
		SpanID:    spanID,
		Labels:    labels,
		Metadata:  metadata,
		Data: map[string]interface{}{
			"operation":   operation,
			"duration_ms": duration.Milliseconds(),
		},
	}
	return p.PublishEvent(ctx, event)
}

// PublishHealth publishes a health check event
func (p *Producer) PublishHealth(ctx context.Context, component string, status string, details map[string]interface{}) error {
	level := EventLevelInfo
	if status != "healthy" {
		level = EventLevelWarning
	}

	event := &ObservabilityEvent{
		Type:      EventTypeHealth,
		Level:     level,
		Component: component,
		Message:   fmt.Sprintf("Health check: %s is %s", component, status),
		Metadata:  details,
		Data: map[string]interface{}{
			"status": status,
		},
	}
	return p.PublishEvent(ctx, event)
}

// PublishError publishes an error event
func (p *Producer) PublishError(ctx context.Context, component, errorType, errorMessage string, labels map[string]string, metadata map[string]interface{}) error {
	event := &ObservabilityEvent{
		Type:      EventTypeLog,
		Level:     EventLevelError,
		Component: component,
		Message:   errorMessage,
		Labels:    labels,
		Metadata:  metadata,
		Data: map[string]interface{}{
			"error_type": errorType,
		},
	}
	return p.PublishEvent(ctx, event)
}

// Publish publishes raw data with a key and value
func (p *Producer) Publish(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: data,
	})
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	return p.writer.Close()
}
