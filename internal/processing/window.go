// Package processing provides data processing capabilities for the real-time data pipeline.
// It includes window-based aggregation of cryptocurrency data for OHLCV analysis.
package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// Redis key patterns for storing time-series data
const (
	PriceKeyPattern  = "prices:%s"
	VolumeKeyPattern = "volume:%s"
)

// WindowProcessor handles time-window based aggregation of cryptocurrency data.
// It retrieves data from Redis and calculates OHLCV (Open, High, Low, Close, Volume) metrics.
type WindowProcessor struct {
	redisClient *redis.Client
}

// NewWindowProcessor creates a new WindowProcessor instance with the provided Redis client.
func NewWindowProcessor(redisClient *redis.Client) *WindowProcessor {
	return &WindowProcessor{redisClient: redisClient}
}

// ProcessWindow aggregates cryptocurrency data for a given symbol and time window.
// It retrieves price and volume data from Redis and calculates OHLCV metrics.
// The window is calculated from the current time, truncated to the nearest window boundary.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - symbol: The cryptocurrency symbol (e.g., "BTC", "ETH")
//   - window: The time window duration for aggregation
//
// Returns:
//   - AggregatedCryptoData containing OHLCV metrics
//   - error if the operation fails
func (w *WindowProcessor) ProcessWindow(ctx context.Context, symbol string, window time.Duration) (*models.AggregatedCryptoData, error) {
	// Input validation
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}
	if window <= 0 {
		return nil, fmt.Errorf("window duration must be positive")
	}
	if w.redisClient == nil {
		return nil, fmt.Errorf("redis client is not initialized")
	}

	now := time.Now()
	windowStart := now.Truncate(window)
	windowEnd := windowStart.Add(window)

	// Get data for the window
	priceKey := fmt.Sprintf(PriceKeyPattern, symbol)
	volumeKey := fmt.Sprintf(VolumeKeyPattern, symbol)

	prices, err := w.getWindowData(ctx, priceKey, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get price data: %w", err)
	}

	volumes, err := w.getWindowData(ctx, volumeKey, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume data: %w", err)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no price data available for symbol %s in window %s", symbol, window.String())
	}

	// Calculate OHLCV in a single pass
	var open, high, low, close, totalVolume float64
	var tradeCount int64

	open = prices[0]
	close = prices[len(prices)-1]
	high = prices[0]
	low = prices[0]

	for i, price := range prices {
		if price > high {
			high = price
		}
		if price < low {
			low = price
		}
		tradeCount++

		// Add volume if available
		if i < len(volumes) {
			totalVolume += volumes[i]
		}
	}

	// Validate calculated values
	if high < low {
		return nil, fmt.Errorf("invalid OHLC data: high price (%f) is less than low price (%f)", high, low)
	}

	return &models.AggregatedCryptoData{
		Symbol:      symbol,
		TimeWindow:  window.String(),
		OpenPrice:   open,
		ClosePrice:  close,
		HighPrice:   high,
		LowPrice:    low,
		Volume:      totalVolume,
		TradeCount:  tradeCount,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	}, nil
}

// getWindowData retrieves time-series data from Redis for a given key and time range.
// It uses Redis sorted sets with timestamps as scores to efficiently query time ranges.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - key: Redis key for the sorted set
//   - start: Start time for the query range
//   - end: End time for the query range
//
// Returns:
//   - Slice of float64 values within the time range
//   - error if the operation fails
func (w *WindowProcessor) getWindowData(ctx context.Context, key string, start, end time.Time) ([]float64, error) {
	startScore := float64(start.Unix())
	endScore := float64(end.Unix())

	// Retrieve data from Redis sorted set within the time window
	results, err := w.redisClient.ZRangeByScore(ctx, key, startScore, endScore)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from Redis for key %s: %w", key, err)
	}

	// Parse the results into float64 values
	values := make([]float64, 0, len(results))
	for _, result := range results {
		var value float64
		if err := json.Unmarshal([]byte(result.Member.(string)), &value); err != nil {
			// Skip invalid data points but log the error
			continue
		}
		values = append(values, value)
	}

	return values, nil
}
