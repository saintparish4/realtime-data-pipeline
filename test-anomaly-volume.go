package main

import (
	"context"
	"fmt"
	"log"

	"github.com/saintparish4/realtime-data-pipeline/internal/config"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/database"
)

func main() {
	fmt.Println("🧪 Testing Anomaly Detection and Trading Volume Alerts")
	fmt.Println("=====================================================")

	// Load configuration
	cfg, err := config.Load("cmd/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize PostgreSQL
	postgresDB, err := database.NewPostgreSQL(cfg.Postgres.Host, fmt.Sprintf("%d", cfg.Postgres.Port), cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Database)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	ctx := context.Background()

	// Test 1: Check if anomaly detection alert rules exist
	fmt.Println("\n📊 Test 1: Checking Anomaly Detection Alert Rules")
	testAnomalyDetectionRules(ctx, postgresDB)

	// Test 2: Check if trading volume alert rules exist
	fmt.Println("\n📈 Test 2: Checking Trading Volume Alert Rules")
	testTradingVolumeRules(ctx, postgresDB)

	// Test 3: Check if anomaly alerts are stored
	fmt.Println("\n🔍 Test 3: Checking Anomaly Detection Alerts")
	testAnomalyAlerts(ctx, postgresDB)

	// Test 4: Check if volume alerts are stored
	fmt.Println("\n📊 Test 4: Checking Trading Volume Alerts")
	testVolumeAlerts(ctx, postgresDB)

	fmt.Println("\n✅ All tests completed!")
}

func testAnomalyDetectionRules(ctx context.Context, postgresDB *database.PostgreSQL) {
	query := &models.AlertRuleQuery{
		Limit: 100,
	}

	rules, err := postgresDB.ListAlertRules(ctx, query)
	if err != nil {
		log.Printf("❌ Failed to list alert rules: %v", err)
		return
	}

	anomalyRules := []*models.AlertRule{}
	for _, rule := range rules {
		if rule.Labels != nil {
			if ruleType, ok := rule.Labels["type"]; ok && ruleType == "anomaly" {
				anomalyRules = append(anomalyRules, rule)
			}
		}
	}

	fmt.Printf("   Found %d anomaly detection rules:\n", len(anomalyRules))
	for _, rule := range anomalyRules {
		fmt.Printf("   - %s: %s (Threshold: %.2f)\n", rule.ID, rule.Name, rule.Threshold)
		if rule.Metadata != nil {
			if method, ok := rule.Metadata["method"]; ok {
				fmt.Printf("     Method: %v\n", method)
			}
		}
	}
}

func testTradingVolumeRules(ctx context.Context, postgresDB *database.PostgreSQL) {
	query := &models.AlertRuleQuery{
		Limit: 100,
	}

	rules, err := postgresDB.ListAlertRules(ctx, query)
	if err != nil {
		log.Printf("❌ Failed to list alert rules: %v", err)
		return
	}

	volumeRules := []*models.AlertRule{}
	for _, rule := range rules {
		if rule.Labels != nil {
			if ruleType, ok := rule.Labels["type"]; ok && ruleType == "volume" {
				volumeRules = append(volumeRules, rule)
			}
		}
		// Also check by condition
		if rule.Condition != "" && (rule.Condition == "volume < 200000" || rule.Condition == "volume_change > 50" || rule.Condition == "volume_change > 100") {
			volumeRules = append(volumeRules, rule)
		}
	}

	fmt.Printf("   Found %d trading volume rules:\n", len(volumeRules))
	for _, rule := range volumeRules {
		fmt.Printf("   - %s: %s (Threshold: %.2f)\n", rule.ID, rule.Name, rule.Threshold)
		fmt.Printf("     Condition: %s\n", rule.Condition)
	}
}

func testAnomalyAlerts(ctx context.Context, postgresDB *database.PostgreSQL) {
	query := &models.AlertHistoryQuery{
		Limit: 100,
	}

	alerts, err := postgresDB.ListAlertHistory(ctx, query)
	if err != nil {
		log.Printf("❌ Failed to list alert history: %v", err)
		return
	}

	anomalyAlerts := []*models.AlertHistory{}
	for _, alert := range alerts {
		if alert.Labels != nil {
			if alertType, ok := alert.Labels["type"]; ok && alertType == "anomaly" {
				anomalyAlerts = append(anomalyAlerts, alert)
			}
		}
		// Also check by condition
		if alert.Condition == "anomaly_detected" || alert.Condition == "trend_reversal" {
			anomalyAlerts = append(anomalyAlerts, alert)
		}
	}

	fmt.Printf("   Found %d anomaly detection alerts:\n", len(anomalyAlerts))
	for _, alert := range anomalyAlerts {
		fmt.Printf("   - %s: %s (Confidence: %.1f%%)\n", alert.ID, alert.Name, alert.CurrentValue*100)
		if alert.Metadata != nil {
			if method, ok := alert.Metadata["method"]; ok {
				fmt.Printf("     Method: %v\n", method)
			}
			if expected, ok := alert.Metadata["expected_value"]; ok {
				fmt.Printf("     Expected: %v\n", expected)
			}
			if deviation, ok := alert.Metadata["deviation"]; ok {
				fmt.Printf("     Deviation: %v\n", deviation)
			}
		}
		fmt.Printf("     Triggered: %s\n", alert.TriggeredAt.Format("2006-01-02 15:04:05"))
	}
}

func testVolumeAlerts(ctx context.Context, postgresDB *database.PostgreSQL) {
	query := &models.AlertHistoryQuery{
		Limit: 100,
	}

	alerts, err := postgresDB.ListAlertHistory(ctx, query)
	if err != nil {
		log.Printf("❌ Failed to list alert history: %v", err)
		return
	}

	volumeAlerts := []*models.AlertHistory{}
	for _, alert := range alerts {
		if alert.Labels != nil {
			if alertType, ok := alert.Labels["type"]; ok && alertType == "volume" {
				volumeAlerts = append(volumeAlerts, alert)
			}
		}
		// Also check by condition and name
		if alert.Condition != "" && (alert.Condition == "volume < 200000" || alert.Condition == "volume_change > 50" || alert.Condition == "volume_change > 100") {
			volumeAlerts = append(volumeAlerts, alert)
		}
	}

	fmt.Printf("   Found %d trading volume alerts:\n", len(volumeAlerts))
	for _, alert := range volumeAlerts {
		fmt.Printf("   - %s: %s (Value: %.2f)\n", alert.ID, alert.Name, alert.CurrentValue)
		fmt.Printf("     Condition: %s\n", alert.Condition)
		fmt.Printf("     Threshold: %.2f\n", alert.Threshold)
		fmt.Printf("     Triggered: %s\n", alert.TriggeredAt.Format("2006-01-02 15:04:05"))
	}
}
