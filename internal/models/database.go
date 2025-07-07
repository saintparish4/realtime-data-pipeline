package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// TimescaleDB Models for Metrics Storage

// MetricRecord represents a metric record stored in TimescaleDB
type MetricRecord struct {
	ID        int64      `db:"id" json:"id"`
	Symbol    string     `db:"symbol" json:"symbol"`
	Type      MetricType `db:"type" json:"type"`
	Value     float64    `db:"value" json:"value"`
	Timestamp time.Time  `db:"timestamp" json:"timestamp"`
	Labels    JSONMap    `db:"labels" json:"labels"`
	Metadata  JSONMap    `db:"metadata" json:"metadata"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// JSONMap is a custom type for storing JSON data in PostgreSQL
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONMap", value)
	}

	return json.Unmarshal(bytes, j)
}

// MetricAggregation represents aggregated metric data
type MetricAggregation struct {
	Symbol     string     `db:"symbol" json:"symbol"`
	Type       MetricType `db:"type" json:"type"`
	TimeBucket time.Time  `db:"time_bucket" json:"time_bucket"`
	Count      int64      `db:"count" json:"count"`
	Min        float64    `db:"min" json:"min"`
	Max        float64    `db:"max" json:"max"`
	Avg        float64    `db:"avg" json:"avg"`
	Sum        float64    `db:"sum" json:"sum"`
}

// PostgreSQL Models for Alert Rules and Configurations

// AlertRule represents an alert rule stored in PostgreSQL
type AlertRule struct {
	ID          string        `db:"id" json:"id"`
	Name        string        `db:"name" json:"name"`
	Description string        `db:"description" json:"description"`
	Severity    AlertSeverity `db:"severity" json:"severity"`
	Symbol      string        `db:"symbol" json:"symbol"`
	Condition   string        `db:"condition" json:"condition"`
	Threshold   float64       `db:"threshold" json:"threshold"`
	Enabled     bool          `db:"enabled" json:"enabled"`
	Labels      JSONMap       `db:"labels" json:"labels"`
	Metadata    JSONMap       `db:"metadata" json:"metadata"`
	CreatedAt   time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at" json:"updated_at"`
}

// AlertRuleNotification represents notification configuration for an alert rule
type AlertRuleNotification struct {
	ID          string    `db:"id" json:"id"`
	AlertRuleID string    `db:"alert_rule_id" json:"alert_rule_id"`
	Type        string    `db:"type" json:"type"` // email, slack, webhook, etc.
	Recipient   string    `db:"recipient" json:"recipient"`
	Template    string    `db:"template" json:"template"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// AlertHistory represents historical alert data stored in PostgreSQL
type AlertHistory struct {
	ID           string        `db:"id" json:"id"`
	AlertRuleID  string        `db:"alert_rule_id" json:"alert_rule_id"`
	Name         string        `db:"name" json:"name"`
	Description  string        `db:"description" json:"description"`
	Severity     AlertSeverity `db:"severity" json:"severity"`
	Status       AlertStatus   `db:"status" json:"status"`
	Symbol       string        `db:"symbol" json:"symbol"`
	Condition    string        `db:"condition" json:"condition"`
	Threshold    float64       `db:"threshold" json:"threshold"`
	CurrentValue float64       `db:"current_value" json:"current_value"`
	TriggeredAt  *time.Time    `db:"triggered_at" json:"triggered_at"`
	ResolvedAt   *time.Time    `db:"resolved_at" json:"resolved_at"`
	Labels       JSONMap       `db:"labels" json:"labels"`
	Metadata     JSONMap       `db:"metadata" json:"metadata"`
	CreatedAt    time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `db:"updated_at" json:"updated_at"`
}

// NotificationHistory represents historical notification data
type NotificationHistory struct {
	ID        string     `db:"id" json:"id"`
	AlertID   string     `db:"alert_id" json:"alert_id"`
	Type      string     `db:"type" json:"type"`
	Recipient string     `db:"recipient" json:"recipient"`
	Message   string     `db:"message" json:"message"`
	Status    string     `db:"status" json:"status"`
	SentAt    *time.Time `db:"sent_at" json:"sent_at"`
	Error     string     `db:"error" json:"error"`
	Metadata  JSONMap    `db:"metadata" json:"metadata"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// SystemConfiguration represents system-wide configuration stored in PostgreSQL
type SystemConfiguration struct {
	ID          string    `db:"id" json:"id"`
	Key         string    `db:"key" json:"key"`
	Value       string    `db:"value" json:"value"`
	Description string    `db:"description" json:"description"`
	Category    string    `db:"category" json:"category"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Database Query Models

// MetricQuery represents a query for metrics from TimescaleDB
type MetricQuery struct {
	Symbol     string            `json:"symbol"`
	Type       MetricType        `json:"type"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Aggregate  bool              `json:"aggregate,omitempty"`
	TimeBucket string            `json:"time_bucket,omitempty"` // e.g., "1 hour", "1 day"
}

// AlertRuleQuery represents a query for alert rules from PostgreSQL
type AlertRuleQuery struct {
	ID       string        `json:"id,omitempty"`
	Symbol   string        `json:"symbol,omitempty"`
	Severity AlertSeverity `json:"severity,omitempty"`
	Enabled  *bool         `json:"enabled,omitempty"`
	Limit    int           `json:"limit,omitempty"`
	Offset   int           `json:"offset,omitempty"`
}

// AlertHistoryQuery represents a query for alert history from PostgreSQL
type AlertHistoryQuery struct {
	AlertRuleID string        `json:"alert_rule_id,omitempty"`
	Symbol      string        `json:"symbol,omitempty"`
	Severity    AlertSeverity `json:"severity,omitempty"`
	Status      AlertStatus   `json:"status,omitempty"`
	StartTime   time.Time     `json:"start_time,omitempty"`
	EndTime     time.Time     `json:"end_time,omitempty"`
	Limit       int           `json:"limit,omitempty"`
	Offset      int           `json:"offset,omitempty"`
}

// Database Statistics Models

// MetricStats represents statistics about metrics in TimescaleDB
type MetricStats struct {
	Symbol       string     `db:"symbol" json:"symbol"`
	Type         MetricType `db:"type" json:"type"`
	Count        int64      `db:"count" json:"count"`
	OldestRecord time.Time  `db:"oldest_record" json:"oldest_record"`
	NewestRecord time.Time  `db:"newest_record" json:"newest_record"`
	MinValue     float64    `db:"min_value" json:"min_value"`
	MaxValue     float64    `db:"max_value" json:"max_value"`
	AvgValue     float64    `db:"avg_value" json:"avg_value"`
	TotalValue   float64    `db:"total_value" json:"total_value"`
}

// AlertStats represents statistics about alerts in PostgreSQL
type AlertStats struct {
	TotalAlerts    int64 `db:"total_alerts" json:"total_alerts"`
	ActiveAlerts   int64 `db:"active_alerts" json:"active_alerts"`
	ResolvedAlerts int64 `db:"resolved_alerts" json:"resolved_alerts"`
	SilencedAlerts int64 `db:"silenced_alerts" json:"silenced_alerts"`
	InfoAlerts     int64 `db:"info_alerts" json:"info_alerts"`
	WarningAlerts  int64 `db:"warning_alerts" json:"warning_alerts"`
	CriticalAlerts int64 `db:"critical_alerts" json:"critical_alerts"`
}
