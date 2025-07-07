package storage

import (
	"context"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
)

// MetricsStorage defines the interface for TimescaleDB metrics storage
type MetricsStorage interface {
	// StoreMetric stores a single metric record
	StoreMetric(ctx context.Context, metric *models.TimeSeriesMetric) error

	// StoreMetrics stores multiple metric records in batch
	StoreMetrics(ctx context.Context, metrics []*models.TimeSeriesMetric) error

	// QueryMetrics retrieves metrics based on query parameters
	QueryMetrics(ctx context.Context, query *models.MetricQuery) ([]*models.MetricRecord, error)

	// QueryAggregatedMetrics retrieves aggregated metrics
	QueryAggregatedMetrics(ctx context.Context, query *models.MetricQuery) ([]*models.MetricAggregation, error)

	// GetMetricStats returns statistics for a specific metric
	GetMetricStats(ctx context.Context, symbol string, metricType models.MetricType) (*models.MetricStats, error)

	// CleanupOldMetrics removes metrics older than the specified duration
	CleanupOldMetrics(ctx context.Context, olderThan time.Duration) error

	// CreateHypertable creates a TimescaleDB hypertable for metrics
	CreateHypertable(ctx context.Context, tableName string) error

	// CreateIndexes creates necessary indexes for efficient querying
	CreateIndexes(ctx context.Context) error

	// Close closes the database connection
	Close() error
}

// AlertStorage defines the interface for PostgreSQL alert storage
type AlertStorage interface {
	// Alert Rules Management
	CreateAlertRule(ctx context.Context, rule *models.AlertRule) error
	GetAlertRule(ctx context.Context, id string) (*models.AlertRule, error)
	UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error
	DeleteAlertRule(ctx context.Context, id string) error
	ListAlertRules(ctx context.Context, query *models.AlertRuleQuery) ([]*models.AlertRule, error)

	// Alert History Management
	StoreAlertHistory(ctx context.Context, alert *models.Alert) error
	GetAlertHistory(ctx context.Context, id string) (*models.AlertHistory, error)
	ListAlertHistory(ctx context.Context, query *models.AlertHistoryQuery) ([]*models.AlertHistory, error)

	// Notification Management
	CreateNotification(ctx context.Context, notification *models.AlertRuleNotification) error
	GetNotification(ctx context.Context, id string) (*models.AlertRuleNotification, error)
	UpdateNotification(ctx context.Context, notification *models.AlertRuleNotification) error
	DeleteNotification(ctx context.Context, id string) error
	ListNotifications(ctx context.Context, alertRuleID string) ([]*models.AlertRuleNotification, error)

	// Notification History
	StoreNotificationHistory(ctx context.Context, notification *models.NotificationHistory) error
	ListNotificationHistory(ctx context.Context, alertID string) ([]*models.NotificationHistory, error)

	// System Configuration
	SetConfiguration(ctx context.Context, config *models.SystemConfiguration) error
	GetConfiguration(ctx context.Context, key string) (*models.SystemConfiguration, error)
	ListConfigurations(ctx context.Context, category string) ([]*models.SystemConfiguration, error)
	DeleteConfiguration(ctx context.Context, key string) error

	// Statistics
	GetAlertStats(ctx context.Context) (*models.AlertStats, error)

	// Database Management
	CreateTables(ctx context.Context) error
	CreateIndexes(ctx context.Context) error
	Close() error
}

// StorageManager combines both storage interfaces for unified access
type StorageManager struct {
	Metrics MetricsStorage
	Alerts  AlertStorage
}

// NewStorageManager creates a new storage manager with both TimescaleDB and PostgreSQL
func NewStorageManager(metrics MetricsStorage, alerts AlertStorage) *StorageManager {
	return &StorageManager{
		Metrics: metrics,
		Alerts:  alerts,
	}
}

// Close closes both storage connections
func (sm *StorageManager) Close() error {
	var errs []error

	if err := sm.Metrics.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := sm.Alerts.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0] // Return first error for simplicity
	}

	return nil
}

// InitializeStorage initializes both storage systems
func (sm *StorageManager) InitializeStorage(ctx context.Context) error {
	// Initialize TimescaleDB
	if err := sm.Metrics.CreateHypertable(ctx, "metrics"); err != nil {
		return err
	}

	if err := sm.Metrics.CreateIndexes(ctx); err != nil {
		return err
	}

	// Initialize PostgreSQL
	if err := sm.Alerts.CreateTables(ctx); err != nil {
		return err
	}

	if err := sm.Alerts.CreateIndexes(ctx); err != nil {
		return err
	}

	return nil
}
