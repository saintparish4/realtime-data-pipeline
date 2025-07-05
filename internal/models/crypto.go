package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Time window constants for data aggregation
const (
	TimeWindow1m  = "1m"
	TimeWindow5m  = "5m"
	TimeWindow15m = "15m"
	TimeWindow1h  = "1h"
	TimeWindow4h  = "4h"
	TimeWindow1d  = "1d"
)

// ValidTimeWindows contains all valid time window values
var ValidTimeWindows = []string{
	TimeWindow1m, TimeWindow5m, TimeWindow15m, TimeWindow1h, TimeWindow4h, TimeWindow1d,
}

// RawCryptoData represents unprocessed cryptocurrency data from external sources.
// This is the initial data structure received from data providers.
type RawCryptoData struct {
	Symbol    string          `json:"symbol" validate:"required"`
	Price     float64         `json:"price" validate:"gt=0"`
	Volume    float64         `json:"volume" validate:"gte=0"`
	Timestamp time.Time       `json:"timestamp" validate:"required"`
	Source    string          `json:"source" validate:"required"`
	Raw       json.RawMessage `json:"-"` // Original raw JSON data
}

// NewRawCryptoData creates a new RawCryptoData instance with validation
func NewRawCryptoData(symbol string, price, volume float64, timestamp time.Time, source string) (*RawCryptoData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}
	if price <= 0 {
		return nil, fmt.Errorf("price must be greater than 0")
	}
	if volume < 0 {
		return nil, fmt.Errorf("volume cannot be negative")
	}
	if source == "" {
		return nil, fmt.Errorf("source cannot be empty")
	}

	return &RawCryptoData{
		Symbol:    symbol,
		Price:     price,
		Volume:    volume,
		Timestamp: timestamp,
		Source:    source,
	}, nil
}

// Validate performs basic validation on the RawCryptoData
func (r *RawCryptoData) Validate() error {
	if r.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if r.Price <= 0 {
		return fmt.Errorf("price must be greater than 0")
	}
	if r.Volume < 0 {
		return fmt.Errorf("volume cannot be negative")
	}
	if r.Source == "" {
		return fmt.Errorf("source cannot be empty")
	}
	return nil
}

// UnmarshalRaw unmarshals the raw JSON data into the provided interface
func (r *RawCryptoData) UnmarshalRaw(v interface{}) error {
	if r.Raw == nil {
		return fmt.Errorf("no raw data available")
	}
	return json.Unmarshal(r.Raw, v)
}

// SetRaw sets the raw JSON data
func (r *RawCryptoData) SetRaw(data json.RawMessage) {
	r.Raw = data
}

// ProcessedCryptoData represents cryptocurrency data after initial processing.
// This includes calculated metrics like moving averages and price changes.
type ProcessedCryptoData struct {
	Symbol          string    `json:"symbol" validate:"required"`
	Price           float64   `json:"price" validate:"gt=0"`
	Volume          float64   `json:"volume" validate:"gte=0"`
	PriceChange24h  float64   `json:"price_change_24h"`
	VolumeChange24h float64   `json:"volume_change_24h"`
	MovingAvg5m     float64   `json:"moving_avg_5m"`
	MovingAvg15m    float64   `json:"moving_avg_15m"`
	MovingAvg1h     float64   `json:"moving_avg_1h"`
	Timestamp       time.Time `json:"timestamp" validate:"required"`
	ProcessedAt     time.Time `json:"processed_at" validate:"required"`
}

// NewProcessedCryptoData creates a new ProcessedCryptoData instance
func NewProcessedCryptoData(symbol string, price, volume float64, timestamp time.Time) *ProcessedCryptoData {
	return &ProcessedCryptoData{
		Symbol:      symbol,
		Price:       price,
		Volume:      volume,
		Timestamp:   timestamp,
		ProcessedAt: time.Now(),
	}
}

// Validate performs basic validation on the ProcessedCryptoData
func (p *ProcessedCryptoData) Validate() error {
	if p.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if p.Price <= 0 {
		return fmt.Errorf("price must be greater than 0")
	}
	if p.Volume < 0 {
		return fmt.Errorf("volume cannot be negative")
	}
	return nil
}

// GetPriceChangePercent returns the 24h price change as a percentage
func (p *ProcessedCryptoData) GetPriceChangePercent() float64 {
	if p.Price == 0 {
		return 0
	}
	return (p.PriceChange24h / p.Price) * 100
}

// GetVolumeChangePercent returns the 24h volume change as a percentage
func (p *ProcessedCryptoData) GetVolumeChangePercent() float64 {
	if p.Volume == 0 {
		return 0
	}
	return (p.VolumeChange24h / p.Volume) * 100
}

// IsPriceUp returns true if the price has increased in the last 24h
func (p *ProcessedCryptoData) IsPriceUp() bool {
	return p.PriceChange24h > 0
}

// IsVolumeUp returns true if the volume has increased in the last 24h
func (p *ProcessedCryptoData) IsVolumeUp() bool {
	return p.VolumeChange24h > 0
}

// GetMovingAverage returns the moving average for the specified time window
func (p *ProcessedCryptoData) GetMovingAverage(timeWindow string) float64 {
	switch timeWindow {
	case TimeWindow5m:
		return p.MovingAvg5m
	case TimeWindow15m:
		return p.MovingAvg15m
	case TimeWindow1h:
		return p.MovingAvg1h
	default:
		return 0
	}
}

// AggregatedCryptoData represents time-window aggregated cryptocurrency data.
// This is used for OHLC (Open, High, Low, Close) data and volume analysis.
type AggregatedCryptoData struct {
	Symbol      string    `json:"symbol" validate:"required"`
	TimeWindow  string    `json:"time_window" validate:"required"`
	OpenPrice   float64   `json:"open_price" validate:"gt=0"`
	ClosePrice  float64   `json:"close_price" validate:"gt=0"`
	HighPrice   float64   `json:"high_price" validate:"gt=0"`
	LowPrice    float64   `json:"low_price" validate:"gt=0"`
	Volume      float64   `json:"volume" validate:"gte=0"`
	TradeCount  int64     `json:"trade_count" validate:"gte=0"`
	WindowStart time.Time `json:"window_start" validate:"required"`
	WindowEnd   time.Time `json:"window_end" validate:"required"`
}

// NewAggregatedCryptoData creates a new AggregatedCryptoData instance
func NewAggregatedCryptoData(symbol, timeWindow string, windowStart, windowEnd time.Time) *AggregatedCryptoData {
	return &AggregatedCryptoData{
		Symbol:      symbol,
		TimeWindow:  timeWindow,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	}
}

// Validate performs basic validation on the AggregatedCryptoData
func (a *AggregatedCryptoData) Validate() error {
	if a.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if !isValidTimeWindow(a.TimeWindow) {
		return fmt.Errorf("invalid time window: %s", a.TimeWindow)
	}
	if a.OpenPrice <= 0 || a.ClosePrice <= 0 || a.HighPrice <= 0 || a.LowPrice <= 0 {
		return fmt.Errorf("all prices must be greater than 0")
	}
	if a.Volume < 0 {
		return fmt.Errorf("volume cannot be negative")
	}
	if a.TradeCount < 0 {
		return fmt.Errorf("trade count cannot be negative")
	}
	if a.WindowStart.After(a.WindowEnd) {
		return fmt.Errorf("window start must be before window end")
	}
	return nil
}

// GetPriceRange returns the difference between high and low prices
func (a *AggregatedCryptoData) GetPriceRange() float64 {
	return a.HighPrice - a.LowPrice
}

// GetPriceChange returns the difference between close and open prices
func (a *AggregatedCryptoData) GetPriceChange() float64 {
	return a.ClosePrice - a.OpenPrice
}

// GetPriceChangePercent returns the price change as a percentage
func (a *AggregatedCryptoData) GetPriceChangePercent() float64 {
	if a.OpenPrice == 0 {
		return 0
	}
	return (a.GetPriceChange() / a.OpenPrice) * 100
}

// IsBullish returns true if the close price is higher than the open price
func (a *AggregatedCryptoData) IsBullish() bool {
	return a.ClosePrice > a.OpenPrice
}

// IsBearish returns true if the close price is lower than the open price
func (a *AggregatedCryptoData) IsBearish() bool {
	return a.ClosePrice < a.OpenPrice
}

// GetDuration returns the duration of the time window
func (a *AggregatedCryptoData) GetDuration() time.Duration {
	return a.WindowEnd.Sub(a.WindowStart)
}

// GetAveragePrice returns the average of high and low prices
func (a *AggregatedCryptoData) GetAveragePrice() float64 {
	return (a.HighPrice + a.LowPrice) / 2
}

// GetVolumePerTrade returns the average volume per trade
func (a *AggregatedCryptoData) GetVolumePerTrade() float64 {
	if a.TradeCount == 0 {
		return 0
	}
	return a.Volume / float64(a.TradeCount)
}

// isValidTimeWindow checks if the provided time window is valid
func isValidTimeWindow(timeWindow string) bool {
	for _, valid := range ValidTimeWindows {
		if valid == timeWindow {
			return true
		}
	}
	return false
}

// IsValidTimeWindow is a public function to check time window validity
func IsValidTimeWindow(timeWindow string) bool {
	return isValidTimeWindow(timeWindow)
}
