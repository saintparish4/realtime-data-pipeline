package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// RateLimiter handles rate limiting for notifications
type RateLimiter struct {
	redisClient *redis.Client
	config      struct {
		MaxPerMinute int           `yaml:"max_per_minute"`
		MaxPerHour   int           `yaml:"max_per_hour"`
		Window       time.Duration `yaml:"window"`
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, config struct {
	MaxPerMinute int           `yaml:"max_per_minute"`
	MaxPerHour   int           `yaml:"max_per_hour"`
	Window       time.Duration `yaml:"window"`
}) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
		config:      config,
	}
}

// CheckLimit checks if a notification can be sent based on rate limits
func (r *RateLimiter) CheckLimit(ctx context.Context, notificationType NotificationType, priority Priority) error {
	now := time.Now()

	// Check per-minute limit
	minuteKey := fmt.Sprintf("rate_limit:%s:%s:minute:%d", notificationType, priority, now.Unix()/60)
	if err := r.checkWindowLimit(ctx, minuteKey, r.config.MaxPerMinute, time.Minute); err != nil {
		return fmt.Errorf("minute rate limit exceeded: %w", err)
	}

	// Check per-hour limit
	hourKey := fmt.Sprintf("rate_limit:%s:%s:hour:%d", notificationType, priority, now.Unix()/3600)
	if err := r.checkWindowLimit(ctx, hourKey, r.config.MaxPerHour, time.Hour); err != nil {
		return fmt.Errorf("hour rate limit exceeded: %w", err)
	}

	// Increment counters
	r.incrementCounter(ctx, minuteKey, time.Minute)
	r.incrementCounter(ctx, hourKey, time.Hour)

	return nil
}

// checkWindowLimit checks if the limit for a specific time window has been exceeded
func (r *RateLimiter) checkWindowLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	// Use the existing CheckAndIncrementRateLimit method
	allowed, err := r.redisClient.CheckAndIncrementRateLimit(ctx, key, limit, window)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	if !allowed {
		return fmt.Errorf("rate limit exceeded: %d", limit)
	}

	return nil
}

// incrementCounter increments the counter for a rate limit window
func (r *RateLimiter) incrementCounter(ctx context.Context, key string, window time.Duration) {
	// The CheckAndIncrementRateLimit method already handles the increment
	// This method is kept for compatibility but doesn't need to do anything
}

// GetRateLimitStatus returns the current rate limit status
func (r *RateLimiter) GetRateLimitStatus(ctx context.Context, notificationType NotificationType, priority Priority) (map[string]interface{}, error) {
	now := time.Now()

	minuteKey := fmt.Sprintf("rate_limit:%s:%s:minute:%d", notificationType, priority, now.Unix()/60)
	hourKey := fmt.Sprintf("rate_limit:%s:%s:hour:%d", notificationType, priority, now.Unix()/3600)

	minuteCount, _ := r.getCounter(ctx, minuteKey)
	hourCount, _ := r.getCounter(ctx, hourKey)

	return map[string]interface{}{
		"notification_type": string(notificationType),
		"priority":          string(priority),
		"minute_count":      minuteCount,
		"minute_limit":      r.config.MaxPerMinute,
		"hour_count":        hourCount,
		"hour_limit":        r.config.MaxPerHour,
		"timestamp":         now,
	}, nil
}

// getCounter gets the current count for a rate limit key
func (r *RateLimiter) getCounter(ctx context.Context, key string) (int, error) {
	var countStr string
	if err := r.redisClient.Get(ctx, key, &countStr); err != nil {
		return 0, err
	}

	var currentCount int
	if _, err := fmt.Sscanf(countStr, "%d", &currentCount); err != nil {
		return 0, err
	}
	return currentCount, nil
}

// ResetRateLimit resets the rate limit for a specific type and priority
func (r *RateLimiter) ResetRateLimit(ctx context.Context, notificationType NotificationType, priority Priority) error {
	now := time.Now()

	minuteKey := fmt.Sprintf("rate_limit:%s:%s:minute:%d", notificationType, priority, now.Unix()/60)
	hourKey := fmt.Sprintf("rate_limit:%s:%s:hour:%d", notificationType, priority, now.Unix()/3600)

	// Set the keys to expire immediately
	r.redisClient.Expire(ctx, minuteKey, 0)
	r.redisClient.Expire(ctx, hourKey, 0)

	return nil
}
