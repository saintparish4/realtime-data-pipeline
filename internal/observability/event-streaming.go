package observability

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/saintparish4/realtime-data-pipeline/internal/config"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// EventStreamingService handles observability event streaming with Kafka
type EventStreamingService struct {
	config      *config.Config
	producer    *kafka.Producer
	redisClient *redis.Client

	// Correlation tracking
	correlationMap map[string]string
	correlationMux sync.RWMutex

	// Trace tracking
	traceMap map[string]*TraceContext
	traceMux sync.RWMutex
}

// TraceContext represents a distributed trace context
type TraceContext struct {
	TraceID   string
	SpanID    string
	ParentID  string
	StartTime time.Time
	Service   string
	Operation string
	Labels    map[string]string
	Metadata  map[string]interface{}
}

// NewEventStreamingService creates a new event streaming service
func NewEventStreamingService(cfg *config.Config, redisClient *redis.Client) (*EventStreamingService, error) {
	producer := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topics.ObservabilityEvents,
		cfg.Observability.EventStreaming.ServiceName,
	)

	return &EventStreamingService{
		config:         cfg,
		producer:       producer,
		redisClient:    redisClient,
		correlationMap: make(map[string]string),
		traceMap:       make(map[string]*TraceContext),
	}, nil
}

// StartTrace starts a new distributed trace
func (s *EventStreamingService) StartTrace(ctx context.Context, operation string, labels map[string]string) (string, string) {
	traceID := uuid.New().String()
	spanID := uuid.New().String()

	traceCtx := &TraceContext{
		TraceID:   traceID,
		SpanID:    spanID,
		StartTime: time.Now(),
		Service:   s.config.Observability.EventStreaming.ServiceName,
		Operation: operation,
		Labels:    labels,
		Metadata:  make(map[string]interface{}),
	}

	s.traceMux.Lock()
	s.traceMap[traceID] = traceCtx
	s.traceMux.Unlock()

	// Publish trace start event
	s.producer.PublishTrace(ctx, traceID, spanID, operation, 0, labels, map[string]interface{}{
		"event": "trace_start",
	})

	return traceID, spanID
}

// EndTrace ends a distributed trace
func (s *EventStreamingService) EndTrace(ctx context.Context, traceID, spanID string, labels map[string]string) error {
	s.traceMux.Lock()
	traceCtx, exists := s.traceMap[traceID]
	s.traceMux.Unlock()

	if !exists {
		return fmt.Errorf("trace %s not found", traceID)
	}

	duration := time.Since(traceCtx.StartTime)

	// Publish trace end event
	err := s.producer.PublishTrace(ctx, traceID, spanID, traceCtx.Operation, duration, labels, map[string]interface{}{
		"event":       "trace_end",
		"duration_ms": duration.Milliseconds(),
	})

	// Clean up trace context
	s.traceMux.Lock()
	delete(s.traceMap, traceID)
	s.traceMux.Unlock()

	return err
}

// AddCorrelationID adds a correlation ID to track related events
func (s *EventStreamingService) AddCorrelationID(correlationID, context string) {
	s.correlationMux.Lock()
	s.correlationMap[correlationID] = context
	s.correlationMux.Unlock()
}

// GetCorrelationContext gets the context for a correlation ID
func (s *EventStreamingService) GetCorrelationContext(correlationID string) (string, bool) {
	s.correlationMux.RLock()
	defer s.correlationMux.RUnlock()

	context, exists := s.correlationMap[correlationID]
	return context, exists
}

// PublishLogEvent publishes a log event with correlation tracking
func (s *EventStreamingService) PublishLogEvent(ctx context.Context, level kafka.EventLevel, component, message string, correlationID string, labels map[string]string) error {
	// Add correlation context if available
	if correlationID != "" {
		if context, exists := s.GetCorrelationContext(correlationID); exists {
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["correlation_context"] = context
		}
	}

	return s.producer.PublishLog(ctx, level, component, message, labels, map[string]interface{}{
		"correlation_id": correlationID,
	})
}

// PublishMetricEvent publishes a metric event
func (s *EventStreamingService) PublishMetricEvent(ctx context.Context, component, metricName string, value float64, correlationID string, labels map[string]string) error {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["correlation_id"] = correlationID

	return s.producer.PublishMetric(ctx, component, metricName, value, labels, map[string]interface{}{
		"correlation_id": correlationID,
	})
}

// PublishAlertEvent publishes an alert event
func (s *EventStreamingService) PublishAlertEvent(ctx context.Context, alert *models.Alert, correlationID string) error {
	level := kafka.EventLevelInfo
	switch alert.Severity {
	case models.AlertSeverityWarning:
		level = kafka.EventLevelWarning
	case models.AlertSeverityCritical:
		level = kafka.EventLevelError
	}

	labels := map[string]string{
		"alert_id":       alert.ID,
		"symbol":         alert.Symbol,
		"severity":       string(alert.Severity),
		"status":         string(alert.Status),
		"correlation_id": correlationID,
	}

	metadata := map[string]interface{}{
		"alert":          alert,
		"correlation_id": correlationID,
	}

	return s.producer.PublishAlert(ctx, alert.ID, alert.Name, alert.Description, level, labels, metadata)
}

// PublishHealthEvent publishes a health check event
func (s *EventStreamingService) PublishHealthEvent(ctx context.Context, component string, status string, details map[string]interface{}) error {
	return s.producer.PublishHealth(ctx, component, status, details)
}

// PublishErrorEvent publishes an error event
func (s *EventStreamingService) PublishErrorEvent(ctx context.Context, component, errorType, errorMessage string, correlationID string, labels map[string]string) error {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["correlation_id"] = correlationID

	return s.producer.PublishError(ctx, component, errorType, errorMessage, labels, map[string]interface{}{
		"correlation_id": correlationID,
	})
}

// StartEventConsumer starts consuming observability events
func (s *EventStreamingService) StartEventConsumer(ctx context.Context, groupID string) error {
	consumer := kafka.NewConsumer(
		s.config.Kafka.Brokers,
		s.config.Kafka.Topics.ObservabilityEvents,
		groupID,
	)

	return consumer.ConsumeEvents(ctx, s.handleEvent)
}

// StartLogConsumer starts consuming log events
func (s *EventStreamingService) StartLogConsumer(ctx context.Context, groupID string) error {
	consumer := kafka.NewConsumer(
		s.config.Kafka.Brokers,
		s.config.Kafka.Topics.Logs,
		groupID,
	)

	return consumer.ConsumeLogs(ctx, s.handleLogEvent)
}

// StartMetricConsumer starts consuming metric events
func (s *EventStreamingService) StartMetricConsumer(ctx context.Context, groupID string) error {
	consumer := kafka.NewConsumer(
		s.config.Kafka.Brokers,
		s.config.Kafka.Topics.Metrics,
		groupID,
	)

	return consumer.ConsumeMetrics(ctx, s.handleMetricEvent)
}

// StartAlertConsumer starts consuming alert events
func (s *EventStreamingService) StartAlertConsumer(ctx context.Context, groupID string) error {
	consumer := kafka.NewConsumer(
		s.config.Kafka.Brokers,
		s.config.Kafka.Topics.Alerts,
		groupID,
	)

	return consumer.ConsumeAlerts(ctx, s.handleAlertEvent)
}

// StartTraceConsumer starts consuming trace events
func (s *EventStreamingService) StartTraceConsumer(ctx context.Context, groupID string) error {
	consumer := kafka.NewConsumer(
		s.config.Kafka.Brokers,
		s.config.Kafka.Topics.Traces,
		groupID,
	)

	return consumer.ConsumeTraces(ctx, s.handleTraceEvent)
}

// handleEvent handles generic observability events
func (s *EventStreamingService) handleEvent(ctx context.Context, event *kafka.ObservabilityEvent) error {
	log.Printf("Received event: %s - %s - %s", event.Type, event.Level, event.Message)

	// Store event in Redis for correlation analysis
	key := fmt.Sprintf("events:%s:%s", event.Type, event.ID)
	if err := s.redisClient.Set(ctx, key, event, 24*time.Hour); err != nil {
		log.Printf("Failed to store event in Redis: %v", err)
	}

	// Add to correlation tracking if correlation ID exists
	if event.CorrelationID != "" {
		s.AddCorrelationID(event.CorrelationID, fmt.Sprintf("%s:%s", event.Type, event.Component))
	}

	return nil
}

// handleLogEvent handles log events specifically
func (s *EventStreamingService) handleLogEvent(ctx context.Context, event *kafka.ObservabilityEvent) error {
	log.Printf("Log event: %s - %s - %s", event.Level, event.Component, event.Message)

	// Store log in Redis with TTL based on level
	ttl := 24 * time.Hour
	if event.Level == kafka.EventLevelError || event.Level == kafka.EventLevelFatal {
		ttl = 7 * 24 * time.Hour // Keep errors longer
	}

	key := fmt.Sprintf("logs:%s:%s", event.Level, event.ID)
	return s.redisClient.Set(ctx, key, event, ttl)
}

// handleMetricEvent handles metric events specifically
func (s *EventStreamingService) handleMetricEvent(ctx context.Context, event *kafka.ObservabilityEvent) error {
	log.Printf("Metric event: %s - %s", event.Component, event.Message)

	// Convert to TimeSeriesMetric and store
	if data, ok := event.Data.(map[string]interface{}); ok {
		if metricName, ok := data["metric_name"].(string); ok {
			if value, ok := data["value"].(float64); ok {
				metric := &models.TimeSeriesMetric{
					Symbol:    event.Labels["symbol"],
					Type:      models.MetricTypeCustom,
					Value:     value,
					Timestamp: event.Timestamp,
					Labels:    event.Labels,
					Metadata: map[string]interface{}{
						"metric_name": metricName,
						"component":   event.Component,
					},
				}

				return s.redisClient.StoreTimeSeriesMetric(ctx, metric)
			}
		}
	}

	return nil
}

// handleAlertEvent handles alert events specifically
func (s *EventStreamingService) handleAlertEvent(ctx context.Context, event *kafka.ObservabilityEvent) error {
	log.Printf("Alert event: %s - %s", event.Level, event.Message)

	// Store alert in Redis
	key := fmt.Sprintf("alert_events:%s", event.ID)
	return s.redisClient.Set(ctx, key, event, 90*24*time.Hour) // 90 days retention
}

// handleTraceEvent handles trace events specifically
func (s *EventStreamingService) handleTraceEvent(ctx context.Context, event *kafka.ObservabilityEvent) error {
	log.Printf("Trace event: %s - %s", event.TraceID, event.Message)

	// Store trace in Redis
	key := fmt.Sprintf("traces:%s:%s", event.TraceID, event.SpanID)
	return s.redisClient.Set(ctx, key, event, 7*24*time.Hour) // 7 days retention
}

// GetCorrelatedEvents retrieves events with the same correlation ID
func (s *EventStreamingService) GetCorrelatedEvents(ctx context.Context, correlationID string) ([]*kafka.ObservabilityEvent, error) {
	pattern := fmt.Sprintf("events:*:*")
	keys, err := s.redisClient.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	var events []*kafka.ObservabilityEvent
	for _, key := range keys {
		var event kafka.ObservabilityEvent
		if err := s.redisClient.Get(ctx, key, &event); err == nil {
			if event.CorrelationID == correlationID {
				events = append(events, &event)
			}
		}
	}

	return events, nil
}

// GetTraceEvents retrieves all events for a specific trace
func (s *EventStreamingService) GetTraceEvents(ctx context.Context, traceID string) ([]*kafka.ObservabilityEvent, error) {
	pattern := fmt.Sprintf("traces:%s:*", traceID)
	keys, err := s.redisClient.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	var events []*kafka.ObservabilityEvent
	for _, key := range keys {
		var event kafka.ObservabilityEvent
		if err := s.redisClient.Get(ctx, key, &event); err == nil {
			events = append(events, &event)
		}
	}

	return events, nil
}

// Close closes the event streaming service
func (s *EventStreamingService) Close() error {
	return s.producer.Close()
}
