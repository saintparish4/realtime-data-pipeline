package observability

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/storage"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// Service handles observability operations including metrics collection,
// alert evaluation, and notification management.
type Service struct {
	redisClient *redis.Client
	storage     *storage.StorageManager
	alertRules  map[string]*AlertRule
	rulesMux    sync.RWMutex

	// Configuration
	metricsRetention      time.Duration
	alertCheckInterval    time.Duration
	notificationRateLimit time.Duration
}

// AlertRule defines a rule for triggering alerts
type AlertRule struct {
	ID          string
	Name        string
	Description string
	Severity    models.AlertSeverity
	Symbol      string
	Condition   string // e.g., "price > 50000", "volume_change > 10"
	Threshold   float64
	Enabled     bool
	CreatedAt   time.Time
}

// NewService creates a new observability service
func NewService(redisClient *redis.Client, storage *storage.StorageManager) *Service {
	return &Service{
		redisClient:           redisClient,
		storage:               storage,
		alertRules:            make(map[string]*AlertRule),
		metricsRetention:      30 * 24 * time.Hour, // 30 days
		alertCheckInterval:    30 * time.Second,
		notificationRateLimit: 5 * time.Minute,
	}
}

// Initialize loads alert rules from database and starts background processes
func (s *Service) Initialize(ctx context.Context) error {
	// Only initialize database features if storage is available
	if s.storage != nil {
		// Load alert rules from database
		if err := s.loadAlertRules(ctx); err != nil {
			return fmt.Errorf("failed to load alert rules: %w", err)
		}

		// Start background processes
		go s.startAlertRuleSync(ctx)
		go s.StartCleanupRoutine(ctx)
	}

	return nil
}

// loadAlertRules loads all enabled alert rules from the database
func (s *Service) loadAlertRules(ctx context.Context) error {
	query := &models.AlertRuleQuery{
		Enabled: &[]bool{true}[0], // Create a pointer to true
		Limit:   1000,             // Load up to 1000 rules
	}

	rules, err := s.storage.Alerts.ListAlertRules(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to list alert rules: %w", err)
	}

	s.rulesMux.Lock()
	defer s.rulesMux.Unlock()

	// Clear existing rules and load from database
	s.alertRules = make(map[string]*AlertRule)
	for _, rule := range rules {
		s.alertRules[rule.ID] = &AlertRule{
			ID:          rule.ID,
			Name:        rule.Name,
			Description: rule.Description,
			Severity:    rule.Severity,
			Symbol:      rule.Symbol,
			Condition:   rule.Condition,
			Threshold:   rule.Threshold,
			Enabled:     rule.Enabled,
			CreatedAt:   rule.CreatedAt,
		}
	}

	log.Printf("Loaded %d alert rules from database", len(s.alertRules))
	return nil
}

// startAlertRuleSync periodically syncs alert rules from database
func (s *Service) startAlertRuleSync(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Sync every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.loadAlertRules(ctx); err != nil {
				log.Printf("Failed to sync alert rules: %v", err)
			}
		}
	}
}

// RecordMetric records a time-series metric
func (s *Service) RecordMetric(ctx context.Context, metric *models.TimeSeriesMetric) error {
	if err := metric.Validate(); err != nil {
		return fmt.Errorf("invalid metric: %w", err)
	}

	// Store in TimescaleDB for persistent storage if available
	if s.storage != nil {
		if err := s.storage.Metrics.StoreMetric(ctx, metric); err != nil {
			log.Printf("Warning: failed to store metric in database: %v", err)
		}
	}

	// Store in Redis for real-time access
	if err := s.redisClient.StoreTimeSeriesMetric(ctx, metric); err != nil {
		return fmt.Errorf("failed to store metric in Redis: %w", err)
	}

	// Evaluate alert rules for this metric
	go s.evaluateAlertRules(ctx, metric)

	return nil
}

// AddAlertRule adds a new alert rule
func (s *Service) AddAlertRule(rule *AlertRule) error {
	if rule.ID == "" {
		return fmt.Errorf("alert rule ID cannot be empty")
	}
	if rule.Name == "" {
		return fmt.Errorf("alert rule name cannot be empty")
	}
	if rule.Condition == "" {
		return fmt.Errorf("alert rule condition cannot be empty")
	}

	// Create database model
	dbRule := &models.AlertRule{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Severity:    rule.Severity,
		Symbol:      rule.Symbol,
		Condition:   rule.Condition,
		Threshold:   rule.Threshold,
		Enabled:     rule.Enabled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store in database
	if err := s.storage.Alerts.CreateAlertRule(context.Background(), dbRule); err != nil {
		return fmt.Errorf("failed to create alert rule in database: %w", err)
	}

	// Add to memory cache
	s.rulesMux.Lock()
	defer s.rulesMux.Unlock()

	rule.CreatedAt = time.Now()
	s.alertRules[rule.ID] = rule

	return nil
}

// RemoveAlertRule removes an alert rule
func (s *Service) RemoveAlertRule(ruleID string) error {
	// Remove from database
	if err := s.storage.Alerts.DeleteAlertRule(context.Background(), ruleID); err != nil {
		return fmt.Errorf("failed to delete alert rule from database: %w", err)
	}

	// Remove from memory cache
	s.rulesMux.Lock()
	defer s.rulesMux.Unlock()

	delete(s.alertRules, ruleID)
	return nil
}

// GetAlertRules returns all alert rules
func (s *Service) GetAlertRules() []*AlertRule {
	s.rulesMux.RLock()
	defer s.rulesMux.RUnlock()

	var rules []*AlertRule
	for _, rule := range s.alertRules {
		rules = append(rules, rule)
	}
	return rules
}

// evaluateAlertRules evaluates all alert rules against a metric
func (s *Service) evaluateAlertRules(ctx context.Context, metric *models.TimeSeriesMetric) {
	s.rulesMux.RLock()
	rules := make([]*AlertRule, 0, len(s.alertRules))
	for _, rule := range s.alertRules {
		if rule.Enabled && (rule.Symbol == "" || rule.Symbol == metric.Symbol) {
			rules = append(rules, rule)
		}
	}
	s.rulesMux.RUnlock()

	for _, rule := range rules {
		if s.shouldTriggerAlert(rule, metric) {
			s.triggerAlert(ctx, rule, metric)
		}
	}
}

// shouldTriggerAlert determines if an alert should be triggered based on the rule and metric
func (s *Service) shouldTriggerAlert(rule *AlertRule, metric *models.TimeSeriesMetric) bool {
	// Simple threshold-based evaluation
	// In a real implementation, you might want to use a more sophisticated expression evaluator
	switch rule.Condition {
	case "price > threshold":
		return metric.Type == models.MetricTypePrice && metric.Value > rule.Threshold
	case "price < threshold":
		return metric.Type == models.MetricTypePrice && metric.Value < rule.Threshold
	case "volume > threshold":
		return metric.Type == models.MetricTypeVolume && metric.Value > rule.Threshold
	case "volume < threshold":
		return metric.Type == models.MetricTypeVolume && metric.Value < rule.Threshold
	default:
		// For custom conditions, you might want to implement a more sophisticated evaluator
		return false
	}
}

// triggerAlert creates and stores an alert
func (s *Service) triggerAlert(ctx context.Context, rule *AlertRule, metric *models.TimeSeriesMetric) {
	// Check rate limiting for notifications
	rateLimitKey := fmt.Sprintf("alert:%s:%s", rule.ID, metric.Symbol)
	exceeded, err := s.redisClient.CheckAndIncrementRateLimit(ctx, rateLimitKey, 1, s.notificationRateLimit)
	if err != nil {
		log.Printf("Failed to check rate limit for alert %s: %v", rule.ID, err)
		return
	}
	if exceeded {
		log.Printf("Rate limit exceeded for alert %s, skipping notification", rule.ID)
		return
	}

	// Create alert
	alert := &models.Alert{
		ID:           fmt.Sprintf("%s_%s_%d", rule.ID, metric.Symbol, time.Now().Unix()),
		Name:         rule.Name,
		Description:  rule.Description,
		Severity:     rule.Severity,
		Status:       models.AlertStatusActive,
		Symbol:       metric.Symbol,
		Condition:    rule.Condition,
		Threshold:    rule.Threshold,
		CurrentValue: metric.Value,
		Labels: map[string]string{
			"rule_id":     rule.ID,
			"metric_type": string(metric.Type),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	alert.Trigger(metric.Value)

	// Store the alert in database
	if err := s.storage.Alerts.StoreAlertHistory(ctx, alert); err != nil {
		log.Printf("Failed to store alert in database: %v", err)
	}

	// Also store in Redis for real-time access
	if err := s.redisClient.StoreAlert(ctx, alert); err != nil {
		log.Printf("Warning: failed to store alert in Redis: %v", err)
	}

	// Send notification
	go s.sendNotification(ctx, alert)
}

// sendNotification sends a notification for an alert
func (s *Service) sendNotification(ctx context.Context, alert *models.Alert) {
	// Create notification
	notification := &models.AlertNotification{
		ID:        fmt.Sprintf("notif_%s_%d", alert.ID, time.Now().Unix()),
		AlertID:   alert.ID,
		Type:      "system", // You might want to make this configurable
		Recipient: "admin",  // You might want to make this configurable
		Message: fmt.Sprintf("Alert triggered: %s - %s (Value: %.2f, Threshold: %.2f)",
			alert.Name, alert.Description, alert.CurrentValue, alert.Threshold),
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Store notification in database
	if err := s.storage.Alerts.StoreNotificationHistory(ctx, &models.NotificationHistory{
		ID:        notification.ID,
		AlertID:   notification.AlertID,
		Type:      notification.Type,
		Recipient: notification.Recipient,
		Message:   notification.Message,
		Status:    notification.Status,
		CreatedAt: notification.CreatedAt,
	}); err != nil {
		log.Printf("Failed to store notification in database: %v", err)
	}

	// Also store in Redis for real-time access
	if err := s.redisClient.StoreAlertNotification(ctx, notification); err != nil {
		log.Printf("Warning: failed to store notification in Redis: %v", err)
	}

	// In a real implementation, you would send the notification here
	// For now, we'll just log it
	log.Printf("Notification sent: %s", notification.Message)

	// Update notification status
	now := time.Now()
	notification.Status = "sent"
	notification.SentAt = &now

	// Update in database
	if err := s.storage.Alerts.StoreNotificationHistory(ctx, &models.NotificationHistory{
		ID:        notification.ID,
		AlertID:   notification.AlertID,
		Type:      notification.Type,
		Recipient: notification.Recipient,
		Message:   notification.Message,
		Status:    notification.Status,
		SentAt:    notification.SentAt,
		CreatedAt: notification.CreatedAt,
	}); err != nil {
		log.Printf("Failed to update notification status in database: %v", err)
	}

	// Update in Redis
	if err := s.redisClient.StoreAlertNotification(ctx, notification); err != nil {
		log.Printf("Warning: failed to update notification status in Redis: %v", err)
	}
}

// GetMetrics retrieves metrics for a given query
func (s *Service) GetMetrics(ctx context.Context, query *models.TimeSeriesQuery) ([]*models.TimeSeriesMetric, error) {
	// Convert to database query
	dbQuery := &models.MetricQuery{
		Symbol:    query.Symbol,
		Type:      query.Type,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		Limit:     query.Limit,
		Labels:    query.Labels,
	}

	// Query from database
	records, err := s.storage.Metrics.QueryMetrics(ctx, dbQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics from database: %w", err)
	}

	// Convert to TimeSeriesMetric format
	var metrics []*models.TimeSeriesMetric
	for _, record := range records {
		// Convert JSONMap to map[string]string
		labels := make(map[string]string)
		for k, v := range record.Labels {
			if str, ok := v.(string); ok {
				labels[k] = str
			}
		}

		metric := &models.TimeSeriesMetric{
			Symbol:    record.Symbol,
			Type:      record.Type,
			Value:     record.Value,
			Timestamp: record.Timestamp,
			Labels:    labels,
			Metadata:  map[string]interface{}(record.Metadata),
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetAggregatedMetrics retrieves aggregated metrics
func (s *Service) GetAggregatedMetrics(ctx context.Context, query *models.MetricQuery) ([]*models.MetricAggregation, error) {
	dbQuery := &models.MetricQuery{
		Symbol:     query.Symbol,
		Type:       query.Type,
		StartTime:  query.StartTime,
		EndTime:    query.EndTime,
		Limit:      query.Limit,
		Labels:     query.Labels,
		Aggregate:  true,
		TimeBucket: query.TimeBucket,
	}

	return s.storage.Metrics.QueryAggregatedMetrics(ctx, dbQuery)
}

// GetActiveAlerts retrieves all active alerts
func (s *Service) GetActiveAlerts(ctx context.Context) ([]*models.Alert, error) {
	// Try Redis first for real-time access
	redisAlerts, err := s.redisClient.GetActiveAlerts(ctx)
	if err == nil && len(redisAlerts) > 0 {
		log.Printf("GetActiveAlerts: Found %d alerts in Redis", len(redisAlerts))
		return redisAlerts, nil
	}

	// Fallback to database
	query := &models.AlertHistoryQuery{
		Status: models.AlertStatusActive,
		Limit:  100,
	}

	history, err := s.storage.Alerts.ListAlertHistory(ctx, query)
	if err != nil {
		log.Printf("GetActiveAlerts: Database error: %v", err)
		return nil, fmt.Errorf("failed to get active alerts from database: %w", err)
	}

	log.Printf("GetActiveAlerts: Found %d alerts in database", len(history))

	// Convert to Alert format
	var dbAlerts []*models.Alert
	for _, record := range history {
		// Convert JSONMap to map[string]string
		labels := make(map[string]string)
		for k, v := range record.Labels {
			if str, ok := v.(string); ok {
				labels[k] = str
			}
		}

		alert := &models.Alert{
			ID:           record.ID,
			Name:         record.Name,
			Description:  record.Description,
			Severity:     record.Severity,
			Status:       record.Status,
			Symbol:       record.Symbol,
			Condition:    record.Condition,
			Threshold:    record.Threshold,
			CurrentValue: record.CurrentValue,
			TriggeredAt:  record.TriggeredAt,
			ResolvedAt:   record.ResolvedAt,
			Labels:       labels,
			Metadata:     map[string]interface{}(record.Metadata),
			CreatedAt:    record.CreatedAt,
			UpdatedAt:    record.UpdatedAt,
		}
		dbAlerts = append(dbAlerts, alert)
	}

	// Ensure we always return an empty slice instead of nil
	if dbAlerts == nil {
		dbAlerts = []*models.Alert{}
	}

	log.Printf("GetActiveAlerts: Returning %d alerts", len(dbAlerts))
	return dbAlerts, nil
}

// ResolveAlert resolves an alert
func (s *Service) ResolveAlert(ctx context.Context, alertID string) error {
	// Update in Redis
	if err := s.redisClient.UpdateAlertStatus(ctx, alertID, models.AlertStatusResolved); err != nil {
		log.Printf("Warning: failed to update alert status in Redis: %v", err)
	}

	// Update in database
	// Note: This would require updating the alert_history table
	// For now, we'll just log that the alert was resolved
	log.Printf("Alert %s resolved", alertID)
	return nil
}

// SilenceAlert silences an alert
func (s *Service) SilenceAlert(ctx context.Context, alertID string) error {
	// Update in Redis
	if err := s.redisClient.UpdateAlertStatus(ctx, alertID, models.AlertStatusSilenced); err != nil {
		log.Printf("Warning: failed to update alert status in Redis: %v", err)
	}

	// Update in database
	// Note: This would require updating the alert_history table
	// For now, we'll just log that the alert was silenced
	log.Printf("Alert %s silenced", alertID)
	return nil
}

// GetMetricStats returns statistics for a metric
func (s *Service) GetMetricStats(ctx context.Context, symbol string, metricType models.MetricType) (map[string]interface{}, error) {
	// Try database first
	stats, err := s.storage.Metrics.GetMetricStats(ctx, symbol, metricType)
	if err == nil {
		return map[string]interface{}{
			"count":         stats.Count,
			"oldest_record": stats.OldestRecord,
			"newest_record": stats.NewestRecord,
			"min_value":     stats.MinValue,
			"max_value":     stats.MaxValue,
			"avg_value":     stats.AvgValue,
			"total_value":   stats.TotalValue,
			"symbol":        stats.Symbol,
			"type":          stats.Type,
		}, nil
	}

	// Fallback to Redis
	return s.redisClient.GetMetricStats(ctx, symbol, metricType)
}

// GetAlertStats returns alert statistics
func (s *Service) GetAlertStats(ctx context.Context) (*models.AlertStats, error) {
	return s.storage.Alerts.GetAlertStats(ctx)
}

// CleanupOldData removes old metrics and notifications
func (s *Service) CleanupOldData(ctx context.Context) error {
	// Cleanup old metrics from database
	if err := s.storage.Metrics.CleanupOldMetrics(ctx, s.metricsRetention); err != nil {
		return fmt.Errorf("failed to cleanup old metrics from database: %w", err)
	}

	// Cleanup old metrics from Redis
	if err := s.redisClient.CleanupOldMetrics(ctx, s.metricsRetention); err != nil {
		log.Printf("Warning: failed to cleanup old metrics from Redis: %v", err)
	}

	return nil
}

// StartCleanupRoutine starts a background routine to cleanup old data
func (s *Service) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Run cleanup daily
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CleanupOldData(ctx); err != nil {
				log.Printf("Failed to cleanup old data: %v", err)
			}
		}
	}
}

// RecordLatencyMetric records a latency metric for performance monitoring
func (s *Service) RecordLatencyMetric(ctx context.Context, operation string, duration time.Duration, labels map[string]string) error {
	metric := &models.TimeSeriesMetric{
		Symbol:    "system",
		Type:      models.MetricTypeLatency,
		Value:     float64(duration.Milliseconds()),
		Timestamp: time.Now(),
		Labels:    labels,
		Metadata: map[string]interface{}{
			"operation": operation,
		},
	}

	return s.RecordMetric(ctx, metric)
}

// RecordErrorMetric records an error metric
func (s *Service) RecordErrorMetric(ctx context.Context, errorType string, errorMessage string, labels map[string]string) error {
	metric := &models.TimeSeriesMetric{
		Symbol:    "system",
		Type:      models.MetricTypeError,
		Value:     1.0, // Count of errors
		Timestamp: time.Now(),
		Labels:    labels,
		Metadata: map[string]interface{}{
			"error_type":    errorType,
			"error_message": errorMessage,
		},
	}

	return s.RecordMetric(ctx, metric)
}

// GetSystemHealth returns the overall system health status
func (s *Service) GetSystemHealth(ctx context.Context) (map[string]interface{}, error) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"checks":    make(map[string]interface{}),
	}

	// Check Redis connectivity
	testKey := "health_check_test"
	if err := s.redisClient.Set(ctx, testKey, "test", time.Second); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]interface{})["redis"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(map[string]interface{})["redis"] = map[string]interface{}{
			"status": "ok",
		}
	}

	// Check database connectivity
	if _, err := s.storage.Alerts.GetAlertStats(ctx); err != nil {
		health["checks"].(map[string]interface{})["database"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(map[string]interface{})["database"] = map[string]interface{}{
			"status": "ok",
		}
	}

	// Check active alerts
	activeAlerts, err := s.GetActiveAlerts(ctx)
	if err != nil {
		health["checks"].(map[string]interface{})["alerts"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(map[string]interface{})["alerts"] = map[string]interface{}{
			"status":       "ok",
			"active_count": len(activeAlerts),
		}

		// If there are critical alerts, mark system as degraded
		for _, alert := range activeAlerts {
			if alert.Severity == models.AlertSeverityCritical {
				health["status"] = "degraded"
				break
			}
		}
	}

	return health, nil
}
