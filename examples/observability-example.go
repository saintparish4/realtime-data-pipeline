package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/observability"
	"github.com/saintparish4/realtime-data-pipeline/internal/storage"
	"github.com/saintparish4/realtime-data-pipeline/pkg/database"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

func main() {
	fmt.Println("=== Enhanced Observability Example with Database Layer ===")
	fmt.Println("This example demonstrates the new database layer with TimescaleDB and PostgreSQL")

	// Initialize Redis client
	redisClient := redis.NewClient("localhost:6379", "", 0)
	defer redisClient.Close()

	// Initialize TimescaleDB for metrics storage
	timescaleDB, err := database.NewTimescaleDB("localhost", "5433", "postgres", "postgres", "timeseries")
	if err != nil {
		log.Printf("Warning: Failed to connect to TimescaleDB: %v", err)
		log.Println("Continuing with Redis-only mode...")
		timescaleDB = nil
	} else {
		defer timescaleDB.Close()
	}

	// Initialize PostgreSQL for alert storage
	postgresDB, err := database.NewPostgreSQL("localhost", "5432", "postgres", "postgres", "pipeline")
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL: %v", err)
		log.Println("Continuing with Redis-only mode...")
		postgresDB = nil
	} else {
		defer postgresDB.Close()
	}

	// Create storage manager if both databases are available
	var storageManager *storage.StorageManager
	if timescaleDB != nil && postgresDB != nil {
		storageManager = storage.NewStorageManager(timescaleDB, postgresDB)
		defer storageManager.Close()

		// Initialize storage
		ctx := context.Background()
		if err := storageManager.InitializeStorage(ctx); err != nil {
			log.Printf("Warning: Failed to initialize storage: %v", err)
		} else {
			fmt.Println("✓ Database layer initialized successfully")
		}
	}

	// Initialize observability service
	var obsService *observability.Service
	if storageManager != nil {
		obsService = observability.NewService(redisClient, storageManager)
		if err := obsService.Initialize(context.Background()); err != nil {
			log.Printf("Warning: Failed to initialize observability service: %v", err)
		}
	} else {
		// Fallback to Redis-only mode (old behavior)
		obsService = observability.NewService(redisClient, nil)
	}

	// Create context
	ctx := context.Background()

	// Example 1: Record metrics
	fmt.Println("=== Recording Metrics ===")

	// Record price metrics
	for i := 0; i < 5; i++ {
		priceMetric := &models.TimeSeriesMetric{
			Symbol:    "BTCUSDT",
			Type:      models.MetricTypePrice,
			Value:     45000 + float64(i*1000), // Simulate price changes
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Labels: map[string]string{
				"exchange": "binance",
				"pair":     "BTCUSDT",
			},
		}

		if err := obsService.RecordMetric(ctx, priceMetric); err != nil {
			log.Printf("Failed to record price metric: %v", err)
		} else {
			fmt.Printf("Recorded price metric: %.2f at %s\n", priceMetric.Value, priceMetric.Timestamp.Format("15:04:05"))
		}
	}

	// Record volume metrics
	for i := 0; i < 5; i++ {
		volumeMetric := &models.TimeSeriesMetric{
			Symbol:    "BTCUSDT",
			Type:      models.MetricTypeVolume,
			Value:     1000000 + float64(i*100000), // Simulate volume changes
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Labels: map[string]string{
				"exchange": "binance",
				"pair":     "BTCUSDT",
			},
		}

		if err := obsService.RecordMetric(ctx, volumeMetric); err != nil {
			log.Printf("Failed to record volume metric: %v", err)
		} else {
			fmt.Printf("Recorded volume metric: %.2f at %s\n", volumeMetric.Value, volumeMetric.Timestamp.Format("15:04:05"))
		}
	}

	// Example 2: Create alert rules
	fmt.Println("\n=== Creating Alert Rules ===")

	// High price alert
	highPriceRule := &observability.AlertRule{
		ID:          "high_price_btc",
		Name:        "BTC High Price Alert",
		Description: "Alert when BTC price exceeds $50,000",
		Severity:    models.AlertSeverityWarning,
		Symbol:      "BTCUSDT",
		Condition:   "price > threshold",
		Threshold:   50000,
		Enabled:     true,
	}

	if err := obsService.AddAlertRule(highPriceRule); err != nil {
		log.Printf("Failed to add high price rule: %v", err)
	} else {
		fmt.Println("Added high price alert rule")
	}

	// Low volume alert
	lowVolumeRule := &observability.AlertRule{
		ID:          "low_volume_btc",
		Name:        "BTC Low Volume Alert",
		Description: "Alert when BTC volume drops below $500,000",
		Severity:    models.AlertSeverityInfo,
		Symbol:      "BTCUSDT",
		Condition:   "volume < threshold",
		Threshold:   500000,
		Enabled:     true,
	}

	if err := obsService.AddAlertRule(lowVolumeRule); err != nil {
		log.Printf("Failed to add low volume rule: %v", err)
	} else {
		fmt.Println("Added low volume alert rule")
	}

	// Example 3: Trigger alerts by recording metrics that exceed thresholds
	fmt.Println("\n=== Triggering Alerts ===")

	// Record a high price that should trigger an alert
	highPriceMetric := &models.TimeSeriesMetric{
		Symbol:    "BTCUSDT",
		Type:      models.MetricTypePrice,
		Value:     52000, // Above the 50,000 threshold
		Timestamp: time.Now(),
		Labels: map[string]string{
			"exchange": "binance",
			"pair":     "BTCUSDT",
		},
	}

	if err := obsService.RecordMetric(ctx, highPriceMetric); err != nil {
		log.Printf("Failed to record high price metric: %v", err)
	} else {
		fmt.Println("Recorded high price metric that should trigger alert")
	}

	// Record a low volume that should trigger an alert
	lowVolumeMetric := &models.TimeSeriesMetric{
		Symbol:    "BTCUSDT",
		Type:      models.MetricTypeVolume,
		Value:     300000, // Below the 500,000 threshold
		Timestamp: time.Now(),
		Labels: map[string]string{
			"exchange": "binance",
			"pair":     "BTCUSDT",
		},
	}

	if err := obsService.RecordMetric(ctx, lowVolumeMetric); err != nil {
		log.Printf("Failed to record low volume metric: %v", err)
	} else {
		fmt.Println("Recorded low volume metric that should trigger alert")
	}

	// Wait a moment for alert processing
	time.Sleep(2 * time.Second)

	// Example 4: Query metrics
	fmt.Println("\n=== Querying Metrics ===")

	// Query price metrics for the last hour
	priceQuery := &models.TimeSeriesQuery{
		Symbol:    "BTCUSDT",
		Type:      models.MetricTypePrice,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Limit:     10,
	}

	priceMetrics, err := obsService.GetMetrics(ctx, priceQuery)
	if err != nil {
		log.Printf("Failed to query price metrics: %v", err)
	} else {
		fmt.Printf("Found %d price metrics in the last hour\n", len(priceMetrics))
		for _, metric := range priceMetrics {
			fmt.Printf("  Price: %.2f at %s\n", metric.Value, metric.Timestamp.Format("15:04:05"))
		}
	}

	// Example 5: Check active alerts
	fmt.Println("\n=== Checking Active Alerts ===")

	activeAlerts, err := obsService.GetActiveAlerts(ctx)
	if err != nil {
		log.Printf("Failed to get active alerts: %v", err)
	} else {
		fmt.Printf("Found %d active alerts\n", len(activeAlerts))
		for _, alert := range activeAlerts {
			fmt.Printf("  Alert: %s - %s (Value: %.2f, Threshold: %.2f)\n",
				alert.Name, alert.Description, alert.CurrentValue, alert.Threshold)
		}
	}

	// Example 6: Get metric statistics
	fmt.Println("\n=== Metric Statistics ===")

	stats, err := obsService.GetMetricStats(ctx, "BTCUSDT", models.MetricTypePrice)
	if err != nil {
		log.Printf("Failed to get metric stats: %v", err)
	} else {
		fmt.Printf("Price metric statistics:\n")
		for key, value := range stats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Example 7: System health check
	fmt.Println("\n=== System Health Check ===")

	health, err := obsService.GetSystemHealth(ctx)
	if err != nil {
		log.Printf("Failed to get system health: %v", err)
	} else {
		fmt.Printf("System status: %s\n", health["status"])
		if checks, ok := health["checks"].(map[string]interface{}); ok {
			for checkName, checkData := range checks {
				if checkMap, ok := checkData.(map[string]interface{}); ok {
					fmt.Printf("  %s: %s\n", checkName, checkMap["status"])
				}
			}
		}
	}

	// Example 8: Record performance metrics
	fmt.Println("\n=== Recording Performance Metrics ===")

	// Simulate some operations and record their latency
	operations := []string{"data_processing", "redis_query", "kafka_publish"}

	for _, operation := range operations {
		// Simulate operation duration
		duration := time.Duration(10+time.Now().UnixNano()%100) * time.Millisecond
		time.Sleep(duration)

		if err := obsService.RecordLatencyMetric(ctx, operation, duration, map[string]string{
			"component": "processing_engine",
		}); err != nil {
			log.Printf("Failed to record latency metric: %v", err)
		} else {
			fmt.Printf("Recorded latency for %s: %v\n", operation, duration)
		}
	}

	// Example 9: Record error metrics
	fmt.Println("\n=== Recording Error Metrics ===")

	// Simulate some errors
	errors := []struct {
		errorType    string
		errorMessage string
	}{
		{"connection_timeout", "Redis connection timeout"},
		{"validation_error", "Invalid data format"},
		{"processing_error", "Failed to calculate moving average"},
	}

	for _, errInfo := range errors {
		if err := obsService.RecordErrorMetric(ctx, errInfo.errorType, errInfo.errorMessage, map[string]string{
			"component": "example",
		}); err != nil {
			log.Printf("Failed to record error metric: %v", err)
		} else {
			fmt.Printf("Recorded error: %s - %s\n", errInfo.errorType, errInfo.errorMessage)
		}
	}

	fmt.Println("\n=== Observability Example Complete ===")
	fmt.Println("Check the logs above for any errors or alerts that were triggered.")
	fmt.Println("You can also use Redis CLI to inspect the stored data:")
	fmt.Println("  redis-cli ZRANGE metrics:price:BTCUSDT 0 -1")
	fmt.Println("  redis-cli SMEMBERS alerts:active")
	fmt.Println("  redis-cli KEYS rate_limit:*")
}
