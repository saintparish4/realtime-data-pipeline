package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Consumer handles consuming observability events from Kafka
type Consumer struct {
	reader *kafka.Reader
}

// EventHandler processes observability events
type EventHandler func(ctx context.Context, event *ObservabilityEvent) error

// EventFilter defines filtering criteria for events
type EventFilter struct {
	Types      []EventType  `json:"types,omitempty"`
	Levels     []EventLevel `json:"levels,omitempty"`
	Services   []string     `json:"services,omitempty"`
	Components []string     `json:"components,omitempty"`
}

// NewConsumer creates a new Kafka consumer for observability events
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})

	return &Consumer{reader: reader}
}

// ConsumeEvents consumes observability events and processes them with the provided handler
func (c *Consumer) ConsumeEvents(ctx context.Context, handler EventHandler) error {
	return c.ConsumeEventsWithFilter(ctx, handler, nil)
}

// ConsumeEventsWithFilter consumes observability events with optional filtering
func (c *Consumer) ConsumeEventsWithFilter(ctx context.Context, handler EventHandler, filter *EventFilter) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Parse the event
			var event ObservabilityEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling event: %v", err)
				continue
			}

			// Apply filter if provided
			if filter != nil && !c.matchesFilter(&event, filter) {
				continue
			}

			// Process the event
			if err := handler(ctx, &event); err != nil {
				log.Printf("Error handling event: %v", err)
			}
		}
	}
}

// ConsumeLogs consumes only log events
func (c *Consumer) ConsumeLogs(ctx context.Context, handler EventHandler) error {
	filter := &EventFilter{Types: []EventType{EventTypeLog}}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeMetrics consumes only metric events
func (c *Consumer) ConsumeMetrics(ctx context.Context, handler EventHandler) error {
	filter := &EventFilter{Types: []EventType{EventTypeMetric}}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeAlerts consumes only alert events
func (c *Consumer) ConsumeAlerts(ctx context.Context, handler EventHandler) error {
	filter := &EventFilter{Types: []EventType{EventTypeAlert}}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeTraces consumes only trace events
func (c *Consumer) ConsumeTraces(ctx context.Context, handler EventHandler) error {
	filter := &EventFilter{Types: []EventType{EventTypeTrace}}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeHealthEvents consumes only health check events
func (c *Consumer) ConsumeHealthEvents(ctx context.Context, handler EventHandler) error {
	filter := &EventFilter{Types: []EventType{EventTypeHealth}}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeErrors consumes only error-level events
func (c *Consumer) ConsumeErrors(ctx context.Context, handler EventHandler) error {
	filter := &EventFilter{Levels: []EventLevel{EventLevelError, EventLevelFatal}}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeByService consumes events from specific services
func (c *Consumer) ConsumeByService(ctx context.Context, services []string, handler EventHandler) error {
	filter := &EventFilter{Services: services}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeByComponent consumes events from specific components
func (c *Consumer) ConsumeByComponent(ctx context.Context, components []string, handler EventHandler) error {
	filter := &EventFilter{Components: components}
	return c.ConsumeEventsWithFilter(ctx, handler, filter)
}

// ConsumeByCorrelationID consumes events with a specific correlation ID
func (c *Consumer) ConsumeByCorrelationID(ctx context.Context, correlationID string, handler EventHandler) error {
	return c.ConsumeEvents(ctx, func(ctx context.Context, event *ObservabilityEvent) error {
		if event.CorrelationID == correlationID {
			return handler(ctx, event)
		}
		return nil
	})
}

// matchesFilter checks if an event matches the given filter criteria
func (c *Consumer) matchesFilter(event *ObservabilityEvent, filter *EventFilter) bool {
	// Check event types
	if len(filter.Types) > 0 {
		matched := false
		for _, eventType := range filter.Types {
			if event.Type == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check event levels
	if len(filter.Levels) > 0 {
		matched := false
		for _, level := range filter.Levels {
			if event.Level == level {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check services
	if len(filter.Services) > 0 {
		matched := false
		for _, service := range filter.Services {
			if event.Service == service {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check components
	if len(filter.Components) > 0 {
		matched := false
		for _, component := range filter.Components {
			if event.Component == component {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// GetEventStats returns statistics about consumed events
func (c *Consumer) GetEventStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"event_types":  make(map[string]int),
		"event_levels": make(map[string]int),
		"services":     make(map[string]int),
		"components":   make(map[string]int),
		"total_events": 0,
	}

	// This is a simplified implementation
	// In a real scenario, you might want to use a more sophisticated approach
	// like maintaining counters in Redis or using Kafka Streams

	return stats, nil
}

// Close closes the Kafka consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// Legacy support - maintain backward compatibility
func (c *Consumer) Consume(ctx context.Context, handler func(ctx context.Context, key, value []byte) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			if err := handler(ctx, msg.Key, msg.Value); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}
