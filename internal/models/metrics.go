package models

import (
	"fmt"
	"time"
)

// AnomalyType represents the type of anomaly detected
type AnomalyType string

const (
	AnomalyTypePriceSpike    AnomalyType = "price_spike"
	AnomalyTypePriceDrop     AnomalyType = "price_drop"
	AnomalyTypeVolumeSpike   AnomalyType = "volume_spike"
	AnomalyTypeVolumeDrop    AnomalyType = "volume_drop"
	AnomalyTypeVolatility    AnomalyType = "volatility"
	AnomalyTypeTrendReversal AnomalyType = "trend_reversal"
	AnomalyTypeOutlier       AnomalyType = "outlier"
)

// AnomalySeverity represents the severity level of an anomaly
type AnomalySeverity string

const (
	AnomalySeverityLow      AnomalySeverity = "low"
	AnomalySeverityMedium   AnomalySeverity = "medium"
	AnomalySeverityHigh     AnomalySeverity = "high"
	AnomalySeverityCritical AnomalySeverity = "critical"
)

// AnomalyDetectionResult represents the result of anomaly detection
type AnomalyDetectionResult struct {
	Symbol        string                 `json:"symbol"`
	Type          AnomalyType            `json:"type"`
	Severity      AnomalySeverity        `json:"severity"`
	CurrentValue  float64                `json:"current_value"`
	ExpectedValue float64                `json:"expected_value"`
	Deviation     float64                `json:"deviation"`
	Confidence    float64                `json:"confidence"` // 0.0 to 1.0
	Timestamp     time.Time              `json:"timestamp"`
	Window        time.Duration          `json:"window"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates the AnomalyDetectionResult
func (a *AnomalyDetectionResult) Validate() error {
	if a.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if a.Type == "" {
		return fmt.Errorf("anomaly type cannot be empty")
	}
	if a.Severity == "" {
		return fmt.Errorf("anomaly severity cannot be empty")
	}
	if a.Confidence < 0 || a.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if a.Window <= 0 {
		return fmt.Errorf("window duration must be positive")
	}
	return nil
}

// GetRedisKey returns the Redis key for storing this anomaly
func (a *AnomalyDetectionResult) GetRedisKey() string {
	return fmt.Sprintf("anomalies:%s:%s", a.Symbol, a.Type)
}

// GetRedisScore returns the Redis score (timestamp) for sorted set operations
func (a *AnomalyDetectionResult) GetRedisScore() float64 {
	return float64(a.Timestamp.Unix())
}

// IsSignificant returns true if the anomaly is significant enough to trigger alerts
func (a *AnomalyDetectionResult) IsSignificant() bool {
	return a.Severity == AnomalySeverityHigh || a.Severity == AnomalySeverityCritical
}

// MetricRollup represents aggregated metrics over different time windows
type MetricRollup struct {
	Symbol       string            `json:"symbol"`
	Type         MetricType        `json:"type"`
	Window       time.Duration     `json:"window"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Count        int64             `json:"count"`
	Sum          float64           `json:"sum"`
	Min          float64           `json:"min"`
	Max          float64           `json:"max"`
	Average      float64           `json:"average"`
	Median       float64           `json:"median"`
	StdDev       float64           `json:"std_dev"`
	Percentile95 float64           `json:"percentile_95"`
	Percentile99 float64           `json:"percentile_99"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// Validate validates the MetricRollup
func (m *MetricRollup) Validate() error {
	if m.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if m.Type == "" {
		return fmt.Errorf("metric type cannot be empty")
	}
	if m.Window <= 0 {
		return fmt.Errorf("window duration must be positive")
	}
	if m.Count < 0 {
		return fmt.Errorf("count cannot be negative")
	}
	if m.StartTime.After(m.EndTime) {
		return fmt.Errorf("start time cannot be after end time")
	}
	return nil
}

// GetRedisKey returns the Redis key for storing this metric rollup
func (m *MetricRollup) GetRedisKey() string {
	return fmt.Sprintf("rollups:%s:%s:%s", m.Symbol, m.Type, m.Window.String())
}

// GetRedisScore returns the Redis score (start time) for sorted set operations
func (m *MetricRollup) GetRedisScore() float64 {
	return float64(m.StartTime.Unix())
}

// GetRange returns the range (max - min) of the metric
func (m *MetricRollup) GetRange() float64 {
	return m.Max - m.Min
}

// GetCoefficientOfVariation returns the coefficient of variation (std dev / mean)
func (m *MetricRollup) GetCoefficientOfVariation() float64 {
	if m.Average == 0 {
		return 0
	}
	return m.StdDev / m.Average
}

// AlertThreshold represents a threshold for triggering alerts
type AlertThreshold struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Symbol     string            `json:"symbol"`
	MetricType MetricType        `json:"metric_type"`
	Condition  string            `json:"condition"` // gt, lt, gte, lte, eq, ne
	Threshold  float64           `json:"threshold"`
	Window     time.Duration     `json:"window"`
	Severity   AlertSeverity     `json:"severity"`
	Enabled    bool              `json:"enabled"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// Validate validates the AlertThreshold
func (t *AlertThreshold) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("threshold ID cannot be empty")
	}
	if t.Name == "" {
		return fmt.Errorf("threshold name cannot be empty")
	}
	if t.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if t.MetricType == "" {
		return fmt.Errorf("metric type cannot be empty")
	}
	if t.Condition == "" {
		return fmt.Errorf("condition cannot be empty")
	}
	if t.Window <= 0 {
		return fmt.Errorf("window duration must be positive")
	}
	if t.Severity == "" {
		return fmt.Errorf("severity cannot be empty")
	}
	return nil
}

// GetRedisKey returns the Redis key for storing this threshold
func (t *AlertThreshold) GetRedisKey() string {
	return fmt.Sprintf("thresholds:%s", t.ID)
}

// Evaluate evaluates the threshold against a given value
func (t *AlertThreshold) Evaluate(value float64) bool {
	if !t.Enabled {
		return false
	}

	switch t.Condition {
	case "gt":
		return value > t.Threshold
	case "gte":
		return value >= t.Threshold
	case "lt":
		return value < t.Threshold
	case "lte":
		return value <= t.Threshold
	case "eq":
		return value == t.Threshold
	case "ne":
		return value != t.Threshold
	default:
		return false
	}
}

// MovingAverageConfig represents configuration for moving average calculations
type MovingAverageConfig struct {
	Window  time.Duration `json:"window"`
	Method  string        `json:"method"`            // simple, exponential, weighted
	Alpha   float64       `json:"alpha,omitempty"`   // For exponential moving average
	Weights []float64     `json:"weights,omitempty"` // For weighted moving average
}

// Validate validates the MovingAverageConfig
func (m *MovingAverageConfig) Validate() error {
	if m.Window <= 0 {
		return fmt.Errorf("window duration must be positive")
	}
	if m.Method == "" {
		return fmt.Errorf("method cannot be empty")
	}
	if m.Method == "exponential" && (m.Alpha <= 0 || m.Alpha > 1) {
		return fmt.Errorf("alpha must be between 0 and 1 for exponential moving average")
	}
	if m.Method == "weighted" && len(m.Weights) == 0 {
		return fmt.Errorf("weights cannot be empty for weighted moving average")
	}
	return nil
}

// AnomalyDetectionConfig represents configuration for anomaly detection
type AnomalyDetectionConfig struct {
	Method          string        `json:"method"` // zscore, iqr, isolation_forest, statistical
	Window          time.Duration `json:"window"`
	Threshold       float64       `json:"threshold"`
	MinDataPoints   int           `json:"min_data_points"`
	OutlierFraction float64       `json:"outlier_fraction,omitempty"`
	ConfidenceLevel float64       `json:"confidence_level,omitempty"`
}

// Validate validates the AnomalyDetectionConfig
func (a *AnomalyDetectionConfig) Validate() error {
	if a.Method == "" {
		return fmt.Errorf("method cannot be empty")
	}
	if a.Window <= 0 {
		return fmt.Errorf("window duration must be positive")
	}
	if a.Threshold <= 0 {
		return fmt.Errorf("threshold must be positive")
	}
	if a.MinDataPoints <= 0 {
		return fmt.Errorf("min data points must be positive")
	}
	if a.OutlierFraction < 0 || a.OutlierFraction > 1 {
		return fmt.Errorf("outlier fraction must be between 0 and 1")
	}
	if a.ConfidenceLevel < 0 || a.ConfidenceLevel > 1 {
		return fmt.Errorf("confidence level must be between 0 and 1")
	}
	return nil
}
