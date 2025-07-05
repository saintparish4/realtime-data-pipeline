package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
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
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
		User string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"postgres"`

	WebSocket struct {
		BinanceURL string `yaml:"binance_url"`
		CoinbaseURL string `yaml:"coinbase_url"`
	} `yaml:"websocket"`

	Metrics struct {
		PrometheusPort string `yaml:"prometheus_port"`
	} `yaml:"metrics"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
