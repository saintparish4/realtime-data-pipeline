package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/processing"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

func main() {
	fmt.Println("🚀 Enhanced Metrics Processing Engine Example")
	fmt.Println("=============================================")

	// Initialize Redis client
	redisClient := redis.NewClient("localhost:6379", "", 0)

	// Initialize Kafka producer and consumer
	producer := kafka.NewProducer([]string{"localhost:9092"}, "crypto.processed", "metrics-engine")
	consumer := kafka.NewConsumer([]string{"localhost:9092"}, "crypto.raw", "metrics-processor")

	// Create enhanced metrics engine configuration
	config := &processing.MetricsEngineConfig{
		MovingAverages: []models.MovingAverageConfig{
			{Window: 5 * time.Minute, Method: "simple"},
			{Window: 15 * time.Minute, Method: "simple"},
			{Window: 1 * time.Hour, Method: "exponential", Alpha: 0.1},
		},
		AnomalyDetection: models.AnomalyDetectionConfig{
			Method:        "zscore",
			Window:        1 * time.Hour,
			Threshold:     2.5,
			MinDataPoints: 10,
		},
		RollupWindows: []time.Duration{
			1 * time.Minute,
			5 * time.Minute,
			15 * time.Minute,
			1 * time.Hour,
		},
		AlertThresholds: []models.AlertThreshold{
			{
				ID:         "price_high",
				Name:       "High Price Alert",
				Symbol:     "BTC",
				MetricType: models.MetricTypePrice,
				Condition:  "gt",
				Threshold:  50000.0,
				Window:     5 * time.Minute,
				Severity:   models.AlertSeverityWarning,
				Enabled:    true,
			},
			{
				ID:         "volume_spike",
				Name:       "Volume Spike Alert",
				Symbol:     "BTC",
				MetricType: models.MetricTypeVolume,
				Condition:  "gt",
				Threshold:  1000000.0,
				Window:     5 * time.Minute,
				Severity:   models.AlertSeverityWarning,
				Enabled:    true,
			},
		},
		CacheTTL: 30 * time.Second,
	}

	// Create metrics engine (not used in this demonstration)
	_ = processing.NewMetricsEngine(consumer, producer, redisClient, nil, config)

	// Simulate data ingestion and processing
	fmt.Println("\n📊 Simulating cryptocurrency data processing...")
	simulateDataProcessing(redisClient)

	// Demonstrate configuration
	fmt.Println("\n⚙️  Demonstrating enhanced configuration...")
	demonstrateConfiguration(config)

	// Demonstrate data models
	fmt.Println("\n📋 Demonstrating enhanced data models...")
	demonstrateDataModels()

	fmt.Println("\n✅ Enhanced Metrics Processing Engine example completed!")
	fmt.Println("\n💡 This example demonstrates the enhanced capabilities:")
	fmt.Println("   - Anomaly detection with multiple algorithms")
	fmt.Println("   - Metric rollups with statistical calculations")
	fmt.Println("   - Configurable alert thresholds")
	fmt.Println("   - Advanced moving average methods")
}

func simulateDataProcessing(redisClient *redis.Client) {
	// Simulate historical data for anomaly detection
	symbols := []string{"BTC", "ETH", "ADA"}
	basePrices := map[string]float64{"BTC": 45000, "ETH": 3000, "ADA": 1.5}
	baseVolumes := map[string]float64{"BTC": 500000, "ETH": 200000, "ADA": 50000}

	// Generate historical data
	for _, symbol := range symbols {
		for i := 0; i < 20; i++ {
			// Generate realistic price data with some noise
			price := basePrices[symbol] + rand.Float64()*1000 - 500
			volume := baseVolumes[symbol] + rand.Float64()*100000 - 50000

			// Store historical data in Redis for anomaly detection
			priceMetric := &models.TimeSeriesMetric{
				Symbol:    symbol,
				Type:      models.MetricTypePrice,
				Value:     price,
				Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			}
			if err := redisClient.StoreTimeSeriesMetric(context.Background(), priceMetric); err != nil {
				log.Printf("Error storing price metric: %v", err)
			}

			volumeMetric := &models.TimeSeriesMetric{
				Symbol:    symbol,
				Type:      models.MetricTypeVolume,
				Value:     volume,
				Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			}
			if err := redisClient.StoreTimeSeriesMetric(context.Background(), volumeMetric); err != nil {
				log.Printf("Error storing volume metric: %v", err)
			}
		}
	}

	fmt.Println("   ✅ Generated historical data for anomaly detection")
}

func demonstrateConfiguration(config *processing.MetricsEngineConfig) {
	fmt.Println("   📊 Moving Average Configurations:")
	for i, ma := range config.MovingAverages {
		fmt.Printf("      %d. %s window, %s method", i+1, ma.Window, ma.Method)
		if ma.Method == "exponential" {
			fmt.Printf(" (alpha: %.2f)", ma.Alpha)
		}
		fmt.Println()
	}

	fmt.Println("   🔍 Anomaly Detection Configuration:")
	fmt.Printf("      Method: %s\n", config.AnomalyDetection.Method)
	fmt.Printf("      Window: %s\n", config.AnomalyDetection.Window)
	fmt.Printf("      Threshold: %.2f\n", config.AnomalyDetection.Threshold)
	fmt.Printf("      Min Data Points: %d\n", config.AnomalyDetection.MinDataPoints)

	fmt.Println("   📈 Rollup Windows:")
	for _, window := range config.RollupWindows {
		fmt.Printf("      - %s\n", window)
	}

	fmt.Println("   🚨 Alert Thresholds:")
	for _, threshold := range config.AlertThresholds {
		fmt.Printf("      - %s: %s %s %.2f\n",
			threshold.Name, threshold.Symbol, threshold.Condition, threshold.Threshold)
	}
}

func demonstrateDataModels() {
	// Demonstrate anomaly detection result
	anomaly := &models.AnomalyDetectionResult{
		Symbol:        "BTC",
		Type:          models.AnomalyTypePriceSpike,
		Severity:      models.AnomalySeverityHigh,
		CurrentValue:  60000.0,
		ExpectedValue: 45000.0,
		Deviation:     15000.0,
		Confidence:    0.85,
		Timestamp:     time.Now(),
		Window:        1 * time.Hour,
	}

	fmt.Println("   🔍 Anomaly Detection Result:")
	fmt.Printf("      Symbol: %s\n", anomaly.Symbol)
	fmt.Printf("      Type: %s\n", anomaly.Type)
	fmt.Printf("      Severity: %s\n", anomaly.Severity)
	fmt.Printf("      Current Value: %.2f\n", anomaly.CurrentValue)
	fmt.Printf("      Expected Value: %.2f\n", anomaly.ExpectedValue)
	fmt.Printf("      Deviation: %.2f\n", anomaly.Deviation)
	fmt.Printf("      Confidence: %.2f\n", anomaly.Confidence)
	fmt.Printf("      Is Significant: %t\n", anomaly.IsSignificant())

	// Demonstrate metric rollup
	rollup := &models.MetricRollup{
		Symbol:       "ETH",
		Type:         models.MetricTypePrice,
		Window:       5 * time.Minute,
		StartTime:    time.Now().Add(-5 * time.Minute),
		EndTime:      time.Now(),
		Count:        100,
		Sum:          320000.0,
		Min:          3150.0,
		Max:          3250.0,
		Average:      3200.0,
		Median:       3205.0,
		StdDev:       25.0,
		Percentile95: 3240.0,
		Percentile99: 3250.0,
	}

	fmt.Println("   📈 Metric Rollup:")
	fmt.Printf("      Symbol: %s\n", rollup.Symbol)
	fmt.Printf("      Window: %s\n", rollup.Window)
	fmt.Printf("      Count: %d\n", rollup.Count)
	fmt.Printf("      Average: %.2f\n", rollup.Average)
	fmt.Printf("      Range: %.2f\n", rollup.GetRange())
	fmt.Printf("      Coefficient of Variation: %.4f\n", rollup.GetCoefficientOfVariation())
}
