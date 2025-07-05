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
			RawData       string `yaml:"raw_data"`
			ProcessedData string `yaml:"processed_data"`
		} `yaml:"topics"`
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

	WebSocket struct {
		BinanceURL  string `yaml:"binance_url"`
		CoinbaseURL string `yaml:"coinbase_url"`
	} `yaml:"websocket"`

	Metrics struct {
		PrometheusPort int `yaml:"prometheus_port"`
	} `yaml:"metrics"`
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
