package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/saintparish4/realtime-data-pipeline/internal/config"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/observability"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

func main() {
	// Load configuration
	cfg, err := config.Load("../../cmd/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)
	defer redisClient.Close()

	// Initialize event streaming service
	eventService, err := observability.NewEventStreamingService(cfg, redisClient)
	if err != nil {
		log.Fatalf("Failed to create event streaming service: %v", err)
	}
	defer eventService.Close()

	// Create context
	ctx := context.Background()

	fmt.Println("=== Observability Event Streaming Example ===")
	fmt.Println("This example demonstrates the new Kafka infrastructure for event streaming")
	fmt.Println("with distributed tracing and correlation tracking.")
	fmt.Println()

	// Example 1: Distributed Tracing
	fmt.Println("=== 1. Distributed Tracing Example ===")

	// Start a trace for a data processing operation
	traceID, spanID := eventService.StartTrace(ctx, "data_processing", map[string]string{
		"symbol": "BTCUSDT",
		"source": "binance",
	})

	fmt.Printf("Started trace: %s, span: %s\n", traceID, spanID)

	// Simulate some processing work
	time.Sleep(100 * time.Millisecond)

	// Publish some events with correlation
	correlationID := uuid.New().String()

	// Publish log event
	err = eventService.PublishLogEvent(ctx, kafka.EventLevelInfo, "processor",
		"Processing BTCUSDT data", correlationID, map[string]string{
			"symbol": "BTCUSDT",
			"price":  "45000",
		})
	if err != nil {
		log.Printf("Failed to publish log event: %v", err)
	}

	// Publish metric event
	err = eventService.PublishMetricEvent(ctx, "processor", "processing_latency",
		150.5, correlationID, map[string]string{
			"symbol": "BTCUSDT",
		})
	if err != nil {
		log.Printf("Failed to publish metric event: %v", err)
	}

	// End the trace
	err = eventService.EndTrace(ctx, traceID, spanID, map[string]string{
		"status":            "completed",
		"records_processed": "1000",
	})
	if err != nil {
		log.Printf("Failed to end trace: %v", err)
	}

	fmt.Println("Completed distributed trace example")
	fmt.Println()

	// Example 2: Alert Event Publishing
	fmt.Println("=== 2. Alert Event Publishing ===")

	// Create an alert
	alert := &models.Alert{
		ID:           "high_price_alert_001",
		Name:         "BTC High Price Alert",
		Description:  "BTC price exceeded $50,000",
		Severity:     models.AlertSeverityWarning,
		Status:       models.AlertStatusActive,
		Symbol:       "BTCUSDT",
		Condition:    "price > threshold",
		Threshold:    50000,
		CurrentValue: 52000,
		Labels: map[string]string{
			"exchange": "binance",
			"pair":     "BTCUSDT",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Trigger the alert
	alert.Trigger(52000)

	// Publish alert event
	err = eventService.PublishAlertEvent(ctx, alert, correlationID)
	if err != nil {
		log.Printf("Failed to publish alert event: %v", err)
	} else {
		fmt.Printf("Published alert event: %s - %s\n", alert.Name, alert.Description)
	}

	fmt.Println()

	// Example 3: Health Check Events
	fmt.Println("=== 3. Health Check Events ===")

	// Publish health check for different components
	components := []string{"redis", "kafka", "websocket", "processor"}

	for _, component := range components {
		status := "healthy"
		details := map[string]interface{}{
			"response_time_ms": 50 + time.Now().UnixNano()%100,
			"connections":      10 + time.Now().UnixNano()%20,
		}

		// Simulate some components as degraded
		if component == "websocket" {
			status = "degraded"
			details["error"] = "High latency detected"
		}

		err = eventService.PublishHealthEvent(ctx, component, status, details)
		if err != nil {
			log.Printf("Failed to publish health event for %s: %v", component, err)
		} else {
			fmt.Printf("Published health check: %s is %s\n", component, status)
		}
	}

	fmt.Println()

	// Example 4: Error Event Publishing
	fmt.Println("=== 4. Error Event Publishing ===")

	// Publish different types of errors
	errors := []struct {
		component     string
		errorType     string
		message       string
		correlationID string
	}{
		{
			component:     "websocket",
			errorType:     "connection_timeout",
			message:       "WebSocket connection timeout after 30 seconds",
			correlationID: correlationID,
		},
		{
			component:     "processor",
			errorType:     "validation_error",
			message:       "Invalid data format received",
			correlationID: correlationID,
		},
		{
			component:     "redis",
			errorType:     "memory_limit",
			message:       "Redis memory usage exceeded 80%",
			correlationID: "",
		},
	}

	for _, errInfo := range errors {
		err = eventService.PublishErrorEvent(ctx, errInfo.component, errInfo.errorType,
			errInfo.message, errInfo.correlationID, map[string]string{
				"component": errInfo.component,
			})
		if err != nil {
			log.Printf("Failed to publish error event: %v", err)
		} else {
			fmt.Printf("Published error: %s - %s\n", errInfo.errorType, errInfo.message)
		}
	}

	fmt.Println()

	// Example 5: Event Correlation Analysis
	fmt.Println("=== 5. Event Correlation Analysis ===")

	// Wait a moment for events to be processed
	time.Sleep(2 * time.Second)

	// Retrieve correlated events
	correlatedEvents, err := eventService.GetCorrelatedEvents(ctx, correlationID)
	if err != nil {
		log.Printf("Failed to get correlated events: %v", err)
	} else {
		fmt.Printf("Found %d events with correlation ID: %s\n", len(correlatedEvents), correlationID)
		for _, event := range correlatedEvents {
			fmt.Printf("  - %s: %s (%s)\n", event.Type, event.Message, event.Component)
		}
	}

	// Retrieve trace events
	traceEvents, err := eventService.GetTraceEvents(ctx, traceID)
	if err != nil {
		log.Printf("Failed to get trace events: %v", err)
	} else {
		fmt.Printf("Found %d events for trace: %s\n", len(traceEvents), traceID)
		for _, event := range traceEvents {
			fmt.Printf("  - %s: %s (span: %s)\n", event.Type, event.Message, event.SpanID)
		}
	}

	fmt.Println()

	// Example 6: Kafka Producer Direct Usage
	fmt.Println("=== 6. Direct Kafka Producer Usage ===")

	// Create a producer for specific topics
	logProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Logs, "example-service")
	metricProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Metrics, "example-service")
	alertProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Alerts, "example-service")

	defer logProducer.Close()
	defer metricProducer.Close()
	defer alertProducer.Close()

	// Publish events directly to specific topics
	err = logProducer.PublishLog(ctx, kafka.EventLevelInfo, "direct_producer",
		"Direct log message", map[string]string{"source": "direct"}, nil)
	if err != nil {
		log.Printf("Failed to publish direct log: %v", err)
	} else {
		fmt.Println("Published direct log message")
	}

	err = metricProducer.PublishMetric(ctx, "direct_producer", "direct_metric",
		42.0, map[string]string{"source": "direct"}, nil)
	if err != nil {
		log.Printf("Failed to publish direct metric: %v", err)
	} else {
		fmt.Println("Published direct metric")
	}

	err = alertProducer.PublishAlert(ctx, "direct_alert_001", "Direct Alert",
		"This is a direct alert", kafka.EventLevelWarning,
		map[string]string{"source": "direct"}, nil)
	if err != nil {
		log.Printf("Failed to publish direct alert: %v", err)
	} else {
		fmt.Println("Published direct alert")
	}

	fmt.Println()

	// Example 7: Kafka Consumer Usage
	fmt.Println("=== 7. Kafka Consumer Usage ===")

	// Start consumers in goroutines to demonstrate filtering
	go func() {
		consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Logs, "log-consumer")
		defer consumer.Close()

		fmt.Println("Starting log consumer...")
		err := consumer.ConsumeLogs(ctx, func(ctx context.Context, event *kafka.ObservabilityEvent) error {
			fmt.Printf("  [LOG] %s: %s\n", event.Level, event.Message)
			return nil
		})
		if err != nil {
			log.Printf("Log consumer error: %v", err)
		}
	}()

	go func() {
		consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Metrics, "metric-consumer")
		defer consumer.Close()

		fmt.Println("Starting metric consumer...")
		err := consumer.ConsumeMetrics(ctx, func(ctx context.Context, event *kafka.ObservabilityEvent) error {
			fmt.Printf("  [METRIC] %s: %s\n", event.Component, event.Message)
			return nil
		})
		if err != nil {
			log.Printf("Metric consumer error: %v", err)
		}
	}()

	go func() {
		consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Alerts, "alert-consumer")
		defer consumer.Close()

		fmt.Println("Starting alert consumer...")
		err := consumer.ConsumeAlerts(ctx, func(ctx context.Context, event *kafka.ObservabilityEvent) error {
			fmt.Printf("  [ALERT] %s: %s\n", event.Level, event.Message)
			return nil
		})
		if err != nil {
			log.Printf("Alert consumer error: %v", err)
		}
	}()

	// Wait a moment for consumers to process events
	time.Sleep(3 * time.Second)

	fmt.Println()
	fmt.Println("=== Event Streaming Example Complete ===")
	fmt.Println()
	fmt.Println("Key Features Demonstrated:")
	fmt.Println("1. Distributed tracing with trace and span IDs")
	fmt.Println("2. Event correlation using correlation IDs")
	fmt.Println("3. Structured event publishing (logs, metrics, alerts, traces, health)")
	fmt.Println("4. Event filtering and routing by type")
	fmt.Println("5. Direct Kafka producer/consumer usage")
	fmt.Println("6. Event storage and retrieval from Redis")
	fmt.Println("7. Health monitoring and error tracking")
	fmt.Println()
	fmt.Println("The new Kafka infrastructure now focuses on:")
	fmt.Println("- Log/metric ingestion instead of raw data")
	fmt.Println("- Alert events instead of processed data")
	fmt.Println("- Distributed tracing and correlation")
	fmt.Println("- Structured observability events")
	fmt.Println("- Event filtering and routing")
}
