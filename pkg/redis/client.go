package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &Client{rdb: rdb}
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.rdb.Set(ctx, key, data, expiration).Err()
}

func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

func (c *Client) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	data, err := json.Marshal(member)
	if err != nil {
		return err
	}

	return c.rdb.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

func (c *Client) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return c.rdb.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

// ZRangeByScore returns members with scores between min and max
func (c *Client) ZRangeByScore(ctx context.Context, key string, min, max float64) ([]redis.Z, error) {
	return c.rdb.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
}

// ZRemRangeByScore removes members with scores between min and max
func (c *Client) ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error) {
	return c.rdb.ZRemRangeByScore(ctx, key, fmt.Sprintf("%f", min), fmt.Sprintf("%f", max)).Result()
}

// Expire sets a timeout on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

// Keys returns all keys matching the given pattern
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.rdb.Keys(ctx, pattern).Result()
}

// ===== OBSERVABILITY & ALERTING METHODS =====

// StoreTimeSeriesMetric stores a time-series metric in Redis
func (c *Client) StoreTimeSeriesMetric(ctx context.Context, metric *models.TimeSeriesMetric) error {
	if err := metric.Validate(); err != nil {
		return fmt.Errorf("invalid metric: %w", err)
	}

	key := metric.GetRedisKey()
	score := metric.GetRedisScore()

	// Store the metric in a sorted set for time-series queries
	if err := c.ZAdd(ctx, key, score, metric); err != nil {
		return fmt.Errorf("failed to store time-series metric: %w", err)
	}

	// Set TTL for automatic cleanup (e.g., 30 days for metrics)
	if err := c.Expire(ctx, key, 30*24*time.Hour); err != nil {
		return fmt.Errorf("failed to set TTL for metric: %w", err)
	}

	return nil
}

// QueryTimeSeriesMetrics retrieves time-series metrics for a given query
func (c *Client) QueryTimeSeriesMetrics(ctx context.Context, query *models.TimeSeriesQuery) ([]*models.TimeSeriesMetric, error) {
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	key := query.GetRedisKey()
	minScore := query.GetMinScore()
	maxScore := query.GetMaxScore()

	// Get metrics within the time range
	results, err := c.ZRangeByScore(ctx, key, minScore, maxScore)
	if err != nil {
		return nil, fmt.Errorf("failed to query time-series metrics: %w", err)
	}

	var metrics []*models.TimeSeriesMetric
	for _, result := range results {
		var metric models.TimeSeriesMetric
		if err := json.Unmarshal([]byte(result.Member.(string)), &metric); err != nil {
			continue // Skip invalid entries
		}
		metrics = append(metrics, &metric)
	}

	// Apply limit if specified
	if query.Limit > 0 && len(metrics) > query.Limit {
		metrics = metrics[:query.Limit]
	}

	return metrics, nil
}

// StoreAlert stores an alert in Redis
func (c *Client) StoreAlert(ctx context.Context, alert *models.Alert) error {
	if err := alert.Validate(); err != nil {
		return fmt.Errorf("invalid alert: %w", err)
	}

	key := alert.GetRedisKey()

	// Store the alert
	if err := c.Set(ctx, key, alert, 0); err != nil {
		return fmt.Errorf("failed to store alert: %w", err)
	}

	// Also store in a set for active alerts if it's active
	if alert.IsActive() {
		activeKey := "alerts:active"
		if err := c.rdb.SAdd(ctx, activeKey, alert.ID).Err(); err != nil {
			return fmt.Errorf("failed to add to active alerts: %w", err)
		}
	}

	return nil
}

// GetAlert retrieves an alert by ID
func (c *Client) GetAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	key := fmt.Sprintf("alerts:%s", alertID)

	var alert models.Alert
	if err := c.Get(ctx, key, &alert); err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return &alert, nil
}

// GetActiveAlerts retrieves all active alerts
func (c *Client) GetActiveAlerts(ctx context.Context) ([]*models.Alert, error) {
	activeKey := "alerts:active"

	// Get all active alert IDs
	alertIDs, err := c.rdb.SMembers(ctx, activeKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active alert IDs: %w", err)
	}

	var alerts []*models.Alert
	for _, alertID := range alertIDs {
		alert, err := c.GetAlert(ctx, alertID)
		if err != nil {
			continue // Skip invalid alerts
		}
		alerts = append(alerts, alert)
	}

	// Ensure we always return an empty slice instead of nil
	if alerts == nil {
		alerts = []*models.Alert{}
	}

	return alerts, nil
}

// UpdateAlertStatus updates the status of an alert
func (c *Client) UpdateAlertStatus(ctx context.Context, alertID string, status models.AlertStatus) error {
	alert, err := c.GetAlert(ctx, alertID)
	if err != nil {
		return fmt.Errorf("failed to get alert for status update: %w", err)
	}

	// Update status
	alert.Status = status
	alert.UpdatedAt = time.Now()

	// Handle status-specific actions
	switch status {
	case models.AlertStatusActive:
		// Add to active alerts set
		activeKey := "alerts:active"
		if err := c.rdb.SAdd(ctx, activeKey, alertID).Err(); err != nil {
			return fmt.Errorf("failed to add to active alerts: %w", err)
		}
	case models.AlertStatusResolved, models.AlertStatusSilenced:
		// Remove from active alerts set
		activeKey := "alerts:active"
		if err := c.rdb.SRem(ctx, activeKey, alertID).Err(); err != nil {
			return fmt.Errorf("failed to remove from active alerts: %w", err)
		}
	}

	// Store updated alert
	return c.StoreAlert(ctx, alert)
}

// StoreAlertNotification stores an alert notification
func (c *Client) StoreAlertNotification(ctx context.Context, notification *models.AlertNotification) error {
	if err := notification.Validate(); err != nil {
		return fmt.Errorf("invalid notification: %w", err)
	}

	key := notification.GetRedisKey()

	// Store the notification with TTL (e.g., 7 days)
	if err := c.Set(ctx, key, notification, 7*24*time.Hour); err != nil {
		return fmt.Errorf("failed to store notification: %w", err)
	}

	return nil
}

// GetRateLimit retrieves a rate limit by key
func (c *Client) GetRateLimit(ctx context.Context, key string) (*models.RateLimit, error) {
	redisKey := fmt.Sprintf("rate_limit:%s", key)

	var rateLimit models.RateLimit
	if err := c.Get(ctx, redisKey, &rateLimit); err != nil {
		if err == redis.Nil {
			// Rate limit doesn't exist, return nil
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}

	return &rateLimit, nil
}

// SetRateLimit stores or updates a rate limit
func (c *Client) SetRateLimit(ctx context.Context, rateLimit *models.RateLimit) error {
	if err := rateLimit.Validate(); err != nil {
		return fmt.Errorf("invalid rate limit: %w", err)
	}

	key := rateLimit.GetRedisKey()

	// Store the rate limit with TTL based on the window
	ttl := rateLimit.Window + time.Hour // Add buffer
	if err := c.Set(ctx, key, rateLimit, ttl); err != nil {
		return fmt.Errorf("failed to store rate limit: %w", err)
	}

	return nil
}

// CheckAndIncrementRateLimit checks if a rate limit is exceeded and increments it
func (c *Client) CheckAndIncrementRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	rateLimit, err := c.GetRateLimit(ctx, key)
	if err != nil {
		return false, err
	}

	// Create new rate limit if it doesn't exist
	if rateLimit == nil {
		rateLimit = &models.RateLimit{
			Key:       key,
			Limit:     limit,
			Window:    window,
			Current:   0,
			ResetAt:   time.Now().Add(window),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Check if rate limit is exceeded
	if rateLimit.IsExceeded() {
		return true, nil // Rate limit exceeded
	}

	// Increment the rate limit
	rateLimit.Increment()

	// Store the updated rate limit
	if err := c.SetRateLimit(ctx, rateLimit); err != nil {
		return false, fmt.Errorf("failed to update rate limit: %w", err)
	}

	return false, nil // Rate limit not exceeded
}

// CleanupOldMetrics removes metrics older than the specified duration
func (c *Client) CleanupOldMetrics(ctx context.Context, olderThan time.Duration) error {
	// Get all metric keys
	pattern := "metrics:*"
	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get metric keys: %w", err)
	}

	cutoffTime := time.Now().Add(-olderThan)
	cutoffScore := float64(cutoffTime.Unix())

	var totalRemoved int64
	for _, key := range keys {
		// Remove metrics older than cutoff
		removed, err := c.ZRemRangeByScore(ctx, key, 0, cutoffScore)
		if err != nil {
			continue // Continue with other keys even if one fails
		}
		totalRemoved += removed
	}

	return nil
}

// GetMetricStats returns basic statistics for a metric
func (c *Client) GetMetricStats(ctx context.Context, symbol string, metricType models.MetricType) (map[string]interface{}, error) {
	key := fmt.Sprintf("metrics:%s:%s", metricType, symbol)

	// Get count of metrics
	count, err := c.rdb.ZCard(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metric count: %w", err)
	}

	// Get min and max scores (timestamps)
	minScore, err := c.rdb.ZRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get min score: %w", err)
	}

	maxScore, err := c.rdb.ZRevRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get max score: %w", err)
	}

	stats := map[string]interface{}{
		"count":  count,
		"symbol": symbol,
		"type":   metricType,
	}

	if len(minScore) > 0 {
		stats["oldest_timestamp"] = time.Unix(int64(minScore[0].Score), 0)
	}
	if len(maxScore) > 0 {
		stats["newest_timestamp"] = time.Unix(int64(maxScore[0].Score), 0)
	}

	return stats, nil
}

// DeleteAlert deletes an alert and removes it from active alerts
func (c *Client) DeleteAlert(ctx context.Context, alertID string) error {
	key := fmt.Sprintf("alerts:%s", alertID)

	// Delete the alert
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	// Remove from active alerts set
	activeKey := "alerts:active"
	if err := c.rdb.SRem(ctx, activeKey, alertID).Err(); err != nil {
		return fmt.Errorf("failed to remove from active alerts: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}
