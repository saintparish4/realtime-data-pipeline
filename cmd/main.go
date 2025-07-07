package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/api"
	"github.com/saintparish4/realtime-data-pipeline/internal/config"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/notifications"
	"github.com/saintparish4/realtime-data-pipeline/internal/observability"
	"github.com/saintparish4/realtime-data-pipeline/internal/processing"
	"github.com/saintparish4/realtime-data-pipeline/internal/storage"
	"github.com/saintparish4/realtime-data-pipeline/internal/tracing"
	"github.com/saintparish4/realtime-data-pipeline/pkg/database"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)

	// Initialize PostgreSQL
	postgresDB, err := database.NewPostgreSQL(cfg.Postgres.Host, fmt.Sprintf("%d", cfg.Postgres.Port), cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Database)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	// Initialize database tables and indexes
	ctx := context.Background()
	if err := postgresDB.CreateTables(ctx); err != nil {
		log.Fatalf("Failed to create PostgreSQL tables: %v", err)
	}
	if err := postgresDB.CreateIndexes(ctx); err != nil {
		log.Fatalf("Failed to create PostgreSQL indexes: %v", err)
	}

	// Initialize TimescaleDB
	timescaleDB, err := database.NewTimescaleDB(cfg.TimescaleDB.Host, fmt.Sprintf("%d", cfg.TimescaleDB.Port), cfg.TimescaleDB.User, cfg.TimescaleDB.Password, cfg.TimescaleDB.Database)
	if err != nil {
		log.Fatalf("Failed to initialize TimescaleDB: %v", err)
	}
	defer timescaleDB.Close()

	// Initialize Kafka producer
	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.RawData, cfg.Kafka.Topics.ProcessedData)
	defer kafkaProducer.Close()

	// Initialize Kafka consumer
	kafkaConsumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topics.RawData, cfg.Kafka.Topics.ProcessedData)
	defer kafkaConsumer.Close()

	// Initialize storage manager with actual database instances
	storageManager := storage.NewStorageManager(timescaleDB, postgresDB)

	// Initialize observability service
	obsService := observability.NewService(redisClient, storageManager)

	// Initialize notification service
	notificationConfig := &notifications.Config{
		Email: struct {
			SMTPHost    string `yaml:"smtp_host"`
			SMTPPort    int    `yaml:"smtp_port"`
			Username    string `yaml:"username"`
			Password    string `yaml:"password"`
			FromAddress string `yaml:"from_address"`
			FromName    string `yaml:"from_name"`
			UseTLS      bool   `yaml:"use_tls"`
		}{
			SMTPHost:    cfg.Notifications.Email.SMTPHost,
			SMTPPort:    cfg.Notifications.Email.SMTPPort,
			Username:    cfg.Notifications.Email.Username,
			Password:    cfg.Notifications.Email.Password,
			FromAddress: cfg.Notifications.Email.FromAddress,
			FromName:    cfg.Notifications.Email.FromName,
			UseTLS:      cfg.Notifications.Email.UseTLS,
		},
		Slack: struct {
			WebhookURL string `yaml:"webhook_url"`
			Channel    string `yaml:"channel"`
			Username   string `yaml:"username"`
			IconEmoji  string `yaml:"icon_emoji"`
		}{
			WebhookURL: cfg.Notifications.Slack.WebhookURL,
			Channel:    cfg.Notifications.Slack.Channel,
			Username:   cfg.Notifications.Slack.Username,
			IconEmoji:  cfg.Notifications.Slack.IconEmoji,
		},
		RateLimit: struct {
			MaxPerMinute int           `yaml:"max_per_minute"`
			MaxPerHour   int           `yaml:"max_per_hour"`
			Window       time.Duration `yaml:"window"`
		}{
			MaxPerMinute: cfg.Notifications.RateLimit.MaxPerMinute,
			MaxPerHour:   cfg.Notifications.RateLimit.MaxPerHour,
			Window:       time.Minute, // Default, could parse from config
		},
		Retry: struct {
			MaxAttempts int           `yaml:"max_attempts"`
			Backoff     time.Duration `yaml:"backoff"`
		}{
			MaxAttempts: cfg.Notifications.Retry.MaxAttempts,
			Backoff:     time.Second * 5, // Default, could parse from config
		},
	}
	notificationService := notifications.NewService(redisClient, notificationConfig)

	// Initialize tracing service
	tracingConfig := &tracing.Config{
		ServiceName:       cfg.Tracing.ServiceName,
		SamplingRate:      cfg.Tracing.SamplingRate,
		MaxTraceDuration:  5 * time.Minute,    // Default, could parse from config
		TraceRetention:    7 * 24 * time.Hour, // Default, could parse from config
		EnableCorrelation: cfg.Tracing.EnableCorrelation,
		EnableAggregation: cfg.Tracing.EnableAggregation,
	}
	tracingService := tracing.NewService(redisClient, tracingConfig)

	// Initialize processing components with default config
	metricsConfig := processing.NewMetricsEngineConfig()
	metricsEngine := processing.NewMetricsEngine(kafkaConsumer, kafkaProducer, redisClient, obsService, metricsConfig)
	alertManager := processing.NewAlertManager(redisClient, []models.AlertThreshold{})

	// Store services for potential use
	_ = notificationService
	_ = tracingService
	_ = alertManager

	// Initialize API server
	server, err := api.NewServer(cfg, redisClient, postgresDB, timescaleDB, kafkaProducer, kafkaConsumer, obsService, metricsEngine, alertManager)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background services
	go func() {
		log.Println("Starting metrics engine...")
		if err := metricsEngine.Start(ctx); err != nil {
			log.Printf("Metrics engine error: %v", err)
		}
	}()

	go func() {
		log.Println("Starting observability service...")
		if err := obsService.Initialize(ctx); err != nil {
			log.Printf("Observability service error: %v", err)
		}
	}()

	go func() {
		log.Println("Starting cleanup routines...")
		obsService.StartCleanupRoutine(ctx)
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, starting graceful shutdown...")
		cancel()

		// Give services time to shutdown gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Stop(shutdownCtx); err != nil {
			log.Printf("Error stopping server: %v", err)
		}

		log.Println("Shutdown complete")
		os.Exit(0)
	}()

	// Start API server
	log.Printf("Starting API server on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}
