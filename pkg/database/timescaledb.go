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

// TimescaleDB implements MetricsStorage interface for TimescaleDB
type TimescaleDB struct {
	db *sqlx.DB
}

// NewTimescaleDB creates a new TimescaleDB connection
func NewTimescaleDB(host, port, user, password, database string) (*TimescaleDB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to TimescaleDB: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping TimescaleDB: %w", err)
	}

	return &TimescaleDB{db: db}, nil
}

// StoreMetric stores a single metric record
func (t *TimescaleDB) StoreMetric(ctx context.Context, metric *models.TimeSeriesMetric) error {
	if err := metric.Validate(); err != nil {
		return fmt.Errorf("invalid metric: %w", err)
	}

	query := `
		INSERT INTO metrics (symbol, type, value, timestamp, labels, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := t.db.ExecContext(ctx, query,
		metric.Symbol,
		metric.Type,
		metric.Value,
		metric.Timestamp,
		metric.Labels,
		metric.Metadata,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}

	return nil
}

// StoreMetrics stores multiple metric records in batch
func (t *TimescaleDB) StoreMetrics(ctx context.Context, metrics []*models.TimeSeriesMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// Start a transaction for batch insert
	tx, err := t.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO metrics (symbol, type, value, timestamp, labels, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, metric := range metrics {
		if err := metric.Validate(); err != nil {
			return fmt.Errorf("invalid metric: %w", err)
		}

		_, err := stmt.ExecContext(ctx,
			metric.Symbol,
			metric.Type,
			metric.Value,
			metric.Timestamp,
			metric.Labels,
			metric.Metadata,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to store metric in batch: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// QueryMetrics retrieves metrics based on query parameters
func (t *TimescaleDB) QueryMetrics(ctx context.Context, query *models.MetricQuery) ([]*models.MetricRecord, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE clause
	if query.Symbol != "" {
		conditions = append(conditions, fmt.Sprintf("symbol = $%d", argIndex))
		args = append(args, query.Symbol)
		argIndex++
	}

	if query.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, query.Type)
		argIndex++
	}

	if !query.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, query.StartTime)
		argIndex++
	}

	if !query.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, query.EndTime)
		argIndex++
	}

	// Build the query
	sqlQuery := "SELECT * FROM metrics"
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " ORDER BY timestamp DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
		argIndex++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, query.Offset)
	}

	var records []*models.MetricRecord
	err := t.db.SelectContext(ctx, &records, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}

	return records, nil
}

// QueryAggregatedMetrics retrieves aggregated metrics using TimescaleDB time_bucket
func (t *TimescaleDB) QueryAggregatedMetrics(ctx context.Context, query *models.MetricQuery) ([]*models.MetricAggregation, error) {
	if query.TimeBucket == "" {
		return nil, fmt.Errorf("time_bucket is required for aggregated queries")
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE clause
	if query.Symbol != "" {
		conditions = append(conditions, fmt.Sprintf("symbol = $%d", argIndex))
		args = append(args, query.Symbol)
		argIndex++
	}

	if query.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, query.Type)
		argIndex++
	}

	if !query.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, query.StartTime)
		argIndex++
	}

	if !query.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, query.EndTime)
		argIndex++
	}

	// Build the aggregation query
	sqlQuery := fmt.Sprintf(`
		SELECT 
			symbol,
			type,
			time_bucket('%s', timestamp) as time_bucket,
			COUNT(*) as count,
			MIN(value) as min,
			MAX(value) as max,
			AVG(value) as avg,
			SUM(value) as sum
		FROM metrics
	`, query.TimeBucket)

	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " GROUP BY symbol, type, time_bucket ORDER BY time_bucket DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
		argIndex++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, query.Offset)
	}

	var aggregations []*models.MetricAggregation
	err := t.db.SelectContext(ctx, &aggregations, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated metrics: %w", err)
	}

	return aggregations, nil
}

// GetMetricStats returns statistics for a specific metric
func (t *TimescaleDB) GetMetricStats(ctx context.Context, symbol string, metricType models.MetricType) (*models.MetricStats, error) {
	query := `
		SELECT 
			symbol,
			type,
			COUNT(*) as count,
			MIN(timestamp) as oldest_record,
			MAX(timestamp) as newest_record,
			MIN(value) as min_value,
			MAX(value) as max_value,
			AVG(value) as avg_value,
			SUM(value) as total_value
		FROM metrics
		WHERE symbol = $1 AND type = $2
		GROUP BY symbol, type
	`

	var stats models.MetricStats
	err := t.db.GetContext(ctx, &stats, query, symbol, metricType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no metrics found for symbol %s and type %s", symbol, metricType)
		}
		return nil, fmt.Errorf("failed to get metric stats: %w", err)
	}

	return &stats, nil
}

// CleanupOldMetrics removes metrics older than the specified duration
func (t *TimescaleDB) CleanupOldMetrics(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	query := "DELETE FROM metrics WHERE timestamp < $1"

	result, err := t.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("Cleaned up %d old metric records\n", rowsAffected)
	return nil
}

// CreateHypertable creates a TimescaleDB hypertable for metrics
func (t *TimescaleDB) CreateHypertable(ctx context.Context, tableName string) error {
	// First, create the table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS metrics (
			id BIGSERIAL PRIMARY KEY,
			symbol VARCHAR(50) NOT NULL,
			type VARCHAR(50) NOT NULL,
			value DOUBLE PRECISION NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL,
			labels JSONB,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	_, err := t.db.ExecContext(ctx, createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create metrics table: %w", err)
	}

	// Convert to hypertable
	hypertableQuery := "SELECT create_hypertable('metrics', 'timestamp', if_not_exists => TRUE)"

	_, err = t.db.ExecContext(ctx, hypertableQuery)
	if err != nil {
		return fmt.Errorf("failed to create hypertable: %w", err)
	}

	return nil
}

// CreateIndexes creates necessary indexes for efficient querying
func (t *TimescaleDB) CreateIndexes(ctx context.Context) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_metrics_symbol_type ON metrics (symbol, type)",
		"CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics (timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_metrics_symbol_timestamp ON metrics (symbol, timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_metrics_type_timestamp ON metrics (type, timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_metrics_labels ON metrics USING GIN (labels)",
		"CREATE INDEX IF NOT EXISTS idx_metrics_metadata ON metrics USING GIN (metadata)",
	}

	for _, indexQuery := range indexes {
		_, err := t.db.ExecContext(ctx, indexQuery)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (t *TimescaleDB) Close() error {
	return t.db.Close()
}
