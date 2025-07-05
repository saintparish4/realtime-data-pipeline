package unit

import (
	"context"
	"testing"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/processing"
	"github.com/stretchr/testify/assert"
)

func TestProcessData(t *testing.T) {
	// Mock Dependecies

	mockConsumer := &mockKafkaConsumer{}
	mockProducer := &mockKafkaProducer{}
	mockRedis := &mockRedisClient{}

	engine := processing.NewEngine(mockConsumer, mockProducer, mockRedis)

	// Test Data Processing
	rawData := &models.RawCryptoData{
		Symbol:    "BTCUSDT",
		Price:     50000.0,
		Volume:    1000.0,
		Timestamp: time.Now(),
		Source:    "binance",
	}

	processedData, err := engine.ProcessData(context.Background(), rawData)

	assert.NoError(t, err)
	assert.Equal(t, rawData.Symbol, processedData.Symbol)
	assert.Equal(t, rawData.Price, processedData.Price)
	assert.Equal(t, rawData.Volume, processedData.Volume)
}
