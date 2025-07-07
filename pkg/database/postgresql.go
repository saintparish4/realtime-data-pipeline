package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
)

// PostgreSQL implements AlertStorage interface for PostgreSQL
type PostgreSQL struct {
	db *sqlx.DB
}

// NewPostgreSQL creates a new PostgreSQL connection
func NewPostgreSQL(host, port, user, password, database string) (*PostgreSQL, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)

	// Add retry logic for connection
	var db *sqlx.DB
	var err error

	for i := 0; i < 5; i++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			// Test the connection
			if err := db.Ping(); err == nil {
				return &PostgreSQL{db: db}, nil
			}
			db.Close()
		}

		// Wait before retrying
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to PostgreSQL after retries: %w", err)
}

// Alert Rules Management

// CreateAlertRule creates a new alert rule
func (p *PostgreSQL) CreateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	query := `
		INSERT INTO alert_rules (id, name, description, severity, symbol, condition, threshold, enabled, labels, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO NOTHING
	`

	now := time.Now()
	_, err := p.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.Severity,
		rule.Symbol,
		rule.Condition,
		rule.Threshold,
		rule.Enabled,
		rule.Labels,
		rule.Metadata,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}

	return nil
}

// GetAlertRule retrieves an alert rule by ID
func (p *PostgreSQL) GetAlertRule(ctx context.Context, id string) (*models.AlertRule, error) {
	query := "SELECT * FROM alert_rules WHERE id = $1"

	var rule models.AlertRule
	err := p.db.GetContext(ctx, &rule, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	return &rule, nil
}

// UpdateAlertRule updates an existing alert rule
func (p *PostgreSQL) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	query := `
		UPDATE alert_rules 
		SET name = $2, description = $3, severity = $4, symbol = $5, condition = $6, 
		    threshold = $7, enabled = $8, labels = $9, metadata = $10, updated_at = $11
		WHERE id = $1
	`

	result, err := p.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.Severity,
		rule.Symbol,
		rule.Condition,
		rule.Threshold,
		rule.Enabled,
		rule.Labels,
		rule.Metadata,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update alert rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert rule not found: %s", rule.ID)
	}

	return nil
}

// DeleteAlertRule deletes an alert rule by ID
func (p *PostgreSQL) DeleteAlertRule(ctx context.Context, id string) error {
	query := "DELETE FROM alert_rules WHERE id = $1"

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert rule not found: %s", id)
	}

	return nil
}

// ListAlertRules retrieves alert rules based on query parameters
func (p *PostgreSQL) ListAlertRules(ctx context.Context, query *models.AlertRuleQuery) ([]*models.AlertRule, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE clause
	if query.ID != "" {
		conditions = append(conditions, fmt.Sprintf("id = $%d", argIndex))
		args = append(args, query.ID)
		argIndex++
	}

	if query.Symbol != "" {
		conditions = append(conditions, fmt.Sprintf("symbol = $%d", argIndex))
		args = append(args, query.Symbol)
		argIndex++
	}

	if query.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argIndex))
		args = append(args, query.Severity)
		argIndex++
	}

	if query.Enabled != nil {
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", argIndex))
		args = append(args, *query.Enabled)
		argIndex++
	}

	// Build the query
	sqlQuery := "SELECT * FROM alert_rules"
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " ORDER BY created_at DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
		argIndex++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, query.Offset)
	}

	var rules []*models.AlertRule
	err := p.db.SelectContext(ctx, &rules, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert rules: %w", err)
	}

	return rules, nil
}

// Alert History Management

// StoreAlertHistory stores alert history
func (p *PostgreSQL) StoreAlertHistory(ctx context.Context, alert *models.Alert) error {
	query := `
		INSERT INTO alert_history (id, alert_rule_id, name, description, severity, status, symbol, condition, threshold, current_value, triggered_at, resolved_at, labels, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO NOTHING
	`

	// Convert map[string]string to JSONMap for labels
	labelsJSON := models.JSONMap{}
	for k, v := range alert.Labels {
		labelsJSON[k] = v
	}

	// Convert map[string]interface{} to JSONMap for metadata
	metadataJSON := models.JSONMap{}
	for k, v := range alert.Metadata {
		metadataJSON[k] = v
	}

	now := time.Now()
	_, err := p.db.ExecContext(ctx, query,
		alert.ID,
		alert.Labels["rule_id"], // Extract rule_id from labels
		alert.Name,
		alert.Description,
		alert.Severity,
		alert.Status,
		alert.Symbol,
		alert.Condition,
		alert.Threshold,
		alert.CurrentValue,
		alert.TriggeredAt,
		alert.ResolvedAt,
		labelsJSON,
		metadataJSON,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to store alert history: %w", err)
	}

	return nil
}

// GetAlertHistory retrieves alert history by ID
func (p *PostgreSQL) GetAlertHistory(ctx context.Context, id string) (*models.AlertHistory, error) {
	query := "SELECT * FROM alert_history WHERE id = $1"

	var history models.AlertHistory
	err := p.db.GetContext(ctx, &history, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert history not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert history: %w", err)
	}

	return &history, nil
}

// ListAlertHistory retrieves alert history based on query parameters
func (p *PostgreSQL) ListAlertHistory(ctx context.Context, query *models.AlertHistoryQuery) ([]*models.AlertHistory, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE clause
	if query.AlertRuleID != "" {
		conditions = append(conditions, fmt.Sprintf("alert_rule_id = $%d", argIndex))
		args = append(args, query.AlertRuleID)
		argIndex++
	}

	if query.Symbol != "" {
		conditions = append(conditions, fmt.Sprintf("symbol = $%d", argIndex))
		args = append(args, query.Symbol)
		argIndex++
	}

	if query.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argIndex))
		args = append(args, query.Severity)
		argIndex++
	}

	if query.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, query.Status)
		argIndex++
	}

	if !query.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, query.StartTime)
		argIndex++
	}

	if !query.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, query.EndTime)
		argIndex++
	}

	// Build the query
	sqlQuery := "SELECT * FROM alert_history"
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " ORDER BY created_at DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
		argIndex++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, query.Offset)
	}

	var history []*models.AlertHistory
	err := p.db.SelectContext(ctx, &history, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert history: %w", err)
	}

	return history, nil
}

// Notification Management

// CreateNotification creates a new notification configuration
func (p *PostgreSQL) CreateNotification(ctx context.Context, notification *models.AlertRuleNotification) error {
	query := `
		INSERT INTO alert_rule_notifications (id, alert_rule_id, type, recipient, template, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	_, err := p.db.ExecContext(ctx, query,
		notification.ID,
		notification.AlertRuleID,
		notification.Type,
		notification.Recipient,
		notification.Template,
		notification.Enabled,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// GetNotification retrieves a notification by ID
func (p *PostgreSQL) GetNotification(ctx context.Context, id string) (*models.AlertRuleNotification, error) {
	query := "SELECT * FROM alert_rule_notifications WHERE id = $1"

	var notification models.AlertRuleNotification
	err := p.db.GetContext(ctx, &notification, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

// UpdateNotification updates an existing notification
func (p *PostgreSQL) UpdateNotification(ctx context.Context, notification *models.AlertRuleNotification) error {
	query := `
		UPDATE alert_rule_notifications 
		SET type = $2, recipient = $3, template = $4, enabled = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := p.db.ExecContext(ctx, query,
		notification.ID,
		notification.Type,
		notification.Recipient,
		notification.Template,
		notification.Enabled,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", notification.ID)
	}

	return nil
}

// DeleteNotification deletes a notification by ID
func (p *PostgreSQL) DeleteNotification(ctx context.Context, id string) error {
	query := "DELETE FROM alert_rule_notifications WHERE id = $1"

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", id)
	}

	return nil
}

// ListNotifications retrieves notifications for an alert rule
func (p *PostgreSQL) ListNotifications(ctx context.Context, alertRuleID string) ([]*models.AlertRuleNotification, error) {
	query := "SELECT * FROM alert_rule_notifications WHERE alert_rule_id = $1 ORDER BY created_at DESC"

	var notifications []*models.AlertRuleNotification
	err := p.db.SelectContext(ctx, &notifications, query, alertRuleID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	return notifications, nil
}

// Notification History

// StoreNotificationHistory stores notification history
func (p *PostgreSQL) StoreNotificationHistory(ctx context.Context, notification *models.NotificationHistory) error {
	query := `
		INSERT INTO notification_history (id, alert_id, type, recipient, message, status, sent_at, error, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := p.db.ExecContext(ctx, query,
		notification.ID,
		notification.AlertID,
		notification.Type,
		notification.Recipient,
		notification.Message,
		notification.Status,
		notification.SentAt,
		notification.Error,
		notification.Metadata,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to store notification history: %w", err)
	}

	return nil
}

// ListNotificationHistory retrieves notification history for an alert
func (p *PostgreSQL) ListNotificationHistory(ctx context.Context, alertID string) ([]*models.NotificationHistory, error) {
	query := "SELECT * FROM notification_history WHERE alert_id = $1 ORDER BY created_at DESC"

	var history []*models.NotificationHistory
	err := p.db.SelectContext(ctx, &history, query, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification history: %w", err)
	}

	return history, nil
}

// System Configuration

// SetConfiguration sets a system configuration
func (p *PostgreSQL) SetConfiguration(ctx context.Context, config *models.SystemConfiguration) error {
	query := `
		INSERT INTO system_configurations (id, key, value, description, category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err := p.db.ExecContext(ctx, query,
		config.ID,
		config.Key,
		config.Value,
		config.Description,
		config.Category,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to set configuration: %w", err)
	}

	return nil
}

// GetConfiguration retrieves a system configuration by key
func (p *PostgreSQL) GetConfiguration(ctx context.Context, key string) (*models.SystemConfiguration, error) {
	query := "SELECT * FROM system_configurations WHERE key = $1"

	var config models.SystemConfiguration
	err := p.db.GetContext(ctx, &config, query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	return &config, nil
}

// ListConfigurations retrieves configurations by category
func (p *PostgreSQL) ListConfigurations(ctx context.Context, category string) ([]*models.SystemConfiguration, error) {
	query := "SELECT * FROM system_configurations WHERE category = $1 ORDER BY key"

	var configs []*models.SystemConfiguration
	err := p.db.SelectContext(ctx, &configs, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to list configurations: %w", err)
	}

	return configs, nil
}

// DeleteConfiguration deletes a system configuration by key
func (p *PostgreSQL) DeleteConfiguration(ctx context.Context, key string) error {
	query := "DELETE FROM system_configurations WHERE key = $1"

	result, err := p.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("configuration not found: %s", key)
	}

	return nil
}

// Statistics

// GetAlertStats returns alert statistics
func (p *PostgreSQL) GetAlertStats(ctx context.Context) (*models.AlertStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_alerts,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_alerts,
			COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved_alerts,
			COUNT(CASE WHEN status = 'silenced' THEN 1 END) as silenced_alerts,
			COUNT(CASE WHEN severity = 'info' THEN 1 END) as info_alerts,
			COUNT(CASE WHEN severity = 'warning' THEN 1 END) as warning_alerts,
			COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical_alerts
		FROM alert_history
	`

	var stats models.AlertStats
	err := p.db.GetContext(ctx, &stats, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats: %w", err)
	}

	return &stats, nil
}

// Database Management

// CreateTables creates all necessary tables
func (p *PostgreSQL) CreateTables(ctx context.Context) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS alert_rules (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			severity VARCHAR(20) NOT NULL,
			symbol VARCHAR(50),
			condition TEXT NOT NULL,
			threshold DOUBLE PRECISION NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT true,
			labels JSONB,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS alert_history (
			id VARCHAR(100) PRIMARY KEY,
			alert_rule_id VARCHAR(100) NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			severity VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL,
			symbol VARCHAR(50),
			condition TEXT NOT NULL,
			threshold DOUBLE PRECISION NOT NULL,
			current_value DOUBLE PRECISION NOT NULL,
			triggered_at TIMESTAMPTZ,
			resolved_at TIMESTAMPTZ,
			labels JSONB,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS alert_rule_notifications (
			id VARCHAR(100) PRIMARY KEY,
			alert_rule_id VARCHAR(100) NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			recipient VARCHAR(255) NOT NULL,
			template TEXT,
			enabled BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS notification_history (
			id VARCHAR(100) PRIMARY KEY,
			alert_id VARCHAR(100) NOT NULL,
			type VARCHAR(50) NOT NULL,
			recipient VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			status VARCHAR(20) NOT NULL,
			sent_at TIMESTAMPTZ,
			error TEXT,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS system_configurations (
			id VARCHAR(100) PRIMARY KEY,
			key VARCHAR(255) UNIQUE NOT NULL,
			value TEXT NOT NULL,
			description TEXT,
			category VARCHAR(100),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}

	for _, tableQuery := range tables {
		_, err := p.db.ExecContext(ctx, tableQuery)
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// CreateIndexes creates necessary indexes
func (p *PostgreSQL) CreateIndexes(ctx context.Context) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_alert_rules_symbol ON alert_rules (symbol)",
		"CREATE INDEX IF NOT EXISTS idx_alert_rules_severity ON alert_rules (severity)",
		"CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules (enabled)",
		"CREATE INDEX IF NOT EXISTS idx_alert_history_alert_rule_id ON alert_history (alert_rule_id)",
		"CREATE INDEX IF NOT EXISTS idx_alert_history_symbol ON alert_history (symbol)",
		"CREATE INDEX IF NOT EXISTS idx_alert_history_status ON alert_history (status)",
		"CREATE INDEX IF NOT EXISTS idx_alert_history_created_at ON alert_history (created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alert_rule_notifications_alert_rule_id ON alert_rule_notifications (alert_rule_id)",
		"CREATE INDEX IF NOT EXISTS idx_notification_history_alert_id ON notification_history (alert_id)",
		"CREATE INDEX IF NOT EXISTS idx_notification_history_created_at ON notification_history (created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_system_configurations_key ON system_configurations (key)",
		"CREATE INDEX IF NOT EXISTS idx_system_configurations_category ON system_configurations (category)",
	}

	for _, indexQuery := range indexes {
		_, err := p.db.ExecContext(ctx, indexQuery)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (p *PostgreSQL) Close() error {
	return p.db.Close()
}
