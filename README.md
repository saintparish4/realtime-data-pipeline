# Real-Time Cryptocurrency Data Pipeline

🚀 **PORTFOLIO PROJECT - MOCK DATA DEMONSTRATION** 🚀

A high-performance, scalable real-time data processing pipeline for cryptocurrency market data. This system demonstrates advanced data processing capabilities using realistic mock data for portfolio purposes.

**Current Status**: ✅ **PRODUCTION-READY DEMO** - Complete real-time data processing pipeline with advanced features, comprehensive testing, and automation tools. **Using realistic mock data for demonstration.**

### 🎯 Recent Improvements
- **Enhanced Anomaly Detection**: Advanced statistical methods with confidence scoring
- **Volume Analysis**: Comprehensive trading volume monitoring and alerting
- **Automation Scripts**: PowerShell automation for easy Windows development
- **Testing Suite**: Dedicated anomaly and volume testing tools
- **Mock Data Validation**: Comprehensive data quality assurance
- **Performance Optimization**: Improved processing engine and storage efficiency

## 📊 Mock Data System

This project uses **realistic mock data** for portfolio demonstration purposes. The mock data system includes:

### 🎯 Realistic Data Features
- **10 Major Cryptocurrencies**: BTC, ETH, ADA, SOL, DOT, LINK, UNI, AVAX, MATIC, ATOM
- **Realistic Price Ranges**: Based on actual market prices (e.g., BTC ~$45K, ETH ~$3.2K)
- **Realistic Volume Data**: Market-appropriate trading volumes for each asset
- **Historical Data**: 1,000 data points per symbol over 1 week
- **Market Movements**: Realistic price volatility and volume variations
- **Time-based Patterns**: Proper timestamp distribution and market timing

### 🔔 Alert System Examples
- **Price Breakouts**: BTC breaking above $50K resistance
- **Support Breaks**: ETH dropping below $3K support
- **Volume Spikes**: Unusual trading activity detection
- **Volatility Alerts**: Extreme price movements (SOL +18.5% in 4h)
- **Anomaly Detection**: Statistical anomalies with confidence scores
- **Trend Analysis**: Pattern recognition and trend reversals
- **Market Sentiment**: Fear & Greed index monitoring

### 📈 Data Quality
- **Proper Data Models**: Structured with validation and business logic
- **Realistic Scenarios**: Mimics actual trading situations
- **Consistent Timestamps**: Proper time-series data structure
- **Market Realism**: Price movements follow realistic patterns
- **Comprehensive Coverage**: Multiple alert types and market conditions

### 💡 Portfolio Benefits
- **Demonstrates Technical Skills**: Shows ability to work with complex data structures
- **Realistic Demo Experience**: Visitors see how the system would work with real data
- **Complete Feature Showcase**: All system capabilities are demonstrated
- **Professional Presentation**: Data looks and behaves like real market data
- **Easy to Understand**: Clear examples of what the system monitors

**Note**: This is mock data for portfolio demonstration. Real-world implementation would connect to live exchange APIs like Binance, Coinbase, etc.

## ⚡ Quick Start

### Windows (Recommended)
```powershell
# Clone and start everything with one command
git clone https://github.com/saintparish4/realtime-data-pipeline.git
cd realtime-data-pipeline
.\start.ps1
```

### Linux/macOS
```bash
# Clone and start manually
git clone https://github.com/saintparish4/realtime-data-pipeline.git
cd realtime-data-pipeline
docker-compose up -d
go run scripts/generate-mock-data.go
go run ./cmd -config cmd/config.yaml
```

### Access the Demo
- **Dashboard**: http://localhost:8080/dashboard
- **API**: http://localhost:8080/api/v1/
- **Health**: http://localhost:8080/health

## 🚀 Features

### ✅ Implemented
- **Alert Rule Engine**: Advanced alert rule engine with evaluation logic and threshold management
  - **Configurable Rules**: Create, update, and delete alert rules via REST API
  - **Multiple Conditions**: Support for various conditions (gt, lt, gte, lte, eq, ne)
  - **Severity Levels**: Critical, High, Medium, Low severity classifications
  - **Time Windows**: Configurable evaluation windows and aggregation periods
  - **Rule Validation**: Comprehensive validation and error handling
  - **Anomaly Detection Rules**: Specialized rules for statistical anomaly detection
  - **Volume Analysis Rules**: Trading volume monitoring and alerting

- **Notification System**: Multi-channel notification system with rate limiting and retry logic
  - **Email Notifications**: SMTP-based email alerts with HTML templates
  - **Slack Integration**: Webhook-based Slack notifications with rich formatting
  - **Webhook Support**: Custom webhook endpoints with configurable headers and timeouts
  - **Rate Limiting**: Per-minute and per-hour rate limits to prevent spam
  - **Retry Logic**: Automatic retry with exponential backoff for failed notifications
  - **Notification History**: Track and manage notification delivery status

- **Metrics Collection Endpoints**: Comprehensive REST API for metrics and data access
  - **Real-time Metrics**: Get current metrics for any cryptocurrency symbol
  - **Aggregated Data**: Time-window aggregated metrics with statistical calculations
  - **Anomaly Data**: Access detected anomalies and their confidence scores
  - **Metric Rollups**: Pre-computed aggregations for different time periods
  - **Alert Management**: Full CRUD operations for alerts and alert rules
  - **Health Checks**: System health and service status endpoints

- **Dashboard/Visualization Layer**: Modern web-based dashboard with real-time updates
  - **Real-time Charts**: Live price and volume charts using Chart.js
  - **System Status**: Real-time system health and service status indicators
  - **Alert Display**: Active alerts with severity-based color coding
  - **Metrics Overview**: Key performance indicators and statistics
  - **WebSocket Integration**: Real-time data updates without page refresh
  - **Responsive Design**: Mobile-friendly interface with Tailwind CSS

- **Distributed Tracing Infrastructure**: Comprehensive tracing and observability
  - **Trace Management**: Start, end, and manage distributed traces
  - **Span Operations**: Create and manage spans within traces
  - **Sampling Control**: Configurable sampling rates for performance
  - **Correlation IDs**: Track requests across service boundaries
  - **Trace Storage**: Redis-based trace storage with configurable retention
  - **Trace Search**: Search and filter traces by various criteria
  - **Performance Metrics**: Trace duration and error rate statistics

- **Enhanced Processing Engine**: Advanced metrics processing with anomaly detection
  - **Anomaly Detection**: Z-score, IQR, and statistical methods for detecting price and volume anomalies
  - **Metric Rollups**: Time-series aggregation with statistical calculations (mean, median, percentiles, std dev)
  - **Alerting Thresholds**: Configurable alert rules with multiple conditions and severity levels
  - **Advanced Moving Averages**: Simple, exponential, and weighted moving average calculations
  - **Window Aggregation**: OHLCV (Open, High, Low, Close, Volume) calculations for time windows

- **Storage & Infrastructure**: Robust data storage and infrastructure components
  - **Redis Integration**: Real-time caching and rate limiting for observability data
  - **TimescaleDB Integration**: Optimized time-series metrics storage with hypertables and aggregations
  - **PostgreSQL Integration**: Persistent storage for alert rules, configurations, and historical data
  - **Kafka Event Streaming**: Observability-focused event streaming with distributed tracing and correlation
  - **Observability System**: Comprehensive monitoring, alerting, and metrics collection with hybrid storage
  - **Infrastructure Setup**: Docker Compose configuration for development environment

- **Automation & Testing**: Comprehensive testing and automation tools
  - **PowerShell Scripts**: Automated startup and shutdown scripts for Windows
  - **Anomaly Testing**: Dedicated test suite for anomaly detection and volume analysis
  - **Mock Data Generation**: Realistic data generation for comprehensive testing
  - **Health Checks**: Automated service health monitoring and validation


## 🏗️ Architecture

```
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
```

## 📋 Prerequisites

- Go 1.24.4 or higher
- Docker and Docker Compose
- Kafka (provided via Docker Compose)
- Redis (provided via Docker Compose)
- PostgreSQL (provided via Docker Compose)
- TimescaleDB (provided via Docker Compose)

## 🛠️ Installation

### 1. Clone the Repository

```bash
git clone https://github.com/saintparish4/realtime-data-pipeline.git
cd realtime-data-pipeline
```

### 2. Start Infrastructure Services

```bash
docker-compose up -d
```

This will start:
- **Zookeeper** (port 2181)
- **Kafka** (port 9092)
- **Redis** (port 6379) - Real-time caching and rate limiting
- **PostgreSQL** (port 5432) - Alert rules and configurations
- **TimescaleDB** (port 5433) - Time-series metrics storage

### 3. Generate Mock Data

```bash
# Generate realistic mock data for the demo
go run scripts/generate-mock-data.go
```

This will create:
- **15 Alert Rules**: Covering various market monitoring scenarios
- **10,000 Historical Data Points**: 1,000 per symbol across 10 cryptocurrencies  
- **25 Sample Alerts**: Realistic alert scenarios with proper timing
- **1 Week of Data**: Historical data spanning 168 hours

### 4. Build and Run the Application

```bash
# Build the application
go build -o bin/pipeline ./cmd

# Run with configuration
./bin/pipeline -config cmd/config.yaml
```

### 5. Access the Dashboard

Open your browser and navigate to:
```
http://localhost:8080/dashboard
```

You'll see a fully functional demo with realistic cryptocurrency market data!

## 📖 Usage Examples

### Creating an Alert Rule

```bash
curl -X POST http://localhost:8080/api/v1/rules \
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
```

### Getting Metrics

```bash
# Get current metrics for BTC
curl http://localhost:8080/api/v1/metrics/BTCUSD

# Get aggregated metrics
curl http://localhost:8080/api/v1/metrics/BTCUSD/aggregated

# Get detected anomalies
curl http://localhost:8080/api/v1/metrics/BTCUSD/anomalies
```

### Managing Alerts

```bash
# List all active alerts
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
```

### Testing Notifications

```bash
# Test email notification
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -d '{"type": "email"}'

# Test Slack notification
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -d '{"type": "slack"}'
```

### WebSocket Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function() {
    console.log('Connected to WebSocket');
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Received:', data);
};
```

## 🧪 Testing & Automation

### Automated Testing

Run the comprehensive test suite to verify functionality:

```bash
# Run all tests
go test ./...

# Run specific test packages
go test ./internal/processing/...
go test ./internal/api/...
go test ./internal/notifications/...

# Run anomaly detection tests
go run test-anomaly-volume.go
```

### PowerShell Automation Scripts

The project includes PowerShell scripts for easy Windows development:

#### Quick Start (Windows)
```powershell
# Start the entire pipeline with one command
.\start.ps1

# Stop all services
.\stop.ps1
```

The `start.ps1` script automatically:
- Starts all Docker services (Kafka, Redis, PostgreSQL, TimescaleDB)
- Waits for services to initialize
- Validates database connections
- Starts the Go application
- Provides access URLs for API, Dashboard, and Health endpoints

### Anomaly Detection Testing

The `test-anomaly-volume.go` file provides comprehensive testing for:

- **Anomaly Detection Rules**: Validates statistical anomaly detection configurations
- **Volume Analysis Rules**: Tests trading volume monitoring and alerting
- **Alert Storage**: Verifies anomaly and volume alerts are properly stored
- **Data Validation**: Ensures realistic mock data generation and storage

### Mock Data Validation

```bash
# Generate and validate mock data
go run scripts/generate-mock-data.go

# Verify data quality and completeness
go run test-anomaly-volume.go
```

This ensures:
- **10,000 Historical Data Points**: 1,000 per symbol across 10 cryptocurrencies
- **15 Alert Rules**: Covering various market monitoring scenarios
- **25 Sample Alerts**: Realistic alert scenarios with proper timing
- **1 Week of Data**: Historical data spanning 168 hours

## ⚙️ Configuration

The application is configured via `cmd/config.yaml`:

```yaml
server:
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
```

## 🚀 API Endpoints

The system provides a comprehensive REST API for accessing metrics, managing alerts, and monitoring system health.

### Health Check
```http
GET /health
```
Returns system health status and service connectivity information.

### Metrics Endpoints
```http
GET /api/v1/metrics/{symbol}
GET /api/v1/metrics/{symbol}/aggregated
GET /api/v1/metrics/{symbol}/anomalies
GET /api/v1/metrics/{symbol}/rollups
```

### Alert Management
```http
GET    /api/v1/alerts
GET    /api/v1/alerts/{id}
POST   /api/v1/alerts
PUT    /api/v1/alerts/{id}
DELETE /api/v1/alerts/{id}
POST   /api/v1/alerts/{id}/resolve
```

### Alert Rules
```http
GET    /api/v1/rules
GET    /api/v1/rules/{id}
POST   /api/v1/rules
PUT    /api/v1/rules/{id}
DELETE /api/v1/rules/{id}
```

### Notifications
```http
GET  /api/v1/notifications
POST /api/v1/notifications/test
PUT  /api/v1/notifications/settings
```

### Observability
```http
GET /api/v1/observability/traces
GET /api/v1/observability/logs
GET /api/v1/observability/events
```

### WebSocket
```http
GET /ws
```
Real-time data streaming via WebSocket connection.

### Dashboard
```http
GET /dashboard
```
Web-based dashboard interface for monitoring and visualization.

## 🗄️ Database Layer

### Storage Architecture

- **TimescaleDB**: Optimized time-series metrics storage with hypertables and aggregations
- **PostgreSQL**: Persistent storage for alert rules, configurations, and historical data
- **Redis**: Real-time caching and rate limiting for observability operations

### TimescaleDB Features

- **Hypertables**: Automatic time-based partitioning for efficient querying
- **Time Buckets**: Built-in time-series aggregations (hourly, daily, etc.)
- **Compression**: Automatic data compression for long-term storage
- **Continuous Aggregates**: Pre-computed aggregations for fast queries

### PostgreSQL Features

- **Alert Rules**: Configurable alert conditions and thresholds
- **System Configuration**: Centralized configuration management
- **Historical Data**: Complete audit trail of alerts and notifications
- **JSON Support**: Flexible metadata storage with JSONB columns

### Example Usage

```bash
# Run the enhanced observability example with database layer
cd examples
go run observability-example.go
```

## 🔍 Metrics Processing Engine

The enhanced Metrics Processing Engine provides advanced analytics capabilities for real-time cryptocurrency data:

### Anomaly Detection

The engine implements multiple anomaly detection algorithms to identify unusual patterns in price and volume data:

#### Detection Methods
- **Z-Score Method**: Identifies outliers based on standard deviations from the mean
- **IQR Method**: Uses interquartile range to detect statistical outliers
- **Statistical Method**: Confidence interval-based anomaly detection

#### Anomaly Types
- **Price Spikes/Drops**: Sudden price movements beyond normal ranges
- **Volume Spikes/Drops**: Unusual trading volume patterns
- **Volatility Anomalies**: Abnormal price volatility
- **Trend Reversals**: Significant trend changes

#### Severity Levels
- **Low**: Minor deviations that may indicate early warning signs
- **Medium**: Notable deviations requiring attention
- **High**: Significant anomalies that likely require immediate action
- **Critical**: Extreme deviations indicating potential market events

### Metric Rollups

Time-series aggregation provides comprehensive statistical analysis across multiple time windows:

#### Rollup Windows
- **1 minute**: High-frequency analysis
- **5 minutes**: Short-term patterns
- **15 minutes**: Medium-term trends
- **1 hour**: Hourly analysis
- **4 hours**: Extended time periods
- **24 hours**: Daily aggregations

#### Statistical Calculations
- **Basic Statistics**: Count, sum, min, max, average
- **Advanced Statistics**: Median, standard deviation, coefficient of variation
- **Percentiles**: 95th and 99th percentile calculations
- **Range Analysis**: Price and volume ranges

### Alerting Thresholds

Configurable alert system with flexible threshold evaluation:

#### Alert Conditions
- **Greater than (gt)**: Trigger when value exceeds threshold
- **Less than (lt)**: Trigger when value falls below threshold
- **Greater than or equal (gte)**: Inclusive upper bound
- **Less than or equal (lte)**: Inclusive lower bound
- **Equal (eq)**: Exact value matching
- **Not equal (ne)**: Value deviation

#### Supported Metrics
- **Raw Metrics**: Price, volume
- **Derived Metrics**: 24h changes, percentage changes
- **Moving Averages**: 5m, 15m, 1h moving averages
- **Anomaly Severity**: Anomaly confidence levels

#### Alert Management
- **Active Alerts**: Real-time alert state tracking
- **Alert Resolution**: Automatic and manual alert resolution
- **Rate Limiting**: Prevent alert spam with configurable limits
- **Alert History**: Complete audit trail of all alerts

### Advanced Moving Averages

Enhanced moving average calculations with multiple methods:

#### Calculation Methods
- **Simple Moving Average (SMA)**: Equal weight for all data points
- **Exponential Moving Average (EMA)**: Weighted average with configurable alpha
- **Weighted Moving Average (WMA)**: Custom weights for different time periods

#### Configuration
```yaml
moving_averages:
  - window: "5m"
    method: "simple"
  - window: "15m"
    method: "simple"
  - window: "1h"
    method: "simple"
  - window: "4h"
    method: "exponential"
    alpha: 0.1
```

### Example Usage

```go
// Create metrics engine with enhanced configuration
config := &processing.MetricsEngineConfig{
    MovingAverages: []models.MovingAverageConfig{
        {Window: 5 * time.Minute, Method: "simple"},
        {Window: 1 * time.Hour, Method: "exponential", Alpha: 0.1},
    },
    AnomalyDetection: models.AnomalyDetectionConfig{
        Method:        "zscore",
        Window:        1 * time.Hour,
        Threshold:     2.5,
        MinDataPoints: 10,
    },
    RollupWindows: []time.Duration{
        1 * time.Minute,
        5 * time.Minute,
        1 * time.Hour,
    },
}

engine := processing.NewMetricsEngine(consumer, producer, redisClient, obsService, config)
```

## 🔍 Observability & Event Streaming

The system now includes comprehensive observability capabilities with Kafka event streaming and hybrid storage:

### Event Types
- **Log Events**: Application logs with structured metadata
- **Metric Events**: Time-series metrics and performance indicators
- **Alert Events**: System alerts and threshold violations
- **Trace Events**: Distributed tracing spans for request flows
- **Health Events**: Component health checks and system status

### Distributed Tracing
- **Trace ID**: Unique identifier for request flows
- **Span ID**: Individual operations within traces
- **Correlation ID**: Links related events across services
- **Context Propagation**: Automatic trace context passing

### Event Correlation
- **Cross-service correlation**: Link events across multiple services
- **Request flow tracking**: Follow complete request journeys
- **Error correlation**: Group related errors and alerts
- **Performance analysis**: Correlate metrics with traces

### Example Usage

```bash
# Run the event streaming example
cd examples/event-streaming
go run main.go

# Run the observability example
cd examples
go run observability-example.go
```

For detailed documentation, see [Kafka Event Streaming Documentation](docs/kafka-event-streaming.md).

## 📊 API Endpoints

**Note**: API endpoints are planned but not yet implemented.

### Planned REST API

- `GET /api/v1/price/{symbol}` - Get current price for a symbol
- `GET /api/v1/ohlcv/{symbol}?window=1h` - Get OHLCV data for a time window
- `GET /api/v1/moving-average/{symbol}?period=5m` - Get moving average data
- `GET /api/v1/24h-change/{symbol}` - Get 24-hour price and volume changes
- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics

### Planned WebSocket API

- `ws://localhost:8080/ws/price/{symbol}` - Real-time price updates
- `ws://localhost:8080/ws/ohlcv/{symbol}` - Real-time OHLCV updates

## 🔧 Development

### Project Structure

```
realtime-data-pipeline/
├── cmd/                    # Application entry points
│   └── config.yaml        # Configuration file
├── internal/              # Internal application code
│   ├── api/              # HTTP and WebSocket handlers
│   ├── config/           # Configuration management
│   ├── ingestion/        # Data ingestion services
│   ├── metrics/          # Prometheus metrics
│   ├── models/           # Data models and validation
│   ├── processing/       # Data processing engine
│   ├── storage/          # Data storage interfaces
│   └── observability/    # Observability service with hybrid storage
├── pkg/                  # Reusable packages
│   ├── database/         # Database implementations (TimescaleDB, PostgreSQL)
│   ├── kafka/           # Kafka client utilities
│   ├── redis/           # Redis client utilities
│   └── websocket/       # WebSocket utilities
├── deployments/          # Deployment configurations
│   ├── docker/          # Docker configurations
│   ├── kubernetes/      # Kubernetes manifests
│   └── terraform/       # Infrastructure as Code
├── tests/               # Test suites
│   ├── integration/     # Integration tests
│   ├── load/           # Load testing
│   └── unit/           # Unit tests
└── scripts/             # Utility scripts
```

### Running Tests

```bash
# Run unit tests
go test ./...

# Run integration tests
go test ./tests/integration/...

# Run load tests
cd tests/load && npm test
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run security scan
gosec ./...
```

## 🚀 Deployment

### Development Environment

```bash
# Start development infrastructure
docker-compose up -d
```

### Production Deployment

**Note**: Production deployment configurations are planned but not yet implemented.

#### Planned Docker Deployment

```bash
# Build Docker image
docker build -t realtime-data-pipeline .

# Run with Docker Compose
docker-compose -f deployments/docker/docker-compose.prod.yml up -d
```

#### Planned Kubernetes Deployment

```bash
# Deploy to Kubernetes
kubectl apply -f deployments/kubernetes/

# Check deployment status
kubectl get pods -n crypto-pipeline
```

#### Planned Terraform (AWS) Deployment

```bash
# Initialize Terraform
cd deployments/terraform
terraform init

# Plan deployment
terraform plan

# Deploy infrastructure
terraform apply
```

## 📈 Monitoring & Observability

The application includes comprehensive observability and alerting capabilities built on Redis:

### Time-Series Metrics

- **Price metrics**: Real-time cryptocurrency price data
- **Volume metrics**: Trading volume tracking
- **Latency metrics**: Performance monitoring
- **Error metrics**: Error tracking and alerting
- **Custom metrics**: User-defined observability data

### Alert Management

- **Configurable alert rules** with different severity levels (Info, Warning, Critical)
- **Real-time alert evaluation** based on metric thresholds
- **Alert state management** (Active, Resolved, Silenced)
- **Rate limiting** for alert notifications to prevent spam

### System Health

- **Health checks** for Redis connectivity and alert status
- **Performance monitoring** with latency tracking
- **Error tracking** with detailed error metrics
- **Automatic data cleanup** to manage storage

### Redis-Based Storage

- **Time-series storage** using Redis sorted sets
- **Alert state persistence** with automatic cleanup
- **Rate limiting** for notification management
- **Efficient querying** with time-range support

### Configuration

Configure observability in `cmd/config.yaml`:

```yaml
observability:
  metrics_retention: "30d"        # How long to keep metrics
  alert_check_interval: "30s"     # How often to check alerts
  notification_rate_limit: "5m"   # Rate limit for notifications
  cleanup_interval: "24h"         # How often to cleanup old data
```

### Example Usage

See `examples/observability-example.go` for complete examples of:
- Recording metrics
- Creating alert rules
- Querying time-series data
- Managing alerts
- Performance monitoring

### Documentation

For detailed information, see [Observability Documentation](docs/observability.md).

## 🔒 Security

- Input validation and sanitization
- Rate limiting on API endpoints
- Secure WebSocket connections
- Environment-based configuration
- Secrets management support

## 🤝 Contributing

This project is in early development. Contributions are welcome, but please note:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and conventions
- Write comprehensive tests for new features
- Update documentation for API changes
- Ensure all tests pass before submitting PR
- Focus on core functionality first (data ingestion, processing, storage)
- Keep the architecture simple and scalable

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/saintparish4/realtime-data-pipeline/issues)
- **Documentation**: [Wiki](https://github.com/saintparish4/realtime-data-pipeline/wiki)
- **Discussions**: [GitHub Discussions](https://github.com/saintparish4/realtime-data-pipeline/discussions)

## 🙏 Acknowledgments

- [Kafka](https://kafka.apache.org/) for stream processing
- [Redis](https://redis.io/) for caching and real-time data
- [PostgreSQL](https://www.postgresql.org/) for persistent storage
- [TimescaleDB](https://www.timescale.com/) for time-series optimization
- [Binance](https://www.binance.com/) and [Coinbase](https://www.coinbase.com/) for market data

---

**Built with ❤️ for the cryptocurrency community** 