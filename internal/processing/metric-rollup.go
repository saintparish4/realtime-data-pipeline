package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// MetricRollupProcessor handles time-series aggregation for metric rollups
type MetricRollupProcessor struct {
	redisClient   *redis.Client
	rollupWindows []time.Duration
}

// NewMetricRollupProcessor creates a new MetricRollupProcessor instance
func NewMetricRollupProcessor(redisClient *redis.Client, rollupWindows []time.Duration) *MetricRollupProcessor {
	return &MetricRollupProcessor{
		redisClient:   redisClient,
		rollupWindows: rollupWindows,
	}
}

// ProcessRollup processes metric rollups for a given time window
func (mrp *MetricRollupProcessor) ProcessRollup(ctx context.Context, rawData *models.RawCryptoData, window time.Duration) error {
	// Process price rollup
	if err := mrp.processPriceRollup(ctx, rawData, window); err != nil {
		return fmt.Errorf("failed to process price rollup: %w", err)
	}

	// Process volume rollup
	if err := mrp.processVolumeRollup(ctx, rawData, window); err != nil {
		return fmt.Errorf("failed to process volume rollup: %w", err)
	}

	return nil
}

// processPriceRollup processes price metric rollups
func (mrp *MetricRollupProcessor) processPriceRollup(ctx context.Context, rawData *models.RawCryptoData, window time.Duration) error {
	rollup, err := mrp.calculateRollup(ctx, rawData.Symbol, models.MetricTypePrice, window, rawData.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to calculate price rollup: %w", err)
	}

	if rollup != nil {
		if err := mrp.storeRollup(ctx, rollup); err != nil {
			return fmt.Errorf("failed to store price rollup: %w", err)
		}
	}

	return nil
}

// processVolumeRollup processes volume metric rollups
func (mrp *MetricRollupProcessor) processVolumeRollup(ctx context.Context, rawData *models.RawCryptoData, window time.Duration) error {
	rollup, err := mrp.calculateRollup(ctx, rawData.Symbol, models.MetricTypeVolume, window, rawData.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to calculate volume rollup: %w", err)
	}

	if rollup != nil {
		if err := mrp.storeRollup(ctx, rollup); err != nil {
			return fmt.Errorf("failed to store volume rollup: %w", err)
		}
	}

	return nil
}

// calculateRollup calculates metric rollup for a given symbol, metric type, and time window
func (mrp *MetricRollupProcessor) calculateRollup(ctx context.Context, symbol string, metricType models.MetricType, window time.Duration, currentTime time.Time) (*models.MetricRollup, error) {
	// Calculate window boundaries
	windowStart := currentTime.Truncate(window)
	windowEnd := windowStart.Add(window)

	// Get data for the window
	data, err := mrp.getWindowData(ctx, symbol, metricType, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get window data: %w", err)
	}

	if len(data) == 0 {
		return nil, nil // No data to rollup
	}

	// Calculate rollup statistics
	rollup := &models.MetricRollup{
		Symbol:    symbol,
		Type:      metricType,
		Window:    window,
		StartTime: windowStart,
		EndTime:   windowEnd,
		Count:     int64(len(data)),
		Labels: map[string]string{
			"window": window.String(),
		},
	}

	// Calculate basic statistics
	rollup.Sum = calculateSum(data)
	rollup.Min = calculateMin(data)
	rollup.Max = calculateMax(data)
	rollup.Average = rollup.Sum / float64(rollup.Count)

	// Calculate standard deviation
	rollup.StdDev = calculateStdDev(data, rollup.Average)

	// Calculate percentiles
	sortedData := make([]float64, len(data))
	copy(sortedData, data)
	sort.Float64s(sortedData)

	rollup.Median = calculatePercentile(sortedData, 50)
	rollup.Percentile95 = calculatePercentile(sortedData, 95)
	rollup.Percentile99 = calculatePercentile(sortedData, 99)

	return rollup, nil
}

// getWindowData retrieves data for a specific time window
func (mrp *MetricRollupProcessor) getWindowData(ctx context.Context, symbol string, metricType models.MetricType, startTime, endTime time.Time) ([]float64, error) {
	key := fmt.Sprintf("metrics:%s:%s", metricType, symbol)
	startScore := float64(startTime.Unix())
	endScore := float64(endTime.Unix())

	results, err := mrp.redisClient.ZRangeByScore(ctx, key, startScore, endScore)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from Redis: %w", err)
	}

	values := make([]float64, 0, len(results))
	for _, result := range results {
		var value float64
		if err := json.Unmarshal([]byte(result.Member.(string)), &value); err != nil {
			log.Printf("Failed to unmarshal metric value: %v", err)
			continue
		}
		values = append(values, value)
	}

	return values, nil
}

// storeRollup stores the metric rollup in Redis
func (mrp *MetricRollupProcessor) storeRollup(ctx context.Context, rollup *models.MetricRollup) error {
	// Validate rollup before storing
	if err := rollup.Validate(); err != nil {
		return fmt.Errorf("invalid rollup: %w", err)
	}

	// Serialize rollup to JSON
	rollupData, err := json.Marshal(rollup)
	if err != nil {
		return fmt.Errorf("failed to marshal rollup: %w", err)
	}

	// Store in Redis sorted set
	key := rollup.GetRedisKey()
	score := rollup.GetRedisScore()

	if err := mrp.redisClient.ZAdd(ctx, key, score, string(rollupData)); err != nil {
		return fmt.Errorf("failed to store rollup in Redis: %w", err)
	}

	// Set expiration for the key (e.g., 7 days)
	if err := mrp.redisClient.Expire(ctx, key, 7*24*time.Hour); err != nil {
		log.Printf("Failed to set expiration for rollup key %s: %v", key, err)
	}

	return nil
}

// GetRollup retrieves a metric rollup for a specific time window
func (mrp *MetricRollupProcessor) GetRollup(ctx context.Context, symbol string, metricType models.MetricType, window time.Duration, timestamp time.Time) (*models.MetricRollup, error) {
	windowStart := timestamp.Truncate(window)
	windowEnd := windowStart.Add(window)

	key := fmt.Sprintf("rollups:%s:%s:%s", symbol, metricType, window.String())
	startScore := float64(windowStart.Unix())
	endScore := float64(windowEnd.Unix())

	results, err := mrp.redisClient.ZRangeByScore(ctx, key, startScore, endScore)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve rollup from Redis: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no rollup found for the specified time window")
	}

	// Get the most recent rollup in the window
	latestResult := results[len(results)-1]

	var rollup models.MetricRollup
	if err := json.Unmarshal([]byte(latestResult.Member.(string)), &rollup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rollup: %w", err)
	}

	return &rollup, nil
}

// GetRollups retrieves multiple metric rollups for a time range
func (mrp *MetricRollupProcessor) GetRollups(ctx context.Context, symbol string, metricType models.MetricType, window time.Duration, startTime, endTime time.Time) ([]*models.MetricRollup, error) {
	key := fmt.Sprintf("rollups:%s:%s:%s", symbol, metricType, window.String())
	startScore := float64(startTime.Unix())
	endScore := float64(endTime.Unix())

	results, err := mrp.redisClient.ZRangeByScore(ctx, key, startScore, endScore)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve rollups from Redis: %w", err)
	}

	rollups := make([]*models.MetricRollup, 0, len(results))
	for _, result := range results {
		var rollup models.MetricRollup
		if err := json.Unmarshal([]byte(result.Member.(string)), &rollup); err != nil {
			log.Printf("Failed to unmarshal rollup: %v", err)
			continue
		}
		rollups = append(rollups, &rollup)
	}

	return rollups, nil
}

// CleanupOldRollups removes old rollup data to prevent memory bloat
func (mrp *MetricRollupProcessor) CleanupOldRollups(ctx context.Context, retentionPeriod time.Duration) error {
	cutoffTime := time.Now().Add(-retentionPeriod)
	cutoffScore := float64(cutoffTime.Unix())

	// Get all rollup keys
	pattern := "rollups:*"
	keys, err := mrp.redisClient.Keys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("failed to get rollup keys: %w", err)
	}

	for _, key := range keys {
		// Remove old entries from each rollup key
		if _, err := mrp.redisClient.ZRemRangeByScore(ctx, key, 0, cutoffScore); err != nil {
			log.Printf("Failed to cleanup old rollups for key %s: %v", key, err)
			continue
		}
	}

	return nil
}

// Statistical helper functions
func calculateSum(data []float64) float64 {
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum
}

func calculateMin(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	min := data[0]
	for _, value := range data {
		if value < min {
			min = value
		}
	}
	return min
}

func calculateMax(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	max := data[0]
	for _, value := range data {
		if value > max {
			max = value
		}
	}
	return max
}
