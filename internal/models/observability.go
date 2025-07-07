package models

import (
	"fmt"
	"time"
)

// MetricType represents the type of metric being stored
type MetricType string

const (
	MetricTypePrice   MetricType = "price"
	MetricTypeVolume  MetricType = "volume"
	MetricTypeTrade   MetricType = "trade"
	MetricTypeLatency MetricType = "latency"
	MetricTypeError   MetricType = "error"
	MetricTypeCustom  MetricType = "custom"
)

// TimeSeriesMetric represents a time-series data point for observability
type TimeSeriesMetric struct {
	Symbol    string                 `json:"symbol"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates the TimeSeriesMetric
func (m *TimeSeriesMetric) Validate() error {
	if m.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if m.Type == "" {
		return fmt.Errorf("metric type cannot be empty")
	}
	if m.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}
	return nil
}

// GetRedisKey returns the Redis key for storing this metric
func (m *TimeSeriesMetric) GetRedisKey() string {
	return fmt.Sprintf("metrics:%s:%s", m.Type, m.Symbol)
}

// GetRedisScore returns the Redis score (timestamp) for sorted set operations
func (m *TimeSeriesMetric) GetRedisScore() float64 {
	return float64(m.Timestamp.Unix())
}

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
)

// Alert represents an alert condition and its current state
type Alert struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Severity     AlertSeverity          `json:"severity"`
	Status       AlertStatus            `json:"status"`
	Symbol       string                 `json:"symbol,omitempty"`
	Condition    string                 `json:"condition"`
	Threshold    float64                `json:"threshold"`
	CurrentValue float64                `json:"current_value"`
	TriggeredAt  *time.Time             `json:"triggered_at,omitempty"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Validate validates the Alert
func (a *Alert) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("alert ID cannot be empty")
	}
	if a.Name == "" {
		return fmt.Errorf("alert name cannot be empty")
	}
	if a.Condition == "" {
		return fmt.Errorf("alert condition cannot be empty")
	}
	if a.Severity == "" {
		return fmt.Errorf("alert severity cannot be empty")
	}
	if a.Status == "" {
		return fmt.Errorf("alert status cannot be empty")
	}
	return nil
}

// GetRedisKey returns the Redis key for storing this alert
func (a *Alert) GetRedisKey() string {
	return fmt.Sprintf("alerts:%s", a.ID)
}

// IsActive returns true if the alert is currently active
func (a *Alert) IsActive() bool {
	return a.Status == AlertStatusActive
}

// IsResolved returns true if the alert has been resolved
func (a *Alert) IsResolved() bool {
	return a.Status == AlertStatusResolved
}

// IsSilenced returns true if the alert is silenced
func (a *Alert) IsSilenced() bool {
	return a.Status == AlertStatusSilenced
}

// Trigger marks the alert as triggered
func (a *Alert) Trigger(currentValue float64) {
	now := time.Now()
	a.Status = AlertStatusActive
	a.CurrentValue = currentValue
	a.TriggeredAt = &now
	a.UpdatedAt = now
}

// Resolve marks the alert as resolved
func (a *Alert) Resolve() {
	now := time.Now()
	a.Status = AlertStatusResolved
	a.ResolvedAt = &now
	a.UpdatedAt = now
}

// Silence marks the alert as silenced
func (a *Alert) Silence() {
	a.Status = AlertStatusSilenced
	a.UpdatedAt = time.Now()
}

// AlertNotification represents a notification sent for an alert
type AlertNotification struct {
	ID        string                 `json:"id"`
	AlertID   string                 `json:"alert_id"`
	Type      string                 `json:"type"` // email, slack, webhook, etc.
	Recipient string                 `json:"recipient"`
	Message   string                 `json:"message"`
	Status    string                 `json:"status"` // sent, failed, pending
	SentAt    *time.Time             `json:"sent_at,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Validate validates the AlertNotification
func (n *AlertNotification) Validate() error {
	if n.ID == "" {
		return fmt.Errorf("notification ID cannot be empty")
	}
	if n.AlertID == "" {
		return fmt.Errorf("alert ID cannot be empty")
	}
	if n.Type == "" {
		return fmt.Errorf("notification type cannot be empty")
	}
	if n.Recipient == "" {
		return fmt.Errorf("recipient cannot be empty")
	}
	if n.Message == "" {
		return fmt.Errorf("message cannot be empty")
	}
	return nil
}

// GetRedisKey returns the Redis key for storing this notification
func (n *AlertNotification) GetRedisKey() string {
	return fmt.Sprintf("notifications:%s", n.ID)
}

// RateLimit represents rate limiting configuration for alert notifications
type RateLimit struct {
	Key       string        `json:"key"`
	Limit     int           `json:"limit"`
	Window    time.Duration `json:"window"`
	Current   int           `json:"current"`
	ResetAt   time.Time     `json:"reset_at"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Validate validates the RateLimit
func (r *RateLimit) Validate() error {
	if r.Key == "" {
		return fmt.Errorf("rate limit key cannot be empty")
	}
	if r.Limit <= 0 {
		return fmt.Errorf("rate limit must be greater than 0")
	}
	if r.Window <= 0 {
		return fmt.Errorf("rate limit window must be greater than 0")
	}
	return nil
}

// GetRedisKey returns the Redis key for storing this rate limit
func (r *RateLimit) GetRedisKey() string {
	return fmt.Sprintf("rate_limit:%s", r.Key)
}

// IsExceeded returns true if the rate limit has been exceeded
func (r *RateLimit) IsExceeded() bool {
	// Check if window has reset
	if time.Now().After(r.ResetAt) {
		r.Current = 0
		r.ResetAt = time.Now().Add(r.Window)
		r.UpdatedAt = time.Now()
	}
	return r.Current >= r.Limit
}

// Increment increments the current count
func (r *RateLimit) Increment() {
	if time.Now().After(r.ResetAt) {
		r.Current = 1
		r.ResetAt = time.Now().Add(r.Window)
	} else {
		r.Current++
	}
	r.UpdatedAt = time.Now()
}

// TimeSeriesQuery represents a query for time-series data
type TimeSeriesQuery struct {
	Symbol    string            `json:"symbol"`
	Type      MetricType        `json:"type"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Limit     int               `json:"limit,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// Validate validates the TimeSeriesQuery
func (q *TimeSeriesQuery) Validate() error {
	if q.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if q.Type == "" {
		return fmt.Errorf("metric type cannot be empty")
	}
	if q.StartTime.IsZero() {
		return fmt.Errorf("start time cannot be zero")
	}
	if q.EndTime.IsZero() {
		return fmt.Errorf("end time cannot be zero")
	}
	if q.StartTime.After(q.EndTime) {
		return fmt.Errorf("start time cannot be after end time")
	}
	return nil
}

// GetRedisKey returns the Redis key for querying this metric
func (q *TimeSeriesQuery) GetRedisKey() string {
	return fmt.Sprintf("metrics:%s:%s", q.Type, q.Symbol)
}

// GetMinScore returns the minimum score for Redis range query
func (q *TimeSeriesQuery) GetMinScore() float64 {
	return float64(q.StartTime.Unix())
}

// GetMaxScore returns the maximum score for Redis range query
func (q *TimeSeriesQuery) GetMaxScore() float64 {
	return float64(q.EndTime.Unix())
}
