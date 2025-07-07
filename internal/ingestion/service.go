package ingestion

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/config"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
)

type Service struct {
	config    *config.Config
	producer  *kafka.Producer
	ingesters []*WebSocketIngester
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewService(cfg *config.Config) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers configuration is required")
	}

	if cfg.Kafka.Topics.RawData == "" {
		return nil, fmt.Errorf("kafka raw data topic configuration is required")
	}

	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.RawData, "ingestion-service")

	return &Service{
		config:   cfg,
		producer: producer,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	defer s.cancel()

	// Initialize Websocket ingesters
	binanceIngester, err := NewWebSocketIngester(s.producer, s.config.WebSocket.BinanceURL, "binance")
	if err != nil {
		return fmt.Errorf("failed to create binance ingester: %w", err)
	}
	s.ingesters = append(s.ingesters, binanceIngester)

	// Start all ingesters concurrently
	for _, ingester := range s.ingesters {
		s.wg.Add(1)
		go func(ing *WebSocketIngester) {
			defer s.wg.Done()
			if err := ing.Start(s.ctx); err != nil {
				log.Printf("Ingester %s error: %v", ing.source, err)
				// Cancel context to signal other ingesters to stop
				s.cancel()
			}
		}(ingester)
	}

	// Wait for context cancellation or all ingesters to complete
	<-s.ctx.Done()

	// Wait for all ingesters to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All ingesters stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for ingesters to stop")
	}

	return nil
}

func (s *Service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	// Wait for all ingesters to stop
	s.wg.Wait()

	// Close the producer
	if s.producer != nil {
		return s.producer.Close()
	}

	return nil
}

// GetStatus returns the current status of the service
func (s *Service) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"active_ingesters": len(s.ingesters),
		"running":          s.ctx != nil && s.ctx.Err() == nil,
	}

	if s.ctx != nil {
		status["context_error"] = s.ctx.Err()
	}

	return status
}
