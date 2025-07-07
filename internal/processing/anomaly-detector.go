package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// AnomalyDetector implements various anomaly detection algorithms
type AnomalyDetector struct {
	redisClient *redis.Client
	config      models.AnomalyDetectionConfig
}

// NewAnomalyDetector creates a new AnomalyDetector instance
func NewAnomalyDetector(redisClient *redis.Client, config models.AnomalyDetectionConfig) *AnomalyDetector {
	return &AnomalyDetector{
		redisClient: redisClient,
		config:      config,
	}
}

// DetectPriceAnomaly detects anomalies in price data
func (ad *AnomalyDetector) DetectPriceAnomaly(ctx context.Context, rawData *models.RawCryptoData) (*models.AnomalyDetectionResult, error) {
	// Get historical price data
	historicalData, err := ad.getHistoricalData(ctx, rawData.Symbol, models.MetricTypePrice, ad.config.Window)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical price data: %w", err)
	}

	if len(historicalData) < ad.config.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points for anomaly detection: %d < %d", len(historicalData), ad.config.MinDataPoints)
	}

	// Detect anomaly based on configured method
	var anomaly *models.AnomalyDetectionResult
	switch ad.config.Method {
	case "zscore":
		anomaly = ad.detectZScoreAnomaly(rawData.Symbol, rawData.Price, historicalData, models.AnomalyTypePriceSpike, models.AnomalyTypePriceDrop)
	case "iqr":
		anomaly = ad.detectIQRAnomaly(rawData.Symbol, rawData.Price, historicalData, models.AnomalyTypePriceSpike, models.AnomalyTypePriceDrop)
	case "statistical":
		anomaly = ad.detectStatisticalAnomaly(rawData.Symbol, rawData.Price, historicalData, models.AnomalyTypePriceSpike, models.AnomalyTypePriceDrop)
	default:
		return nil, fmt.Errorf("unsupported anomaly detection method: %s", ad.config.Method)
	}

	if anomaly != nil {
		anomaly.Timestamp = rawData.Timestamp
		anomaly.Window = ad.config.Window
	}

	return anomaly, nil
}

// DetectVolumeAnomaly detects anomalies in volume data
func (ad *AnomalyDetector) DetectVolumeAnomaly(ctx context.Context, rawData *models.RawCryptoData) (*models.AnomalyDetectionResult, error) {
	// Get historical volume data
	historicalData, err := ad.getHistoricalData(ctx, rawData.Symbol, models.MetricTypeVolume, ad.config.Window)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical volume data: %w", err)
	}

	if len(historicalData) < ad.config.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points for anomaly detection: %d < %d", len(historicalData), ad.config.MinDataPoints)
	}

	// Detect anomaly based on configured method
	var anomaly *models.AnomalyDetectionResult
	switch ad.config.Method {
	case "zscore":
		anomaly = ad.detectZScoreAnomaly(rawData.Symbol, rawData.Volume, historicalData, models.AnomalyTypeVolumeSpike, models.AnomalyTypeVolumeDrop)
	case "iqr":
		anomaly = ad.detectIQRAnomaly(rawData.Symbol, rawData.Volume, historicalData, models.AnomalyTypeVolumeSpike, models.AnomalyTypeVolumeDrop)
	case "statistical":
		anomaly = ad.detectStatisticalAnomaly(rawData.Symbol, rawData.Volume, historicalData, models.AnomalyTypeVolumeSpike, models.AnomalyTypeVolumeDrop)
	default:
		return nil, fmt.Errorf("unsupported anomaly detection method: %s", ad.config.Method)
	}

	if anomaly != nil {
		anomaly.Timestamp = rawData.Timestamp
		anomaly.Window = ad.config.Window
	}

	return anomaly, nil
}

// detectZScoreAnomaly detects anomalies using Z-score method
func (ad *AnomalyDetector) detectZScoreAnomaly(symbol string, currentValue float64, historicalData []float64, spikeType, dropType models.AnomalyType) *models.AnomalyDetectionResult {
	mean := calculateMean(historicalData)
	stdDev := calculateStdDev(historicalData, mean)

	if stdDev == 0 {
		return nil // No variation in data
	}

	zScore := math.Abs((currentValue - mean) / stdDev)

	if zScore > ad.config.Threshold {
		anomalyType := spikeType
		if currentValue < mean {
			anomalyType = dropType
		}

		severity := ad.determineSeverity(zScore)
		confidence := math.Min(zScore/ad.config.Threshold, 1.0)

		return &models.AnomalyDetectionResult{
			Symbol:        symbol,
			Type:          anomalyType,
			Severity:      severity,
			CurrentValue:  currentValue,
			ExpectedValue: mean,
			Deviation:     currentValue - mean,
			Confidence:    confidence,
		}
	}

	return nil
}

// detectIQRAnomaly detects anomalies using Interquartile Range method
func (ad *AnomalyDetector) detectIQRAnomaly(symbol string, currentValue float64, historicalData []float64, spikeType, dropType models.AnomalyType) *models.AnomalyDetectionResult {
	sortedData := make([]float64, len(historicalData))
	copy(sortedData, historicalData)
	sort.Float64s(sortedData)

	q1 := calculatePercentile(sortedData, 25)
	q3 := calculatePercentile(sortedData, 75)
	iqr := q3 - q1

	if iqr == 0 {
		return nil // No variation in data
	}

	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	if currentValue < lowerBound || currentValue > upperBound {
		anomalyType := spikeType
		if currentValue < lowerBound {
			anomalyType = dropType
		}

		// Calculate severity based on distance from bounds
		var distance float64
		if currentValue < lowerBound {
			distance = (lowerBound - currentValue) / iqr
		} else {
			distance = (currentValue - upperBound) / iqr
		}

		severity := ad.determineSeverity(distance)
		confidence := math.Min(distance/3.0, 1.0) // Normalize to 0-1 range

		return &models.AnomalyDetectionResult{
			Symbol:        symbol,
			Type:          anomalyType,
			Severity:      severity,
			CurrentValue:  currentValue,
			ExpectedValue: (q1 + q3) / 2, // Median
			Deviation:     currentValue - (q1+q3)/2,
			Confidence:    confidence,
		}
	}

	return nil
}

// detectStatisticalAnomaly detects anomalies using statistical methods
func (ad *AnomalyDetector) detectStatisticalAnomaly(symbol string, currentValue float64, historicalData []float64, spikeType, dropType models.AnomalyType) *models.AnomalyDetectionResult {
	mean := calculateMean(historicalData)
	stdDev := calculateStdDev(historicalData, mean)

	if stdDev == 0 {
		return nil
	}

	// Calculate confidence interval
	confidenceLevel := ad.config.ConfidenceLevel
	if confidenceLevel == 0 {
		confidenceLevel = 0.95 // Default 95% confidence
	}

	// For normal distribution, 95% confidence interval is ±1.96 standard deviations
	zValue := 1.96 // For 95% confidence
	if confidenceLevel == 0.99 {
		zValue = 2.58 // For 99% confidence
	} else if confidenceLevel == 0.90 {
		zValue = 1.65 // For 90% confidence
	}

	lowerBound := mean - zValue*stdDev
	upperBound := mean + zValue*stdDev

	if currentValue < lowerBound || currentValue > upperBound {
		anomalyType := spikeType
		if currentValue < lowerBound {
			anomalyType = dropType
		}

		// Calculate severity based on distance from confidence interval
		var distance float64
		if currentValue < lowerBound {
			distance = (lowerBound - currentValue) / stdDev
		} else {
			distance = (currentValue - upperBound) / stdDev
		}

		severity := ad.determineSeverity(distance)
		confidence := math.Min(distance/zValue, 1.0)

		return &models.AnomalyDetectionResult{
			Symbol:        symbol,
			Type:          anomalyType,
			Severity:      severity,
			CurrentValue:  currentValue,
			ExpectedValue: mean,
			Deviation:     currentValue - mean,
			Confidence:    confidence,
		}
	}

	return nil
}

// determineSeverity determines the severity level based on the deviation
func (ad *AnomalyDetector) determineSeverity(deviation float64) models.AnomalySeverity {
	if deviation >= 4.0 {
		return models.AnomalySeverityCritical
	} else if deviation >= 3.0 {
		return models.AnomalySeverityHigh
	} else if deviation >= 2.0 {
		return models.AnomalySeverityMedium
	} else {
		return models.AnomalySeverityLow
	}
}

// getHistoricalData retrieves historical data from Redis
func (ad *AnomalyDetector) getHistoricalData(ctx context.Context, symbol string, metricType models.MetricType, window time.Duration) ([]float64, error) {
	endTime := time.Now()
	startTime := endTime.Add(-window)

	key := fmt.Sprintf("metrics:%s:%s", metricType, symbol)
	startScore := float64(startTime.Unix())
	endScore := float64(endTime.Unix())

	results, err := ad.redisClient.ZRangeByScore(ctx, key, startScore, endScore)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from Redis: %w", err)
	}

	values := make([]float64, 0, len(results))
	for _, result := range results {
		var value float64
		if err := json.Unmarshal([]byte(result.Member.(string)), &value); err != nil {
			log.Printf("Failed to unmarshal metric value: %v", err)
			continue
		}
		values = append(values, value)
	}

	return values, nil
}

// Statistical helper functions
func calculateMean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func calculateStdDev(data []float64, mean float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range data {
		diff := value - mean
		sum += diff * diff
	}

	variance := sum / float64(len(data))
	return math.Sqrt(variance)
}

func calculatePercentile(data []float64, percentile float64) float64 {
	if len(data) == 0 {
		return 0
	}

	index := (percentile / 100.0) * float64(len(data)-1)
	if index == float64(int(index)) {
		return data[int(index)]
	}

	lower := int(index)
	upper := lower + 1
	if upper >= len(data) {
		return data[lower]
	}

	weight := index - float64(lower)
	return data[lower]*(1-weight) + data[upper]*weight
}
