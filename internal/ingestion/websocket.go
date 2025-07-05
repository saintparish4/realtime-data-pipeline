package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
)

type WebSocketIngester struct {
	producer    *kafka.Producer
	conn        *websocket.Conn
	source      string
	wsURL       string
	mu          sync.RWMutex
	connected   bool
	reconnectCh chan struct{}
	stopCh      chan struct{}
}

type WebSocketConfig struct {
	URL           string
	ReconnectWait time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	MaxRetries    int
}

func NewWebSocketIngester(producer *kafka.Producer, wsURL, source string) (*WebSocketIngester, error) {
	if producer == nil {
		return nil, fmt.Errorf("producer cannot be nil")
	}

	if wsURL == "" {
		return nil, fmt.Errorf("websocket URL cannot be empty")
	}

	if source == "" {
		return nil, fmt.Errorf("source cannot be empty")
	}

	return &WebSocketIngester{
		producer:    producer,
		source:      source,
		wsURL:       wsURL,
		reconnectCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}, nil
}

func (w *WebSocketIngester) Start(ctx context.Context) error {
	log.Printf("Starting WebSocket ingester for source: %s", w.source)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return fmt.Errorf("ingester stopped")
		default:
			if err := w.connect(); err != nil {
				log.Printf("Failed to connect to %s: %v", w.source, err)
				time.Sleep(5 * time.Second)
				continue
			}

			if err := w.listen(ctx); err != nil {
				log.Printf("WebSocket error for %s: %v", w.source, err)
				w.disconnect()

				// Check if we should stop
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-w.stopCh:
					return fmt.Errorf("ingester stopped")
				default:
					// Trigger reconnection
					select {
					case w.reconnectCh <- struct{}{}:
					default:
					}
				}
			}
		}
	}
}

func (w *WebSocketIngester) connect() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.connected {
		return nil
	}

	u, err := url.Parse(w.wsURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	// Set connection parameters
	conn.SetReadLimit(512 * 1024) // 512KB max message size
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	w.conn = conn
	w.connected = true
	log.Printf("Connected to WebSocket for source: %s", w.source)

	return nil
}

func (w *WebSocketIngester) disconnect() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connected = false
	log.Printf("Disconnected from WebSocket for source: %s", w.source)
}

func (w *WebSocketIngester) listen(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return fmt.Errorf("ingester stopped")
		default:
			_, message, err := w.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return fmt.Errorf("websocket read error: %w", err)
				}
				return err
			}

			if err := w.processMessage(ctx, message); err != nil {
				log.Printf("Error processing message from %s: %v", w.source, err)
				// Continue processing other messages even if one fails
			}
		}
	}
}

func (w *WebSocketIngester) processMessage(ctx context.Context, message []byte) error {
	if len(message) == 0 {
		return fmt.Errorf("empty message received")
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(message, &rawData); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate required fields based on source
	if err := w.validateMessage(rawData); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Transform based on source
	cryptoData := w.transformData(rawData)
	if cryptoData == nil {
		return fmt.Errorf("failed to transform data for source %s", w.source)
	}

	// Publish to Kafka
	if err := w.producer.Publish(ctx, cryptoData.Symbol, cryptoData); err != nil {
		return fmt.Errorf("failed to publish to kafka: %w", err)
	}

	return nil
}

func (w *WebSocketIngester) validateMessage(data map[string]interface{}) error {
	switch w.source {
	case "binance":
		// Validate Binance ticker data structure
		if _, ok := data["s"]; !ok {
			return fmt.Errorf("missing symbol field")
		}
		if _, ok := data["c"]; !ok {
			return fmt.Errorf("missing price field")
		}
		if _, ok := data["v"]; !ok {
			return fmt.Errorf("missing volume field")
		}
	default:
		// For unknown sources, just check if data is not empty
		if len(data) == 0 {
			return fmt.Errorf("empty data received")
		}
	}
	return nil
}

func (w *WebSocketIngester) transformData(rawData map[string]interface{}) *models.RawCryptoData {
	// This would be source-specific transformation logic
	// For Binance ticker data
	if w.source == "binance" {
		rawJSON, _ := json.Marshal(rawData)
		return &models.RawCryptoData{
			Symbol:    getString(rawData, "s"),
			Price:     getFloat64(rawData, "c"),
			Volume:    getFloat64(rawData, "v"),
			Timestamp: time.Now(),
			Source:    w.source,
			Raw:       rawJSON,
		}
	}

	return nil
}

func (w *WebSocketIngester) Stop() {
	close(w.stopCh)
	w.disconnect()
}

func (w *WebSocketIngester) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connected
}

func (w *WebSocketIngester) GetSource() string {
	return w.source
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	if val, ok := data[key].(int64); ok {
		return float64(val)
	}
	return 0
}
