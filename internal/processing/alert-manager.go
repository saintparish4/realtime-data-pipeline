package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// AlertManager handles alert threshold evaluation and alert management
type AlertManager struct {
	redisClient     *redis.Client
	alertThresholds []models.AlertThreshold
}

// NewAlertManager creates a new AlertManager instance
func NewAlertManager(redisClient *redis.Client, alertThresholds []models.AlertThreshold) *AlertManager {
	return &AlertManager{
		redisClient:     redisClient,
		alertThresholds: alertThresholds,
	}
}

// EvaluateThresholds evaluates all configured alert thresholds against processed data
func (am *AlertManager) EvaluateThresholds(ctx context.Context, processedData *models.ProcessedCryptoData) error {
	for _, threshold := range am.alertThresholds {
		if threshold.Symbol != processedData.Symbol {
			continue // Skip thresholds for different symbols
		}

		var value float64
		switch threshold.MetricType {
		case models.MetricTypePrice:
			value = processedData.Price
		case models.MetricTypeVolume:
			value = processedData.Volume
		default:
			// For derived metrics, calculate based on the metric type
			value = am.calculateDerivedMetric(processedData, threshold.MetricType)
		}

		if threshold.Evaluate(value) {
			if err := am.triggerAlert(ctx, &threshold, value, processedData.Timestamp); err != nil {
				log.Printf("Failed to trigger alert for threshold %s: %v", threshold.ID, err)
			}
		}
	}

	return nil
}

// EvaluateAnomalyThresholds evaluates thresholds specifically for anomalies
func (am *AlertManager) EvaluateAnomalyThresholds(ctx context.Context, anomaly *models.AnomalyDetectionResult) error {
	// Create a special threshold for anomaly severity
	anomalyThreshold := models.AlertThreshold{
		ID:         fmt.Sprintf("anomaly_%s_%s", anomaly.Symbol, anomaly.Type),
		Name:       fmt.Sprintf("Anomaly Alert - %s %s", anomaly.Symbol, anomaly.Type),
		Symbol:     anomaly.Symbol,
		MetricType: "anomaly_severity",
		Condition:  "gte",
		Threshold:  0.7, // Trigger for high confidence anomalies
		Window:     anomaly.Window,
		Severity:   models.AlertSeverityWarning,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if anomalyThreshold.Evaluate(anomaly.Confidence) {
		if err := am.triggerAnomalyAlert(ctx, &anomalyThreshold, anomaly); err != nil {
			log.Printf("Failed to trigger anomaly alert: %v", err)
		}
	}

	return nil
}

// triggerAlert creates and stores an alert when a threshold is triggered
func (am *AlertManager) triggerAlert(ctx context.Context, threshold *models.AlertThreshold, value float64, timestamp time.Time) error {
	alert := &models.Alert{
		ID:           fmt.Sprintf("alert_%s_%d", threshold.ID, timestamp.Unix()),
		Name:         threshold.Name,
		Description:  fmt.Sprintf("Threshold %s triggered for %s", threshold.Condition, threshold.Symbol),
		Severity:     threshold.Severity,
		Status:       models.AlertStatusActive,
		Symbol:       threshold.Symbol,
		Condition:    threshold.Condition,
		Threshold:    threshold.Threshold,
		CurrentValue: value,
		TriggeredAt:  &timestamp,
		Labels:       threshold.Labels,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := alert.Validate(); err != nil {
		return fmt.Errorf("invalid alert: %w", err)
	}

	return am.storeAlert(ctx, alert)
}

// triggerAnomalyAlert creates and stores an alert for anomalies
func (am *AlertManager) triggerAnomalyAlert(ctx context.Context, threshold *models.AlertThreshold, anomaly *models.AnomalyDetectionResult) error {
	alert := &models.Alert{
		ID:           fmt.Sprintf("anomaly_alert_%s_%d", anomaly.Symbol, anomaly.Timestamp.Unix()),
		Name:         threshold.Name,
		Description:  fmt.Sprintf("Anomaly detected: %s %s with confidence %.2f", anomaly.Symbol, anomaly.Type, anomaly.Confidence),
		Severity:     threshold.Severity,
		Status:       models.AlertStatusActive,
		Symbol:       anomaly.Symbol,
		Condition:    "anomaly_detected",
		Threshold:    threshold.Threshold,
		CurrentValue: anomaly.Confidence,
		TriggeredAt:  &anomaly.Timestamp,
		Labels: map[string]string{
			"anomaly_type": string(anomaly.Type),
			"severity":     string(anomaly.Severity),
		},
		Annotations: map[string]string{
			"expected_value": fmt.Sprintf("%.2f", anomaly.ExpectedValue),
			"deviation":      fmt.Sprintf("%.2f", anomaly.Deviation),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := alert.Validate(); err != nil {
		return fmt.Errorf("invalid anomaly alert: %w", err)
	}

	return am.storeAlert(ctx, alert)
}

// storeAlert stores an alert in Redis
func (am *AlertManager) storeAlert(ctx context.Context, alert *models.Alert) error {
	alertData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	key := alert.GetRedisKey()
	score := float64(alert.CreatedAt.Unix())

	if err := am.redisClient.ZAdd(ctx, key, score, string(alertData)); err != nil {
		return fmt.Errorf("failed to store alert in Redis: %w", err)
	}

	// Set expiration for the alert (e.g., 30 days)
	if err := am.redisClient.Expire(ctx, key, 30*24*time.Hour); err != nil {
		log.Printf("Failed to set expiration for alert key %s: %v", key, err)
	}

	// Also store in active alerts set
	activeKey := fmt.Sprintf("active_alerts:%s", alert.Symbol)
	if err := am.redisClient.ZAdd(ctx, activeKey, score, alert.ID); err != nil {
		log.Printf("Failed to add alert to active alerts set: %v", err)
	}

	return nil
}

// GetActiveAlerts retrieves active alerts for a symbol
func (am *AlertManager) GetActiveAlerts(ctx context.Context, symbol string) ([]*models.Alert, error) {
	// Use the existing GetActiveAlerts method from Redis client
	alerts, err := am.redisClient.GetActiveAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active alerts: %w", err)
	}

	// Filter by symbol
	var filteredAlerts []*models.Alert
	for _, alert := range alerts {
		if alert.Symbol == symbol && alert.IsActive() {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	return filteredAlerts, nil
}

// getAlert retrieves a specific alert by ID
func (am *AlertManager) getAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	return am.redisClient.GetAlert(ctx, alertID)
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(ctx context.Context, alertID string) error {
	alert, err := am.getAlert(ctx, alertID)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	alert.Resolve()

	// Update the alert in Redis
	alertData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	key := alert.GetRedisKey()
	if err := am.redisClient.Set(ctx, key, string(alertData), 30*24*time.Hour); err != nil {
		return fmt.Errorf("failed to update alert in Redis: %w", err)
	}

	// Remove from active alerts set using Redis client's DeleteAlert method
	if err := am.redisClient.DeleteAlert(ctx, alertID); err != nil {
		log.Printf("Failed to remove alert from active alerts set: %v", err)
	}

	return nil
}

// AddThreshold adds a new alert threshold
func (am *AlertManager) AddThreshold(threshold models.AlertThreshold) error {
	if err := threshold.Validate(); err != nil {
		return fmt.Errorf("invalid threshold: %w", err)
	}

	threshold.CreatedAt = time.Now()
	threshold.UpdatedAt = time.Now()

	am.alertThresholds = append(am.alertThresholds, threshold)
	return nil
}

// RemoveThreshold removes an alert threshold
func (am *AlertManager) RemoveThreshold(thresholdID string) {
	for i, threshold := range am.alertThresholds {
		if threshold.ID == thresholdID {
			am.alertThresholds = append(am.alertThresholds[:i], am.alertThresholds[i+1:]...)
			break
		}
	}
}

// GetThresholds returns all configured thresholds
func (am *AlertManager) GetThresholds() []models.AlertThreshold {
	return am.alertThresholds
}

// calculateDerivedMetric calculates derived metrics from processed data
func (am *AlertManager) calculateDerivedMetric(processedData *models.ProcessedCryptoData, metricType models.MetricType) float64 {
	switch metricType {
	case "price_change_24h":
		return processedData.PriceChange24h
	case "volume_change_24h":
		return processedData.VolumeChange24h
	case "price_change_percent":
		return processedData.GetPriceChangePercent()
	case "volume_change_percent":
		return processedData.GetVolumeChangePercent()
	case "moving_avg_5m":
		return processedData.MovingAvg5m
	case "moving_avg_15m":
		return processedData.MovingAvg15m
	case "moving_avg_1h":
		return processedData.MovingAvg1h
	default:
		return 0
	}
}
