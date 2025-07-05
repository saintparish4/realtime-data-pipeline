package models

import (
	"encoding/json"
	"time"
)

type RawCryptoData struct {
	Symbol    string          `json:"symbol"`
	Price     float64         `json:"price"`
	Volume    float64         `json:"volume"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Raw       json.RawMessage `json:"-"`
}

type ProcessedCryptoData struct {
	Symbol          string    `json:"symbol"`
	Price           float64   `json:"price"`
	Volume          float64   `json:"volume"`
	PriceChange24h  float64   `json:"price_change_24h"`
	VolumeChange24h float64   `json:"volume_change_24h"`
	MovingAvg5m     float64   `json:"moving_avg_5m"`
	MovingAvg15m    float64   `json:"moving_avg_15m"`
	MovingAvg1h     float64   `json:"moving_avg_1h"`
	Timestamp       time.Time `json:"timestamp"`
	ProcessedAt     time.Time `json:"processed_at"`
}

type AggregatedCryptoData struct {
	Symbol      string    `json:"symbol"`
	TimeWindow  string    `json:"time_window"`
	OpenPrice   float64   `json:"open_price"`
	ClosePrice  float64   `json:"close_price"`
	HighPrice   float64   `json:"high_price"`
	LowPrice    float64   `json:"low_price"`
	Volume      float64   `json:"volume"`
	TradeCount  int64     `json:"trade_count"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
}
