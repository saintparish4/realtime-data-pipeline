package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration structure.
// It contains all configuration settings for the realtime data pipeline.
type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	Kafka struct {
		Brokers []string `yaml:"brokers"`
		Topics  struct {
			// Observability Event Streaming Topics
			ObservabilityEvents string `yaml:"observability_events"`
			Logs                string `yaml:"logs"`
			Metrics             string `yaml:"metrics"`
			Alerts              string `yaml:"alerts"`
			Traces              string `yaml:"traces"`
			HealthChecks        string `yaml:"health_checks"`

			// Legacy topics for backward compatibility
			RawData       string `yaml:"raw_data"`
			ProcessedData string `yaml:"processed_data"`
		} `yaml:"topics"`

		EventStreaming struct {
			BatchSize         int    `yaml:"batch_size"`
			BatchTimeout      string `yaml:"batch_timeout"`
			MaxMessageSize    string `yaml:"max_message_size"`
			RetentionPolicy   string `yaml:"retention_policy"`
			ReplicationFactor int    `yaml:"replication_factor"`
			Partitions        int    `yaml:"partitions"`
		} `yaml:"event_streaming"`
	} `yaml:"kafka"`

	Redis struct {
		Address  string `yaml:"address"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	Postgres struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		SSLMode  string `yaml:"ssl_mode"`
	} `yaml:"postgres"`

	TimescaleDB struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		SSLMode  string `yaml:"ssl_mode"`
	} `yaml:"timescaledb"`

	WebSocket struct {
		BinanceURL  string `yaml:"binance_url"`
		CoinbaseURL string `yaml:"coinbase_url"`
	} `yaml:"websocket"`

	Metrics struct {
		PrometheusPort int `yaml:"prometheus_port"`
	} `yaml:"metrics"`

	Observability struct {
		MetricsRetention      string `yaml:"metrics_retention"`       // e.g., "30d", "7d"
		AlertCheckInterval    string `yaml:"alert_check_interval"`    // e.g., "30s", "1m"
		NotificationRateLimit string `yaml:"notification_rate_limit"` // e.g., "5m", "1h"
		CleanupInterval       string `yaml:"cleanup_interval"`        // e.g., "24h", "12h"

		EventStreaming struct {
			CorrelationIDHeader       string  `yaml:"correlation_id_header"`
			TraceIDHeader             string  `yaml:"trace_id_header"`
			SpanIDHeader              string  `yaml:"span_id_header"`
			ServiceName               string  `yaml:"service_name"`
			DefaultEventLevel         string  `yaml:"default_event_level"`
			EnableDistributedTracing  bool    `yaml:"enable_distributed_tracing"`
			EnableCorrelationTracking bool    `yaml:"enable_correlation_tracking"`
			EventProcessingTimeout    string  `yaml:"event_processing_timeout"`
			MaxEventSize              string  `yaml:"max_event_size"`
			EnableEventValidation     bool    `yaml:"enable_event_validation"`
			AlertEventRetention       string  `yaml:"alert_event_retention"`
			AlertCorrelationWindow    string  `yaml:"alert_correlation_window"`
			TraceSamplingRate         float64 `yaml:"trace_sampling_rate"`
			TraceRetention            string  `yaml:"trace_retention"`
			EnableTraceAggregation    bool    `yaml:"enable_trace_aggregation"`
		} `yaml:"event_streaming"`
	} `yaml:"observability"`

	Notifications struct {
		Email struct {
			SMTPHost    string `yaml:"smtp_host"`
			SMTPPort    int    `yaml:"smtp_port"`
			Username    string `yaml:"username"`
			Password    string `yaml:"password"`
			FromAddress string `yaml:"from_address"`
			FromName    string `yaml:"from_name"`
			UseTLS      bool   `yaml:"use_tls"`
		} `yaml:"email"`

		Slack struct {
			WebhookURL string `yaml:"webhook_url"`
			Channel    string `yaml:"channel"`
			Username   string `yaml:"username"`
			IconEmoji  string `yaml:"icon_emoji"`
		} `yaml:"slack"`

		Webhooks []struct {
			Name    string            `yaml:"name"`
			URL     string            `yaml:"url"`
			Method  string            `yaml:"method"`
			Headers map[string]string `yaml:"headers"`
			Timeout string            `yaml:"timeout"`
			Enabled bool              `yaml:"enabled"`
		} `yaml:"webhooks"`

		RateLimit struct {
			MaxPerMinute int    `yaml:"max_per_minute"`
			MaxPerHour   int    `yaml:"max_per_hour"`
			Window       string `yaml:"window"`
		} `yaml:"rate_limit"`

		Retry struct {
			MaxAttempts int    `yaml:"max_attempts"`
			Backoff     string `yaml:"backoff"`
		} `yaml:"retry"`
	} `yaml:"notifications"`

	Tracing struct {
		ServiceName       string  `yaml:"service_name"`
		SamplingRate      float64 `yaml:"sampling_rate"`
		MaxTraceDuration  string  `yaml:"max_trace_duration"`
		TraceRetention    string  `yaml:"trace_retention"`
		EnableCorrelation bool    `yaml:"enable_correlation"`
		EnableAggregation bool    `yaml:"enable_aggregation"`
	} `yaml:"tracing"`
}

// Load reads and parses the configuration file from the given path.
// It returns a validated Config struct or an error if the file cannot be read or parsed.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values
	setDefaults(&cfg)

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration fields that are not specified.
func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	if cfg.Redis.Address == "" {
		cfg.Redis.Address = "localhost:6379"
	}
	if cfg.Postgres.Port == 0 {
		cfg.Postgres.Port = 5432
	}
	if cfg.Postgres.SSLMode == "" {
		cfg.Postgres.SSLMode = "disable"
	}
	if cfg.Metrics.PrometheusPort == 0 {
		cfg.Metrics.PrometheusPort = 9090
	}
}

// validate performs validation on the configuration values.
func validate(cfg *Config) error {
	var errors []string

	// Validate Server configuration
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		errors = append(errors, "server port must be between 1 and 65535")
	}
	if cfg.Server.Host == "" {
		errors = append(errors, "server host cannot be empty")
	}

	// Validate Kafka configuration
	if len(cfg.Kafka.Brokers) == 0 {
		errors = append(errors, "at least one Kafka broker must be specified")
	}
	if cfg.Kafka.Topics.RawData == "" {
		errors = append(errors, "kafka raw_data topic cannot be empty")
	}
	if cfg.Kafka.Topics.ProcessedData == "" {
		errors = append(errors, "kafka processed_data topic cannot be empty")
	}

	// Validate Redis configuration
	if cfg.Redis.Address == "" {
		errors = append(errors, "redis address cannot be empty")
	}
	if cfg.Redis.DB < 0 {
		errors = append(errors, "redis db must be non-negative")
	}

	// Validate Postgres configuration
	if cfg.Postgres.Host == "" {
		errors = append(errors, "postgres host cannot be empty")
	}
	if cfg.Postgres.Port <= 0 || cfg.Postgres.Port > 65535 {
		errors = append(errors, "postgres port must be between 1 and 65535")
	}
	if cfg.Postgres.User == "" {
		errors = append(errors, "postgres user cannot be empty")
	}
	if cfg.Postgres.Database == "" {
		errors = append(errors, "postgres database cannot be empty")
	}

	// Validate WebSocket URLs
	if cfg.WebSocket.BinanceURL != "" {
		if _, err := url.Parse(cfg.WebSocket.BinanceURL); err != nil {
			errors = append(errors, fmt.Sprintf("invalid binance URL: %v", err))
		}
	}
	if cfg.WebSocket.CoinbaseURL != "" {
		if _, err := url.Parse(cfg.WebSocket.CoinbaseURL); err != nil {
			errors = append(errors, fmt.Sprintf("invalid coinbase URL: %v", err))
		}
	}

	// Validate Metrics configuration
	if cfg.Metrics.PrometheusPort <= 0 || cfg.Metrics.PrometheusPort > 65535 {
		errors = append(errors, "prometheus port must be between 1 and 65535")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetDSN returns the PostgreSQL connection string.
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Postgres.Host, c.Postgres.Port, c.Postgres.User, c.Postgres.Password,
		c.Postgres.Database, c.Postgres.SSLMode)
}

// GetRedisAddr returns the Redis address.
func (c *Config) GetRedisAddr() string {
	return c.Redis.Address
}

// GetServerAddr returns the server address in the format "host:port".
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
