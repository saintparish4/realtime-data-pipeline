 Overview
This project demonstrates advanced real-time data processing capabilities for cryptocurrency markets, featuring:

Real-time Data Processing: Advanced metrics processing with anomaly detection
Scalable Architecture: Kafka-based event streaming with hybrid storage
Comprehensive Alerting: Multi-channel notifications with rate limiting
Modern Dashboard: Web-based visualization with real-time updates
Production-Ready: Complete testing suite and automation tools

Current Status
✅ PRODUCTION-READY DEMO - Complete real-time data processing pipeline with advanced features, comprehensive testing, and automation tools using realistic mock data.
🚀 Features
Core Functionality

✅ Alert Rule Engine - Advanced rule engine with evaluation logic and threshold management
✅ Notification System - Multi-channel notifications (Email, Slack, Webhook) with rate limiting
✅ Metrics Collection - Comprehensive REST API for metrics and data access
✅ Dashboard/Visualization - Modern web-based dashboard with real-time updates
✅ Enhanced Processing Engine - Advanced analytics with anomaly detection

Advanced Features

✅ Distributed Tracing - Comprehensive tracing and observability infrastructure
✅ Storage & Infrastructure - TimescaleDB, PostgreSQL, Redis, and Kafka integration
✅ Automation & Testing - PowerShell scripts and comprehensive test suites
✅ Observability System - Monitoring, alerting, and metrics with hybrid storage

🏗️ Architecture
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Exchanges     │    │   WebSocket     │    │   Kafka         │
│  (Binance,      │───▶│   Ingestion     │───▶│   (Raw Data)    │
│   Coinbase)     │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST API      │    │   Processing    │    │   Kafka         │
│   & WebSocket   │◀───│   Engine        │◀───│   (Processed)   │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Redis         │    │  TimescaleDB    │    │   PostgreSQL    │
│ (Real-time      │    │ (Time-series    │    │ (Alert Rules    │
│  Caching)       │    │  Metrics)       │    │  & Config)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
⚡ Quick Start
Windows (Recommended)
bash# Clone and start everything with one command
git clone https://github.com/saintparish4/realtime-data-pipeline.git
cd realtime-data-pipeline
.\start.ps1
Linux/macOS
bash# Clone and start manually
git clone https://github.com/saintparish4/realtime-data-pipeline.git
cd realtime-data-pipeline
docker-compose up -d
go run scripts/generate-mock-data.go
go run ./cmd -config cmd/config.yaml
Access the Demo

Dashboard: http://localhost:8080/dashboard
API: http://localhost:8080/api/v1/
Health: http://localhost:8080/health

📊 Mock Data System
This project uses realistic mock data for portfolio demonstration purposes:
🎯 Realistic Data Features

10 Major Cryptocurrencies: BTC, ETH, ADA, SOL, DOT, LINK, UNI, AVAX, MATIC, ATOM
Realistic Price Ranges: Based on actual market prices (e.g., BTC ~$45K, ETH ~$3.2K)
Realistic Volume Data: Market-appropriate trading volumes for each asset
Historical Data: 1,000 data points per symbol over 1 week
Market Movements: Realistic price volatility and volume variations
Time-based Patterns: Proper timestamp distribution and market timing

🔔 Alert System Examples

Price Breakouts: BTC breaking above $50K resistance
Support Breaks: ETH dropping below $3K support
Volume Spikes: Unusual trading activity detection
Volatility Alerts: Extreme price movements (SOL +18.5% in 4h)
Anomaly Detection: Statistical anomalies with confidence scores
Trend Analysis: Pattern recognition and trend reversals
Market Sentiment: Fear & Greed index monitoring

📈 Data Quality

Proper Data Models: Structured with validation and business logic
Realistic Scenarios: Mimics actual trading situations
Consistent Timestamps: Proper time-series data structure
Market Realism: Price movements follow realistic patterns
Comprehensive Coverage: Multiple alert types and market conditions


Note: This is mock data for portfolio demonstration. Real-world implementation would connect to live exchange APIs like Binance, Coinbase, etc.

🛠️ Installation
Prerequisites

Go 1.24.4 or higher
Docker and Docker Compose
Kafka (provided via Docker Compose)
Redis (provided via Docker Compose)
PostgreSQL (provided via Docker Compose)
TimescaleDB (provided via Docker Compose)

Step-by-Step Installation

Clone the Repository
bashgit clone https://github.com/saintparish4/realtime-data-pipeline.git
cd realtime-data-pipeline

Start Infrastructure Services
bashdocker-compose up -d
This will start:

Zookeeper (port 2181)
Kafka (port 9092)
Redis (port 6379) - Real-time caching and rate limiting
PostgreSQL (port 5432) - Alert rules and configurations
TimescaleDB (port 5433) - Time-series metrics storage


Generate Mock Data
bashgo run scripts/generate-mock-data.go
This will create:

15 Alert Rules: Covering various market monitoring scenarios
10,000 Historical Data Points: 1,000 per symbol across 10 cryptocurrencies
25 Sample Alerts: Realistic alert scenarios with proper timing
1 Week of Data: Historical data spanning 168 hours


Build and Run the Application
bash# Build the application
go build -o bin/pipeline ./cmd

# Run with configuration
./bin/pipeline -config cmd/config.yaml

Access the Dashboard
Open your browser and navigate to: http://localhost:8080/dashboard

The application is configured via cmd/config.yaml:
yamlserver:
  port: 8080
  host: "0.0.0.0"

kafka:
  brokers: ["localhost:9092"]
  topics:
    # Observability Event Streaming Topics
    observability_events: "observability.events"
    logs: "observability.logs"
    metrics: "observability.metrics"
    alerts: "observability.alerts"
    traces: "observability.traces"
    health_checks: "observability.health"
    
    # Legacy topics for backward compatibility
    raw_data: "crypto.raw"
    processed_data: "crypto.processed"

redis:
  address: "localhost:6379"
  password: ""
  db: 0

postgres:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  database: "pipeline"

timescaledb:
  host: "localhost"
  port: 5433
  user: "postgres"
  password: "postgres"
  database: "timeseries"

websocket:
  binance_url: "wss://fstream.binance.com:9443/ws/btcusdt@ticker"
  coinbase_url: "wss://ws-feed.pro.coinbase.com"

metrics:
  prometheus_port: 9090

observability:
  metrics_retention: "30d"
  alert_check_interval: "30s"
  notification_rate_limit: "5m"
  cleanup_interval: "24h"
  
  # Event streaming observability settings
  event_streaming:
    correlation_id_header: "X-Correlation-ID"
    trace_id_header: "X-Trace-ID"
    span_id_header: "X-Span-ID"
    service_name: "realtime-data-pipeline"
    enable_distributed_tracing: true
    enable_correlation_tracking: true
    trace_sampling_rate: 0.1
🚀 API Documentation
Health Check

GET /health - Returns system health status and service connectivity

Metrics Endpoints

GET /api/v1/metrics/{symbol} - Get current metrics for a symbol
GET /api/v1/metrics/{symbol}/aggregated - Get aggregated metrics
GET /api/v1/metrics/{symbol}/anomalies - Get detected anomalies
GET /api/v1/metrics/{symbol}/rollups - Get metric rollups

Alert Management

GET /api/v1/alerts - List all alerts
GET /api/v1/alerts/{id} - Get specific alert
POST /api/v1/alerts - Create new alert
PUT /api/v1/alerts/{id} - Update alert
DELETE /api/v1/alerts/{id} - Delete alert
POST /api/v1/alerts/{id}/resolve - Resolve alert

Alert Rules

GET /api/v1/rules - List all alert rules
GET /api/v1/rules/{id} - Get specific rule
POST /api/v1/rules - Create new rule
PUT /api/v1/rules/{id} - Update rule
DELETE /api/v1/rules/{id} - Delete rule

Notifications

GET /api/v1/notifications - List notifications
POST /api/v1/notifications/test - Test notification
PUT /api/v1/notifications/settings - Update settings

Observability

GET /api/v1/observability/traces - Get traces
GET /api/v1/observability/logs - Get logs
GET /api/v1/observability/events - Get events

WebSocket & Dashboard

GET /ws - WebSocket connection for real-time data
GET /dashboard - Web-based dashboard interface

📖 Usage Examples
Creating an Alert Rule
bashcurl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "btc_price_high",
    "name": "BTC Price High Alert",
    "description": "Alert when BTC price exceeds $50,000",
    "symbol": "BTCUSD",
    "condition": "price > 50000",
    "threshold": 50000,
    "severity": "high",
    "enabled": true
  }'
Getting Metrics
bash# Get current metrics for BTC
curl http://localhost:8080/api/v1/metrics/BTCUSD

# Get aggregated metrics
curl http://localhost:8080/api/v1/metrics/BTCUSD/aggregated

# Get detected anomalies
curl http://localhost:8080/api/v1/metrics/BTCUSD/anomalies
Managing Alerts
bash# List all active alerts
curl http://localhost:8080/api/v1/alerts

# Resolve an alert
curl -X POST http://localhost:8080/api/v1/alerts/alert_123/resolve

# Create a new alert
curl -X POST http://localhost:8080/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Volume Spike",
    "description": "Unusual trading volume detected",
    "symbol": "BTCUSD",
    "severity": "medium",
    "condition": "volume > 1000000"
  }'
Testing Notifications
bash# Test email notification
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -d '{"type": "email"}'

# Test Slack notification
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -d '{"type": "slack"}'
WebSocket Connection
javascriptconst ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function() {
    console.log('Connected to WebSocket');
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Received:', data);
};
🧪 Testing & Automation
Automated Testing
bash# Run all tests
go test ./...

# Run specific test packages
go test ./internal/processing/...
go test ./internal/api/...
go test ./internal/notifications/...

# Run anomaly detection tests
go run test-anomaly-volume.go
PowerShell Automation Scripts
The project includes PowerShell scripts for easy Windows development:
powershell# Start the entire pipeline with one command
.\start.ps1

# Stop all services
.\stop.ps1
The start.ps1 script automatically:

Starts all Docker services (Kafka, Redis, PostgreSQL, TimescaleDB)
Waits for services to initialize
Validates database connections
Starts the Go application
Provides access URLs for API, Dashboard, and Health endpoints

Anomaly Detection Testing
The test-anomaly-volume.go file provides comprehensive testing for:

Anomaly Detection Rules: Validates statistical anomaly detection configurations
Volume Analysis Rules: Tests trading volume monitoring and alerting
Alert Storage: Verifies anomaly and volume alerts are properly stored
Data Validation: Ensures realistic mock data generation and storage

Mock Data Validation
bash# Generate and validate mock data
go run scripts/generate-mock-data.go

# Verify data quality and completeness
go run test-anomaly-volume.go
🔍 Metrics Processing Engine
The enhanced Metrics Processing Engine provides advanced analytics capabilities:
Anomaly Detection

Z-Score Method: Identifies outliers based on standard deviations
IQR Method: Uses interquartile range to detect statistical outliers
Statistical Method: Confidence interval-based anomaly detection

Anomaly Types

Price Spikes/Drops: Sudden price movements beyond normal ranges
Volume Spikes/Drops: Unusual trading volume patterns
Volatility Anomalies: Abnormal price volatility
Trend Reversals: Significant trend changes

Severity Levels

Low: Minor deviations that may indicate early warning signs
Medium: Notable deviations requiring attention
High: Significant anomalies that likely require immediate action
Critical: Extreme deviations indicating potential market events

Metric Rollups
Time-series aggregation with comprehensive statistical analysis:

Basic Statistics: Count, sum, min, max, average
Advanced Statistics: Median, standard deviation, coefficient of variation
Percentiles: 95th and 99th percentile calculations
Range Analysis: Price and volume ranges

Advanced Moving Averages

Simple Moving Average (SMA): Equal weight for all data points
Exponential Moving Average (EMA): Weighted average with configurable alpha
Weighted Moving Average (WMA): Custom weights for different time periods

🗄️ Database Layer
Storage Architecture

TimescaleDB: Optimized time-series metrics storage with hypertables
PostgreSQL: Persistent storage for alert rules and configurations
Redis: Real-time caching and rate limiting

TimescaleDB Features

Hypertables: Automatic time-based partitioning
Time Buckets: Built-in time-series aggregations
Compression: Automatic data compression
Continuous Aggregates: Pre-computed aggregations

PostgreSQL Features

Alert Rules: Configurable alert conditions
System Configuration: Centralized configuration management
Historical Data: Complete audit trail
JSON Support: Flexible metadata storage

🔍 Observability & Event Streaming
Event Types

Log Events: Application logs with structured metadata
Metric Events: Time-series metrics and performance indicators
Alert Events: System alerts and threshold violations
Trace Events: Distributed tracing spans for request flows
Health Events: Component health checks and system status

Distributed Tracing

Trace ID: Unique identifier for request flows
Span ID: Individual operations within traces
Correlation ID: Links related events across services
Context Propagation: Automatic trace context passing

Event Correlation

Cross-service correlation: Link events across multiple services
Request flow tracking: Follow complete request journeys
Error correlation: Group related errors and alerts
Performance analysis: Correlate metrics with traces

📈 Monitoring & Observability
Time-Series Metrics

Price metrics: Real-time cryptocurrency price data
Volume metrics: Trading volume tracking
Latency metrics: Performance monitoring
Error metrics: Error tracking and alerting
Custom metrics: User-defined observability data

Alert Management

Configurable alert rules with different severity levels
Real-time alert evaluation based on metric thresholds
Alert state management (Active, Resolved, Silenced)
Rate limiting for alert notifications

System Health

Health checks for service connectivity
Performance monitoring with latency tracking
Error tracking with detailed error metrics
Automatic data cleanup to manage storage

Running Tests
bash# Run unit tests
go test ./...

# Run integration tests
go test ./tests/integration/...

# Run load tests
cd tests/load && npm test
Code Quality
bash# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run security scan
gosec ./...
🚀 Deployment
Development Environment
bash# Start development infrastructure
docker-compose up -d
Production Deployment
Note: Production deployment configurations are planned but not yet implemented.
Planned Docker Deployment
bash# Build Docker image
docker build -t realtime-data-pipeline .

# Run with Docker Compose
docker-compose -f deployments/docker/docker-compose.prod.yml up -d
Planned Kubernetes Deployment
bash# Deploy to Kubernetes
kubectl apply -f deployments/kubernetes/

# Check deployment status
kubectl get pods -n crypto-pipeline
Planned Terraform (AWS) Deployment
bash# Initialize Terraform
cd deployments/terraform
terraform init

# Plan deployment
terraform plan

# Deploy infrastructure
terraform apply
🔒 Security

Input validation and sanitization
Rate limiting on API endpoints
Secure WebSocket connections
Environment-based configuration
Secrets management support

🤝 Contributing
This project is in active development. To contribute:

Fork the repository
Create a feature branch (git checkout -b feature/amazing-feature)
Commit your changes (git commit -m 'Add amazing feature')
Push to the branch (git push origin feature/amazing-feature)
Open a Pull Request

Development Guidelines

Follow Go best practices and conventions
Write comprehensive tests for new features
Update documentation for API changes
Ensure all tests pass before submitting PR
Focus on core functionality first
Keep the architecture simple and scalable

📝 License
This project is licensed under the MIT License - see the LICENSE file for details.
🆘 Support

Issues: GitHub Issues
Documentation: Wiki
Discussions: GitHub Discussions

🙏 Acknowledgments

Kafka for stream processing
Redis for caching and real-time data
PostgreSQL for persistent storage
TimescaleDB for time-series optimization
Binance and Coinbase for market data inspiration