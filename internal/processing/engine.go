package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// Engine processes raw cryptocurrency data by calculating moving averages,
// 24-hour changes, and other metrics. It handles data ingestion from Kafka,
// processing, storage in Redis, and publishing processed results.
type Engine struct {
	consumer        *kafka.Consumer
	producer        *kafka.Producer
	redisClient     *redis.Client
	windowProcessor *WindowProcessor

	// Performance optimizations
	cache    map[string]cacheEntry
	cacheMux sync.RWMutex
	cacheTTL time.Duration
}

type cacheEntry struct {
	value     float64
	timestamp time.Time
}

// NewEngine creates a new processing engine with the given dependencies.
func NewEngine(consumer *kafka.Consumer, producer *kafka.Producer, redisClient *redis.Client) *Engine {
	return &Engine{
		consumer:        consumer,
		producer:        producer,
		redisClient:     redisClient,
		windowProcessor: NewWindowProcessor(redisClient),
		cache:           make(map[string]cacheEntry),
		cacheTTL:        30 * time.Second, // Cache for 30 seconds
	}
}

// Start begins consuming messages from Kafka and processing them.
// It blocks until the context is cancelled or an error occurs.
func (e *Engine) Start(ctx context.Context) error {
	return e.consumer.Consume(ctx, e.handleMessage)
}

// handleMessage processes a single message from Kafka.
// It unmarshals the raw data, validates it, processes it, stores it in Redis,
// and publishes the processed result.
func (e *Engine) handleMessage(ctx context.Context, key, value []byte) error {
	var rawData models.RawCryptoData
	if err := json.Unmarshal(value, &rawData); err != nil {
		return fmt.Errorf("failed to unmarshal raw data: %w", err)
	}

	// Validate the raw data
	if err := rawData.Validate(); err != nil {
		return fmt.Errorf("invalid raw data: %w", err)
	}

	// Process the data
	processedData, err := e.processData(ctx, &rawData)
	if err != nil {
		return fmt.Errorf("failed to process data: %w", err)
	}

	// Store in Redis for real-time queries
	if err := e.storeInRedis(ctx, processedData); err != nil {
		log.Printf("Failed to store data in Redis: %v", err)
		// Continue processing even if Redis storage fails
	}

	// Publish processed data
	if err := e.producer.Publish(ctx, processedData.Symbol, processedData); err != nil {
		return fmt.Errorf("failed to publish processed data: %w", err)
	}

	return nil
}

// processData calculates moving averages and 24-hour changes for the given raw data.
// It handles errors gracefully by logging them and continuing with default values.
func (e *Engine) processData(ctx context.Context, rawData *models.RawCryptoData) (*models.ProcessedCryptoData, error) {
	// Calculate moving averages with proper error handling
	ma5m, err := e.calculateMovingAverage(ctx, rawData.Symbol, 5*time.Minute)
	if err != nil {
		log.Printf("Failed to calculate 5m moving average for %s: %v", rawData.Symbol, err)
		// Continue with 0 value
	}

	ma15m, err := e.calculateMovingAverage(ctx, rawData.Symbol, 15*time.Minute)
	if err != nil {
		log.Printf("Failed to calculate 15m moving average for %s: %v", rawData.Symbol, err)
		// Continue with 0 value
	}

	ma1h, err := e.calculateMovingAverage(ctx, rawData.Symbol, 1*time.Hour)
	if err != nil {
		log.Printf("Failed to calculate 1h moving average for %s: %v", rawData.Symbol, err)
		// Continue with 0 value
	}

	// Calculate 24h changes with proper error handling
	priceChange24h, err := e.calculate24hChange(ctx, rawData.Symbol, "price")
	if err != nil {
		log.Printf("Failed to calculate 24h price change for %s: %v", rawData.Symbol, err)
		// Continue with 0 value
	}

	volumeChange24h, err := e.calculate24hChange(ctx, rawData.Symbol, "volume")
	if err != nil {
		log.Printf("Failed to calculate 24h volume change for %s: %v", rawData.Symbol, err)
		// Continue with 0 value
	}

	processedData := &models.ProcessedCryptoData{
		Symbol:          rawData.Symbol,
		Price:           rawData.Price,
		Volume:          rawData.Volume,
		PriceChange24h:  priceChange24h,
		VolumeChange24h: volumeChange24h,
		MovingAvg5m:     ma5m,
		MovingAvg15m:    ma15m,
		MovingAvg1h:     ma1h,
		Timestamp:       rawData.Timestamp,
		ProcessedAt:     time.Now(),
	}

	// Validate the processed data
	if err := processedData.Validate(); err != nil {
		return nil, fmt.Errorf("processed data validation failed: %w", err)
	}

	return processedData, nil
}

// calculateMovingAverage computes the moving average for a given symbol and time window.
// It retrieves historical data from Redis and calculates the average of values within the window.
func (e *Engine) calculateMovingAverage(ctx context.Context, symbol string, window time.Duration) (float64, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("ma:%s:%d", symbol, window)
	if cached := e.getFromCache(cacheKey); cached != nil {
		return *cached, nil
	}

	key := fmt.Sprintf("prices:%s", symbol)
	now := time.Now()
	start := now.Add(-window).Unix()

	// Use ZRangeByScore for efficient range query - only fetch data within the window
	prices, err := e.redisClient.ZRangeByScore(ctx, key, float64(start), float64(now.Unix()))
	if err != nil {
		return 0, err
	}

	var sum float64
	var count int

	for _, price := range prices {
		if val, ok := price.Member.(string); ok {
			// Parse the JSON string back to float64
			var parsedVal float64
			if err := json.Unmarshal([]byte(val), &parsedVal); err == nil {
				sum += parsedVal
				count++
			}
		}
	}

	var result float64
	if count == 0 {
		result = 0
	} else {
		result = sum / float64(count)
	}

	// Cache the result
	e.setCache(cacheKey, result)

	return result, nil
}

// getFromCache retrieves a value from the cache if it's still valid
func (e *Engine) getFromCache(key string) *float64 {
	e.cacheMux.RLock()
	defer e.cacheMux.RUnlock()

	if entry, exists := e.cache[key]; exists {
		if time.Since(entry.timestamp) < e.cacheTTL {
			return &entry.value
		}
		// Cache expired, remove it
		delete(e.cache, key)
	}
	return nil
}

// setCache stores a value in the cache
func (e *Engine) setCache(key string, value float64) {
	e.cacheMux.Lock()
	defer e.cacheMux.Unlock()

	e.cache[key] = cacheEntry{
		value:     value,
		timestamp: time.Now(),
	}
}

// calculate24hChange computes the percentage change over the last 24 hours for a given metric.
// It finds the closest value to 24 hours ago and calculates the percentage change from current value.
func (e *Engine) calculate24hChange(ctx context.Context, symbol, metric string) (float64, error) {
	key := fmt.Sprintf("%s:%s", metric, symbol)
	now := time.Now()
	past24h := now.Add(-24 * time.Hour).Unix()

	// Get current value (most recent)
	current, err := e.redisClient.ZRevRangeWithScores(ctx, key, 0, 0)
	if err != nil || len(current) == 0 {
		return 0, err
	}

	// Use ZRangeByScore to get values around 24h ago - more efficient than fetching all data
	// Look for values within a 1-hour window around 24h ago
	windowStart := past24h - 1800 // 30 minutes before
	windowEnd := past24h + 1800   // 30 minutes after

	past, err := e.redisClient.ZRangeByScore(ctx, key, float64(windowStart), float64(windowEnd))
	if err != nil || len(past) == 0 {
		return 0, err
	}

	// Find the closest value to 24h ago
	var pastVal float64
	var closestDiff float64 = float64(now.Unix()) // Initialize with max difference

	for _, item := range past {
		diff := math.Abs(item.Score - float64(past24h))
		if diff < closestDiff {
			closestDiff = diff
			if val, ok := item.Member.(string); ok {
				// Parse the JSON string back to float64
				var parsedVal float64
				if err := json.Unmarshal([]byte(val), &parsedVal); err == nil {
					pastVal = parsedVal
				}
			}
		}
	}

	currentVal := current[0].Score

	if pastVal == 0 {
		return 0, nil
	}

	return ((currentVal - pastVal) / pastVal) * 100, nil
}

// storeInRedis stores the processed data in Redis for real-time queries and historical analysis.
// It stores current values, price time series, and volume time series with appropriate TTLs.
func (e *Engine) storeInRedis(ctx context.Context, data *models.ProcessedCryptoData) error {
	// Store current price for real-time queries
	currentKey := fmt.Sprintf("current:%s", data.Symbol)
	if err := e.redisClient.Set(ctx, currentKey, data, 5*time.Minute); err != nil {
		return err
	}

	// Store in time-series for moving averages
	priceKey := fmt.Sprintf("prices:%s", data.Symbol)
	timestamp := float64(data.Timestamp.Unix())

	if err := e.redisClient.ZAdd(ctx, priceKey, timestamp, data.Price); err != nil {
		return err
	}

	volumeKey := fmt.Sprintf("volume:%s", data.Symbol)
	if err := e.redisClient.ZAdd(ctx, volumeKey, timestamp, data.Volume); err != nil {
		return err
	}

	// Clean up old data (older than 7 days) to prevent memory bloat
	// Only clean up occasionally to avoid performance impact
	if data.Timestamp.Unix()%60 == 0 { // Clean up every minute
		e.cleanupOldData(ctx, priceKey, volumeKey)
	}

	return nil
}

// cleanupOldData removes data older than 7 days from Redis sorted sets
// This prevents memory bloat and improves query performance
func (e *Engine) cleanupOldData(ctx context.Context, priceKey, volumeKey string) {
	// Remove data older than 7 days
	cutoff := time.Now().Add(-7 * 24 * time.Hour).Unix()

	// Clean up price data
	if _, err := e.redisClient.ZRemRangeByScore(ctx, priceKey, 0, float64(cutoff)); err != nil {
		log.Printf("Failed to cleanup old price data for %s: %v", priceKey, err)
	}

	// Clean up volume data
	if _, err := e.redisClient.ZRemRangeByScore(ctx, volumeKey, 0, float64(cutoff)); err != nil {
		log.Printf("Failed to cleanup old volume data for %s: %v", volumeKey, err)
	}
}
