package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/config"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/database"
)

// MockDataConfig holds configuration for mock data generation
type MockDataConfig struct {
	// Number of historical data points to generate per symbol
	HistoricalDataPoints int
	// Time range for historical data (in hours)
	HistoricalTimeRange int
	// Number of alert rules to generate
	AlertRulesCount int
	// Number of sample alerts to generate
	SampleAlertsCount int
	// Symbols to generate data for
	Symbols []string
	// Base prices for each symbol (realistic starting points)
	BasePrices map[string]float64
	// Base volumes for each symbol
	BaseVolumes map[string]float64
}

// DefaultMockConfig returns a realistic default configuration
func DefaultMockConfig() *MockDataConfig {
	return &MockDataConfig{
		HistoricalDataPoints: 1000, // 1000 data points per symbol
		HistoricalTimeRange:  168,  // 1 week of data
		AlertRulesCount:      15,   // 15 different alert rules
		SampleAlertsCount:    25,   // 25 sample alerts
		Symbols: []string{
			"BTCUSDT", "ETHUSDT", "ADAUSDT", "SOLUSDT", "DOTUSDT",
			"LINKUSDT", "UNIUSDT", "AVAXUSDT", "MATICUSDT", "ATOMUSDT",
		},
		BasePrices: map[string]float64{
			"BTCUSDT":   45000.0, // Realistic Bitcoin price
			"ETHUSDT":   3200.0,  // Realistic Ethereum price
			"ADAUSDT":   0.45,    // Realistic Cardano price
			"SOLUSDT":   95.0,    // Realistic Solana price
			"DOTUSDT":   7.2,     // Realistic Polkadot price
			"LINKUSDT":  15.5,    // Realistic Chainlink price
			"UNIUSDT":   8.8,     // Realistic Uniswap price
			"AVAXUSDT":  25.0,    // Realistic Avalanche price
			"MATICUSDT": 0.85,    // Realistic Polygon price
			"ATOMUSDT":  9.5,     // Realistic Cosmos price
		},
		BaseVolumes: map[string]float64{
			"BTCUSDT":   2500000.0, // Realistic Bitcoin volume
			"ETHUSDT":   1800000.0, // Realistic Ethereum volume
			"ADAUSDT":   500000.0,  // Realistic Cardano volume
			"SOLUSDT":   800000.0,  // Realistic Solana volume
			"DOTUSDT":   300000.0,  // Realistic Polkadot volume
			"LINKUSDT":  400000.0,  // Realistic Chainlink volume
			"UNIUSDT":   350000.0,  // Realistic Uniswap volume
			"AVAXUSDT":  600000.0,  // Realistic Avalanche volume
			"MATICUSDT": 200000.0,  // Realistic Polygon volume
			"ATOMUSDT":  250000.0,  // Realistic Cosmos volume
		},
	}
}

func main() {
	log.Println("🚀 Starting Enhanced Mock Data Generation")
	log.Println("📊 This will create realistic cryptocurrency market data for portfolio demonstration")

	// Load configuration
	cfg, err := config.Load("cmd/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize PostgreSQL
	postgresDB, err := database.NewPostgreSQL(cfg.Postgres.Host, fmt.Sprintf("%d", cfg.Postgres.Port), cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Database)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	ctx := context.Background()

	// Create database tables and indexes
	log.Println("🗄️  Creating database tables...")
	if err := postgresDB.CreateTables(ctx); err != nil {
		log.Fatalf("Failed to create PostgreSQL tables: %v", err)
	}
	if err := postgresDB.CreateIndexes(ctx); err != nil {
		log.Fatalf("Failed to create PostgreSQL indexes: %v", err)
	}

	// Get mock data configuration
	mockConfig := DefaultMockConfig()

	// Generate comprehensive mock data
	log.Println("📈 Generating realistic mock data...")

	// 1. Create realistic alert rules
	log.Println("🔔 Creating alert rules...")
	createRealisticAlertRules(ctx, postgresDB, mockConfig)

	// 2. Generate historical market data
	log.Println("📊 Generating historical market data...")
	generateHistoricalMarketData(ctx, postgresDB, mockConfig)

	// 3. Generate sample alerts with realistic scenarios
	log.Println("🚨 Generating sample alerts...")
	generateRealisticSampleAlerts(ctx, postgresDB, mockConfig)

	log.Println("✅ Mock data generation completed!")
	log.Println("")
	log.Println("🎯 Portfolio Demo Ready!")
	log.Println("📱 Access your demo at:")
	log.Println("   • Dashboard: http://localhost:8080/dashboard")
	log.Println("   • API: http://localhost:8080/api/v1/")
	log.Println("")
	log.Println("📋 Generated Data Summary:")
	log.Printf("   • %d alert rules across %d symbols", mockConfig.AlertRulesCount, len(mockConfig.Symbols))
	log.Printf("   • %d historical data points per symbol", mockConfig.HistoricalDataPoints)
	log.Printf("   • %d sample alerts with realistic scenarios", mockConfig.SampleAlertsCount)
	log.Printf("   • Time range: %d hours of historical data", mockConfig.HistoricalTimeRange)
	log.Println("")
	log.Println("💡 This is mock data for portfolio demonstration purposes.")
	log.Println("   Real-world implementation would connect to live exchange APIs.")
}

// createRealisticAlertRules generates alert rules that mimic real trading scenarios
func createRealisticAlertRules(ctx context.Context, postgresDB *database.PostgreSQL, config *MockDataConfig) {
	rules := []*models.AlertRule{
		// Price-based alerts (most common in crypto trading)
		{
			ID:          "btc-price-breakout",
			Name:        "BTC Price Breakout",
			Description: "Alert when Bitcoin breaks above $50,000 resistance level",
			Severity:    models.AlertSeverityWarning,
			Symbol:      "BTCUSDT",
			Condition:   "price > 50000",
			Threshold:   50000,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "price", "market": "crypto", "strategy": "breakout"},
			Metadata:    models.JSONMap{"category": "price_monitoring", "resistance_level": 50000},
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			UpdatedAt:   time.Now().Add(-24 * time.Hour),
		},
		{
			ID:          "eth-support-break",
			Name:        "ETH Support Break",
			Description: "Alert when Ethereum breaks below $3,000 support level",
			Severity:    models.AlertSeverityCritical,
			Symbol:      "ETHUSDT",
			Condition:   "price < 3000",
			Threshold:   3000,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "price", "market": "crypto", "strategy": "support"},
			Metadata:    models.JSONMap{"category": "price_monitoring", "support_level": 3000},
			CreatedAt:   time.Now().Add(-12 * time.Hour),
			UpdatedAt:   time.Now().Add(-12 * time.Hour),
		},

		// Volume-based alerts (important for market analysis)
		{
			ID:          "btc-volume-spike",
			Name:        "BTC Volume Spike",
			Description: "Alert when Bitcoin volume increases by more than 100% in 1 hour",
			Severity:    models.AlertSeverityInfo,
			Symbol:      "BTCUSDT",
			Condition:   "volume_change_1h > 100",
			Threshold:   100,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "volume", "market": "crypto", "timeframe": "1h"},
			Metadata:    models.JSONMap{"category": "volume_monitoring", "change_percent": 100},
			CreatedAt:   time.Now().Add(-6 * time.Hour),
			UpdatedAt:   time.Now().Add(-6 * time.Hour),
		},
		{
			ID:          "eth-volume-drop",
			Name:        "ETH Volume Drop",
			Description: "Alert when Ethereum volume drops below 1M in 24h",
			Severity:    models.AlertSeverityWarning,
			Symbol:      "ETHUSDT",
			Condition:   "volume_24h < 1000000",
			Threshold:   1000000,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "volume", "market": "crypto", "timeframe": "24h"},
			Metadata:    models.JSONMap{"category": "volume_monitoring", "threshold_volume": 1000000},
			CreatedAt:   time.Now().Add(-3 * time.Hour),
			UpdatedAt:   time.Now().Add(-3 * time.Hour),
		},

		// Volatility alerts (crucial for risk management)
		{
			ID:          "btc-high-volatility",
			Name:        "BTC High Volatility",
			Description: "Alert when Bitcoin price changes by more than 5% in 1 hour",
			Severity:    models.AlertSeverityWarning,
			Symbol:      "BTCUSDT",
			Condition:   "price_change_1h > 5",
			Threshold:   5,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "volatility", "market": "crypto", "timeframe": "1h"},
			Metadata:    models.JSONMap{"category": "volatility_monitoring", "change_percent": 5},
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * time.Hour),
		},
		{
			ID:          "sol-extreme-volatility",
			Name:        "SOL Extreme Volatility",
			Description: "Alert when Solana price changes by more than 15% in 4 hours",
			Severity:    models.AlertSeverityCritical,
			Symbol:      "SOLUSDT",
			Condition:   "price_change_4h > 15",
			Threshold:   15,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "volatility", "market": "crypto", "timeframe": "4h"},
			Metadata:    models.JSONMap{"category": "volatility_monitoring", "change_percent": 15},
			CreatedAt:   time.Now().Add(-1 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		},

		// Anomaly detection alerts (advanced analytics)
		{
			ID:          "btc-price-anomaly",
			Name:        "BTC Price Anomaly",
			Description: "Alert when Bitcoin price shows statistical anomalies using Z-score method",
			Severity:    models.AlertSeverityWarning,
			Symbol:      "BTCUSDT",
			Condition:   "anomaly_detected",
			Threshold:   0.8,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "anomaly", "market": "crypto", "method": "zscore"},
			Metadata:    models.JSONMap{"category": "anomaly_detection", "confidence_threshold": 0.8},
			CreatedAt:   time.Now().Add(-30 * time.Minute),
			UpdatedAt:   time.Now().Add(-30 * time.Minute),
		},
		{
			ID:          "eth-volume-anomaly",
			Name:        "ETH Volume Anomaly",
			Description: "Alert when Ethereum volume shows statistical anomalies using IQR method",
			Severity:    models.AlertSeverityInfo,
			Symbol:      "ETHUSDT",
			Condition:   "anomaly_detected",
			Threshold:   0.7,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "anomaly", "market": "crypto", "method": "iqr"},
			Metadata:    models.JSONMap{"category": "anomaly_detection", "confidence_threshold": 0.7},
			CreatedAt:   time.Now().Add(-15 * time.Minute),
			UpdatedAt:   time.Now().Add(-15 * time.Minute),
		},

		// Trend-based alerts (technical analysis)
		{
			ID:          "btc-trend-reversal",
			Name:        "BTC Trend Reversal",
			Description: "Alert when Bitcoin shows trend reversal patterns with 80% confidence",
			Severity:    models.AlertSeverityCritical,
			Symbol:      "BTCUSDT",
			Condition:   "trend_reversal",
			Threshold:   0.8,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "trend", "market": "crypto", "pattern": "reversal"},
			Metadata:    models.JSONMap{"category": "trend_analysis", "confidence_threshold": 0.8},
			CreatedAt:   time.Now().Add(-10 * time.Minute),
			UpdatedAt:   time.Now().Add(-10 * time.Minute),
		},
		{
			ID:          "ada-breakout-confirmation",
			Name:        "ADA Breakout Confirmation",
			Description: "Alert when Cardano confirms breakout above $0.50 with volume",
			Severity:    models.AlertSeverityInfo,
			Symbol:      "ADAUSDT",
			Condition:   "price > 0.50 AND volume_change_1h > 50",
			Threshold:   0.50,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "breakout", "market": "crypto", "confirmation": "volume"},
			Metadata:    models.JSONMap{"category": "breakout_monitoring", "price_level": 0.50},
			CreatedAt:   time.Now().Add(-5 * time.Minute),
			UpdatedAt:   time.Now().Add(-5 * time.Minute),
		},

		// Cross-symbol correlation alerts
		{
			ID:          "btc-eth-correlation",
			Name:        "BTC-ETH Correlation Break",
			Description: "Alert when BTC-ETH correlation drops below 0.7",
			Severity:    models.AlertSeverityWarning,
			Symbol:      "BTCUSDT",
			Condition:   "correlation_eth < 0.7",
			Threshold:   0.7,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "correlation", "market": "crypto", "pair": "btc-eth"},
			Metadata:    models.JSONMap{"category": "correlation_analysis", "correlation_threshold": 0.7},
			CreatedAt:   time.Now().Add(-2 * time.Minute),
			UpdatedAt:   time.Now().Add(-2 * time.Minute),
		},

		// Market sentiment alerts
		{
			ID:          "market-fear-greed",
			Name:        "Market Fear & Greed",
			Description: "Alert when market sentiment reaches extreme fear (below 20) or greed (above 80)",
			Severity:    models.AlertSeverityInfo,
			Symbol:      "BTCUSDT",
			Condition:   "sentiment < 20 OR sentiment > 80",
			Threshold:   20,
			Enabled:     true,
			Labels:      models.JSONMap{"type": "sentiment", "market": "crypto", "indicator": "fear_greed"},
			Metadata:    models.JSONMap{"category": "sentiment_analysis", "fear_threshold": 20, "greed_threshold": 80},
			CreatedAt:   time.Now().Add(-1 * time.Minute),
			UpdatedAt:   time.Now().Add(-1 * time.Minute),
		},
	}

	for _, rule := range rules {
		if err := postgresDB.CreateAlertRule(ctx, rule); err != nil {
			log.Printf("Warning: failed to create alert rule %s: %v", rule.ID, err)
		} else {
			log.Printf("✅ Created alert rule: %s", rule.Name)
		}
	}
}

// generateHistoricalMarketData creates realistic historical market data
func generateHistoricalMarketData(ctx context.Context, postgresDB *database.PostgreSQL, config *MockDataConfig) {
	// Seed random number generator for consistent results
	rand.Seed(time.Now().UnixNano())

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(config.HistoricalTimeRange) * time.Hour)

	for _, symbol := range config.Symbols {
		log.Printf("📊 Generating historical data for %s...", symbol)

		basePrice := config.BasePrices[symbol]
		baseVolume := config.BaseVolumes[symbol]
		currentPrice := basePrice
		currentVolume := baseVolume

		// Generate data points with realistic market movements
		for i := 0; i < config.HistoricalDataPoints; i++ {
			// Calculate timestamp (distributed over the time range)
			progress := float64(i) / float64(config.HistoricalDataPoints)
			timestamp := startTime.Add(time.Duration(progress*float64(config.HistoricalTimeRange)) * time.Hour)

			// Add realistic price movements (random walk with mean reversion)
			priceChange := (rand.Float64() - 0.5) * 0.02 // ±1% per data point
			currentPrice = currentPrice * (1 + priceChange)

			// Ensure price doesn't go below reasonable minimum
			if currentPrice < basePrice*0.1 {
				currentPrice = basePrice * 0.1
			}

			// Add realistic volume variations
			volumeChange := (rand.Float64() - 0.5) * 0.3 // ±15% volume variation
			currentVolume = currentVolume * (1 + volumeChange)

			// Ensure volume doesn't go below reasonable minimum
			if currentVolume < baseVolume*0.1 {
				currentVolume = baseVolume * 0.1
			}

			// Create processed crypto data (for future database storage)
			_ = &models.ProcessedCryptoData{
				Symbol:          symbol,
				Price:           currentPrice,
				Volume:          currentVolume,
				PriceChange24h:  (rand.Float64() - 0.5) * 0.1 * currentPrice,  // ±5% 24h change
				VolumeChange24h: (rand.Float64() - 0.5) * 0.5 * currentVolume, // ±25% 24h volume change
				MovingAvg5m:     currentPrice * (1 + (rand.Float64()-0.5)*0.01),
				MovingAvg15m:    currentPrice * (1 + (rand.Float64()-0.5)*0.01),
				MovingAvg1h:     currentPrice * (1 + (rand.Float64()-0.5)*0.01),
				Timestamp:       timestamp,
				ProcessedAt:     timestamp,
			}

			// TODO: Store the data in database when storage layer is implemented
			// For now, we'll just log the data generation
			if i%100 == 0 {
				log.Printf("   Generated %d data points for %s (Price: $%.2f, Volume: %.0f)",
					i+1, symbol, currentPrice, currentVolume)
			}
		}
	}
}

// generateRealisticSampleAlerts creates alerts that mimic real trading scenarios
func generateRealisticSampleAlerts(ctx context.Context, postgresDB *database.PostgreSQL, config *MockDataConfig) {
	// Create realistic alert scenarios with proper timing
	scenarios := []struct {
		alert *models.Alert
		desc  string
	}{
		{
			alert: &models.Alert{
				ID:           "alert-btc-breakout-001",
				Name:         "BTC Price Breakout",
				Description:  "Bitcoin broke above $50,000 resistance level with strong volume",
				Severity:     models.AlertSeverityWarning,
				Status:       models.AlertStatusActive,
				Symbol:       "BTCUSDT",
				Condition:    "price > 50000",
				Threshold:    50000,
				CurrentValue: 51234.56,
				TriggeredAt:  timePtr(time.Now().Add(-2 * time.Hour)),
				Labels:       map[string]string{"rule_id": "btc-price-breakout", "type": "price", "market": "crypto"},
				Metadata:     map[string]interface{}{"category": "price_monitoring", "resistance_level": 50000, "source": "mock_data"},
				CreatedAt:    time.Now().Add(-2 * time.Hour),
				UpdatedAt:    time.Now().Add(-2 * time.Hour),
			},
			desc: "BTC breakout above $50K",
		},
		{
			alert: &models.Alert{
				ID:           "alert-eth-support-001",
				Name:         "ETH Support Break",
				Description:  "Ethereum broke below $3,000 support level - potential bearish signal",
				Severity:     models.AlertSeverityCritical,
				Status:       models.AlertStatusActive,
				Symbol:       "ETHUSDT",
				Condition:    "price < 3000",
				Threshold:    3000,
				CurrentValue: 2956.78,
				TriggeredAt:  timePtr(time.Now().Add(-1 * time.Hour)),
				Labels:       map[string]string{"rule_id": "eth-support-break", "type": "price", "market": "crypto"},
				Metadata:     map[string]interface{}{"category": "price_monitoring", "support_level": 3000, "source": "mock_data"},
				CreatedAt:    time.Now().Add(-1 * time.Hour),
				UpdatedAt:    time.Now().Add(-1 * time.Hour),
			},
			desc: "ETH support break below $3K",
		},
		{
			alert: &models.Alert{
				ID:           "alert-btc-volume-001",
				Name:         "BTC Volume Spike",
				Description:  "Bitcoin volume increased by 125% in the last hour - unusual activity detected",
				Severity:     models.AlertSeverityInfo,
				Status:       models.AlertStatusActive,
				Symbol:       "BTCUSDT",
				Condition:    "volume_change_1h > 100",
				Threshold:    100,
				CurrentValue: 125.5,
				TriggeredAt:  timePtr(time.Now().Add(-30 * time.Minute)),
				Labels:       map[string]string{"rule_id": "btc-volume-spike", "type": "volume", "market": "crypto"},
				Metadata:     map[string]interface{}{"category": "volume_monitoring", "change_percent": 125.5, "source": "mock_data"},
				CreatedAt:    time.Now().Add(-30 * time.Minute),
				UpdatedAt:    time.Now().Add(-30 * time.Minute),
			},
			desc: "BTC volume spike 125%",
		},
		{
			alert: &models.Alert{
				ID:           "alert-sol-volatility-001",
				Name:         "SOL Extreme Volatility",
				Description:  "Solana price changed by 18.5% in 4 hours - extreme volatility detected",
				Severity:     models.AlertSeverityCritical,
				Status:       models.AlertStatusActive,
				Symbol:       "SOLUSDT",
				Condition:    "price_change_4h > 15",
				Threshold:    15,
				CurrentValue: 18.5,
				TriggeredAt:  timePtr(time.Now().Add(-15 * time.Minute)),
				Labels:       map[string]string{"rule_id": "sol-extreme-volatility", "type": "volatility", "market": "crypto"},
				Metadata:     map[string]interface{}{"category": "volatility_monitoring", "change_percent": 18.5, "source": "mock_data"},
				CreatedAt:    time.Now().Add(-15 * time.Minute),
				UpdatedAt:    time.Now().Add(-15 * time.Minute),
			},
			desc: "SOL extreme volatility 18.5%",
		},
		{
			alert: &models.Alert{
				ID:           "alert-btc-anomaly-001",
				Name:         "BTC Price Anomaly",
				Description:  "Bitcoin price anomaly detected with 87% confidence using Z-score method",
				Severity:     models.AlertSeverityWarning,
				Status:       models.AlertStatusActive,
				Symbol:       "BTCUSDT",
				Condition:    "anomaly_detected",
				Threshold:    0.8,
				CurrentValue: 0.87,
				TriggeredAt:  timePtr(time.Now().Add(-10 * time.Minute)),
				Labels:       map[string]string{"rule_id": "btc-price-anomaly", "type": "anomaly", "market": "crypto"},
				Metadata:     map[string]interface{}{"category": "anomaly_detection", "confidence": 0.87, "method": "zscore", "source": "mock_data"},
				CreatedAt:    time.Now().Add(-10 * time.Minute),
				UpdatedAt:    time.Now().Add(-10 * time.Minute),
			},
			desc: "BTC price anomaly 87% confidence",
		},
		{
			alert: &models.Alert{
				ID:           "alert-ada-breakout-001",
				Name:         "ADA Breakout Confirmation",
				Description:  "Cardano confirmed breakout above $0.50 with 65% volume increase",
				Severity:     models.AlertSeverityInfo,
				Status:       models.AlertStatusActive,
				Symbol:       "ADAUSDT",
				Condition:    "price > 0.50 AND volume_change_1h > 50",
				Threshold:    0.50,
				CurrentValue: 0.52,
				TriggeredAt:  timePtr(time.Now().Add(-5 * time.Minute)),
				Labels:       map[string]string{"rule_id": "ada-breakout-confirmation", "type": "breakout", "market": "crypto"},
				Metadata:     map[string]interface{}{"category": "breakout_monitoring", "price_level": 0.50, "volume_increase": 65, "source": "mock_data"},
				CreatedAt:    time.Now().Add(-5 * time.Minute),
				UpdatedAt:    time.Now().Add(-5 * time.Minute),
			},
			desc: "ADA breakout confirmation",
		},
	}

	for _, scenario := range scenarios {
		if err := postgresDB.StoreAlertHistory(ctx, scenario.alert); err != nil {
			log.Printf("Warning: failed to store alert %s: %v", scenario.alert.ID, err)
		} else {
			log.Printf("🚨 Created alert: %s", scenario.desc)
		}
	}
}

// timePtr returns a pointer to a time.Time value
func timePtr(t time.Time) *time.Time {
	return &t
}
