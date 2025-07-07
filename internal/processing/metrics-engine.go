package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/observability"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// MetricsEngine processes raw cryptocurrency data by calculating moving averages,
// detecting anomalies, performing metric rollups, and evaluating alert thresholds.
// It handles data ingestion from Kafka, processing, storage in Redis, and publishing processed results.
type MetricsEngine struct {
	consumer        *kafka.Consumer
	producer        *kafka.Producer
	redisClient     *redis.Client
	windowProcessor *WindowProcessor
	obsService      *observability.Service

	// Enhanced processing components
	anomalyDetector *AnomalyDetector
	metricRollup    *MetricRollupProcessor
	alertManager    *AlertManager

	// Performance optimizations
	cache    map[string]cacheEntry
	cacheMux sync.RWMutex
	cacheTTL time.Duration

	// Configuration
	config *MetricsEngineConfig
}

// MetricsEngineConfig holds configuration for the metrics engine
type MetricsEngineConfig struct {
	// Moving average configurations
	MovingAverages []models.MovingAverageConfig `json:"moving_averages"`

	// Anomaly detection configurations
	AnomalyDetection models.AnomalyDetectionConfig `json:"anomaly_detection"`

	// Metric rollup configurations
	RollupWindows []time.Duration `json:"rollup_windows"`

	// Alert threshold configurations
	AlertThresholds []models.AlertThreshold `json:"alert_thresholds"`

	// Performance settings
	CacheTTL time.Duration `json:"cache_ttl"`
}

// NewMetricsEngineConfig creates a default configuration
func NewMetricsEngineConfig() *MetricsEngineConfig {
	return &MetricsEngineConfig{
		MovingAverages: []models.MovingAverageConfig{
			{Window: 5 * time.Minute, Method: "simple"},
			{Window: 15 * time.Minute, Method: "simple"},
			{Window: 1 * time.Hour, Method: "simple"},
			{Window: 4 * time.Hour, Method: "exponential", Alpha: 0.1},
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
			4 * time.Hour,
			1 * 24 * time.Hour,
		},
		CacheTTL: 30 * time.Second,
	}
}

// NewMetricsEngine creates a new metrics processing engine with the given dependencies.
func NewMetricsEngine(consumer *kafka.Consumer, producer *kafka.Producer, redisClient *redis.Client, obsService *observability.Service, config *MetricsEngineConfig) *MetricsEngine {
	if config == nil {
		config = NewMetricsEngineConfig()
	}

	engine := &MetricsEngine{
		consumer:        consumer,
		producer:        producer,
		redisClient:     redisClient,
		windowProcessor: NewWindowProcessor(redisClient),
		obsService:      obsService,
		cache:           make(map[string]cacheEntry),
		cacheTTL:        config.CacheTTL,
		config:          config,
	}

	// Initialize enhanced processing components
	engine.anomalyDetector = NewAnomalyDetector(redisClient, config.AnomalyDetection)
	engine.metricRollup = NewMetricRollupProcessor(redisClient, config.RollupWindows)
	engine.alertManager = NewAlertManager(redisClient, config.AlertThresholds)

	return engine
}

// Start begins consuming messages from Kafka and processing them.
// It blocks until the context is cancelled or an error occurs.
func (e *MetricsEngine) Start(ctx context.Context) error {
	return e.consumer.Consume(ctx, e.handleMessage)
}

// handleMessage processes a single message from Kafka with enhanced metrics processing.
func (e *MetricsEngine) handleMessage(ctx context.Context, key, value []byte) error {
	startTime := time.Now()

	var rawData models.RawCryptoData
	if err := json.Unmarshal(value, &rawData); err != nil {
		e.recordErrorMetric(ctx, "unmarshal_error", err.Error(), map[string]string{
			"symbol": string(key),
		})
		return fmt.Errorf("failed to unmarshal raw data: %w", err)
	}

	// Validate the raw data
	if err := rawData.Validate(); err != nil {
		e.recordErrorMetric(ctx, "validation_error", err.Error(), map[string]string{
			"symbol": rawData.Symbol,
		})
		return fmt.Errorf("invalid raw data: %w", err)
	}

	// Record base metrics
	e.recordBaseMetrics(ctx, &rawData)

	// Process the data with enhanced metrics
	processedData, err := e.processData(ctx, &rawData)
	if err != nil {
		e.recordErrorMetric(ctx, "processing_error", err.Error(), map[string]string{
			"symbol": rawData.Symbol,
		})
		return fmt.Errorf("failed to process data: %w", err)
	}

	// Detect anomalies
	anomalies, err := e.detectAnomalies(ctx, &rawData, processedData)
	if err != nil {
		log.Printf("Failed to detect anomalies for %s: %v", rawData.Symbol, err)
		// Continue processing even if anomaly detection fails
	}

	// Perform metric rollups
	if err := e.performMetricRollups(ctx, &rawData); err != nil {
		log.Printf("Failed to perform metric rollups for %s: %v", rawData.Symbol, err)
		// Continue processing even if rollups fail
	}

	// Evaluate alert thresholds
	if err := e.evaluateAlertThresholds(ctx, processedData, anomalies); err != nil {
		log.Printf("Failed to evaluate alert thresholds for %s: %v", rawData.Symbol, err)
		// Continue processing even if alert evaluation fails
	}

	// Store in Redis for real-time queries
	if err := e.storeInRedis(ctx, processedData); err != nil {
		log.Printf("Failed to store data in Redis: %v", err)
		e.recordErrorMetric(ctx, "redis_storage_error", err.Error(), map[string]string{
			"symbol": processedData.Symbol,
		})
		// Continue processing even if Redis storage fails
	}

	// Publish processed data
	if err := e.producer.Publish(ctx, processedData.Symbol, processedData); err != nil {
		e.recordErrorMetric(ctx, "kafka_publish_error", err.Error(), map[string]string{
			"symbol": processedData.Symbol,
		})
		return fmt.Errorf("failed to publish processed data: %w", err)
	}

	// Record processing latency
	e.recordLatencyMetric(ctx, "message_processing", time.Since(startTime), map[string]string{
		"symbol": rawData.Symbol,
	})

	return nil
}

// processData calculates moving averages and 24-hour changes for the given raw data.
// Enhanced with configurable moving average methods.
func (e *MetricsEngine) processData(ctx context.Context, rawData *models.RawCryptoData) (*models.ProcessedCryptoData, error) {
	processedData := &models.ProcessedCryptoData{
		Symbol:      rawData.Symbol,
		Price:       rawData.Price,
		Volume:      rawData.Volume,
		Timestamp:   rawData.Timestamp,
		ProcessedAt: time.Now(),
	}

	// Calculate moving averages using configured methods
	for _, maConfig := range e.config.MovingAverages {
		ma, err := e.calculateMovingAverage(ctx, rawData.Symbol, maConfig)
		if err != nil {
			log.Printf("Failed to calculate moving average for %s with config %+v: %v", rawData.Symbol, maConfig, err)
			continue
		}

		// Store the moving average based on window duration
		switch maConfig.Window {
		case 5 * time.Minute:
			processedData.MovingAvg5m = ma
		case 15 * time.Minute:
			processedData.MovingAvg15m = ma
		case 1 * time.Hour:
			processedData.MovingAvg1h = ma
		}
	}

	// Calculate 24h changes
	priceChange24h, err := e.calculate24hChange(ctx, rawData.Symbol, "price")
	if err != nil {
		log.Printf("Failed to calculate 24h price change for %s: %v", rawData.Symbol, err)
	} else {
		processedData.PriceChange24h = priceChange24h
	}

	volumeChange24h, err := e.calculate24hChange(ctx, rawData.Symbol, "volume")
	if err != nil {
		log.Printf("Failed to calculate 24h volume change for %s: %v", rawData.Symbol, err)
	} else {
		processedData.VolumeChange24h = volumeChange24h
	}

	return processedData, nil
}

// calculateMovingAverage calculates moving average using the specified configuration
func (e *MetricsEngine) calculateMovingAverage(ctx context.Context, symbol string, config models.MovingAverageConfig) (float64, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("ma:%s:%s:%s", symbol, config.Method, config.Window.String())
	if cached := e.getFromCache(cacheKey); cached != nil {
		return *cached, nil
	}

	// Get historical data for the window
	data, err := e.getHistoricalData(ctx, symbol, config.Window)
	if err != nil {
		return 0, fmt.Errorf("failed to get historical data: %w", err)
	}

	if len(data) == 0 {
		return 0, fmt.Errorf("no data available for moving average calculation")
	}

	var result float64
	switch config.Method {
	case "simple":
		result = e.calculateSimpleMovingAverage(data)
	case "exponential":
		result = e.calculateExponentialMovingAverage(data, config.Alpha)
	case "weighted":
		result = e.calculateWeightedMovingAverage(data, config.Weights)
	default:
		return 0, fmt.Errorf("unsupported moving average method: %s", config.Method)
	}

	// Cache the result
	e.setCache(cacheKey, result)
	return result, nil
}

// calculateSimpleMovingAverage calculates a simple moving average
func (e *MetricsEngine) calculateSimpleMovingAverage(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

// calculateExponentialMovingAverage calculates an exponential moving average
func (e *MetricsEngine) calculateExponentialMovingAverage(data []float64, alpha float64) float64 {
	if len(data) == 0 {
		return 0
	}

	if alpha <= 0 || alpha > 1 {
		alpha = 0.1 // Default alpha
	}

	ema := data[0]
	for i := 1; i < len(data); i++ {
		ema = alpha*data[i] + (1-alpha)*ema
	}
	return ema
}

// calculateWeightedMovingAverage calculates a weighted moving average
func (e *MetricsEngine) calculateWeightedMovingAverage(data []float64, weights []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	// Use default weights if not provided
	if len(weights) == 0 {
		weights = make([]float64, len(data))
		for i := range weights {
			weights[i] = float64(i + 1) // Linear weights
		}
	}

	// Ensure weights array matches data length
	if len(weights) != len(data) {
		// Truncate or extend weights as needed
		if len(weights) > len(data) {
			weights = weights[:len(data)]
		} else {
			for i := len(weights); i < len(data); i++ {
				weights = append(weights, 1.0) // Default weight
			}
		}
	}

	sum := 0.0
	weightSum := 0.0
	for i, value := range data {
		sum += value * weights[i]
		weightSum += weights[i]
	}

	if weightSum == 0 {
		return 0
	}
	return sum / weightSum
}

// detectAnomalies detects anomalies in the data using the configured method
func (e *MetricsEngine) detectAnomalies(ctx context.Context, rawData *models.RawCryptoData, processedData *models.ProcessedCryptoData) ([]models.AnomalyDetectionResult, error) {
	var anomalies []models.AnomalyDetectionResult

	// Detect price anomalies
	if priceAnomaly, err := e.anomalyDetector.DetectPriceAnomaly(ctx, rawData); err == nil && priceAnomaly != nil {
		anomalies = append(anomalies, *priceAnomaly)
	}

	// Detect volume anomalies
	if volumeAnomaly, err := e.anomalyDetector.DetectVolumeAnomaly(ctx, rawData); err == nil && volumeAnomaly != nil {
		anomalies = append(anomalies, *volumeAnomaly)
	}

	// Store anomalies in Redis
	for _, anomaly := range anomalies {
		if err := e.storeAnomaly(ctx, &anomaly); err != nil {
			log.Printf("Failed to store anomaly: %v", err)
		}
	}

	return anomalies, nil
}

// performMetricRollups performs metric rollups for different time windows
func (e *MetricsEngine) performMetricRollups(ctx context.Context, rawData *models.RawCryptoData) error {
	for _, window := range e.config.RollupWindows {
		if err := e.metricRollup.ProcessRollup(ctx, rawData, window); err != nil {
			log.Printf("Failed to process rollup for window %s: %v", window, err)
			continue
		}
	}
	return nil
}

// evaluateAlertThresholds evaluates configured alert thresholds
func (e *MetricsEngine) evaluateAlertThresholds(ctx context.Context, processedData *models.ProcessedCryptoData, anomalies []models.AnomalyDetectionResult) error {
	// Evaluate thresholds for processed data
	if err := e.alertManager.EvaluateThresholds(ctx, processedData); err != nil {
		return fmt.Errorf("failed to evaluate thresholds: %w", err)
	}

	// Evaluate thresholds for anomalies
	for _, anomaly := range anomalies {
		if anomaly.IsSignificant() {
			if err := e.alertManager.EvaluateAnomalyThresholds(ctx, &anomaly); err != nil {
				log.Printf("Failed to evaluate anomaly thresholds: %v", err)
			}
		}
	}

	return nil
}

// Helper methods for data retrieval and storage
func (e *MetricsEngine) getHistoricalData(ctx context.Context, symbol string, window time.Duration) ([]float64, error) {
	// Implementation would retrieve historical data from Redis
	// This is a simplified version
	return []float64{}, nil
}

func (e *MetricsEngine) calculate24hChange(ctx context.Context, symbol, metric string) (float64, error) {
	// Implementation would calculate 24h change
	// This is a simplified version
	return 0.0, nil
}

func (e *MetricsEngine) storeInRedis(ctx context.Context, data *models.ProcessedCryptoData) error {
	// Implementation would store processed data in Redis
	return nil
}

func (e *MetricsEngine) storeAnomaly(ctx context.Context, anomaly *models.AnomalyDetectionResult) error {
	// Implementation would store anomaly in Redis
	return nil
}

// Cache management methods
func (e *MetricsEngine) getFromCache(key string) *float64 {
	e.cacheMux.RLock()
	defer e.cacheMux.RUnlock()

	if entry, exists := e.cache[key]; exists && time.Since(entry.timestamp) < e.cacheTTL {
		return &entry.value
	}
	return nil
}

func (e *MetricsEngine) setCache(key string, value float64) {
	e.cacheMux.Lock()
	defer e.cacheMux.Unlock()

	e.cache[key] = cacheEntry{
		value:     value,
		timestamp: time.Now(),
	}
}

// Observability helper methods
func (e *MetricsEngine) recordErrorMetric(ctx context.Context, errorType, errorMsg string, labels map[string]string) {
	if e.obsService != nil {
		e.obsService.RecordErrorMetric(ctx, errorType, errorMsg, labels)
	}
}

func (e *MetricsEngine) recordLatencyMetric(ctx context.Context, operation string, duration time.Duration, labels map[string]string) {
	if e.obsService != nil {
		e.obsService.RecordLatencyMetric(ctx, operation, duration, labels)
	}
}

func (e *MetricsEngine) recordBaseMetrics(ctx context.Context, rawData *models.RawCryptoData) {
	if e.obsService != nil {
		// Record price metric
		priceMetric := &models.TimeSeriesMetric{
			Symbol:    rawData.Symbol,
			Type:      models.MetricTypePrice,
			Value:     rawData.Price,
			Timestamp: rawData.Timestamp,
			Labels: map[string]string{
				"source": "kafka",
			},
		}
		e.obsService.RecordMetric(ctx, priceMetric)

		// Record volume metric
		volumeMetric := &models.TimeSeriesMetric{
			Symbol:    rawData.Symbol,
			Type:      models.MetricTypeVolume,
			Value:     rawData.Volume,
			Timestamp: rawData.Timestamp,
			Labels: map[string]string{
				"source": "kafka",
			},
		}
		e.obsService.RecordMetric(ctx, volumeMetric)
	}
}
