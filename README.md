# Real-Time Cryptocurrency Data Pipeline

🚧 **WORK IN PROGRESS** 🚧

A high-performance, scalable real-time data processing pipeline for cryptocurrency market data. This system is currently under active development and will eventually ingest live cryptocurrency data from multiple exchanges, process it with advanced analytics, and provide real-time insights through REST APIs and WebSocket connections.

**Current Status**: Core processing engine and window aggregation implemented. Infrastructure setup and basic data models in place.

## 🚀 Features

### ✅ Implemented
- **Data Processing Engine**: Core processing logic with moving averages and 24-hour change calculations
- **Window Aggregation**: OHLCV (Open, High, Low, Close, Volume) calculations for time windows
- **Redis Integration**: Time-series data storage with efficient range queries
- **Kafka Integration**: Producer and consumer utilities for stream processing
- **Infrastructure Setup**: Docker Compose configuration for development environment

### 🚧 In Progress
- **WebSocket Ingestion**: Real-time data ingestion from cryptocurrency exchanges
- **REST API**: HTTP endpoints for data retrieval and analytics
- **WebSocket API**: Real-time data streaming to clients
- **PostgreSQL Integration**: Historical data storage
- **Monitoring**: Prometheus metrics and health checks

### 📋 Planned
- **Exchange Integration**: Binance and Coinbase WebSocket connections
- **Advanced Analytics**: Additional technical indicators
- **Scalable Architecture**: Kubernetes and Terraform deployment options
- **Load Testing**: Performance validation and optimization

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
┌─────────────────┐    ┌─────────────────┐
│   Redis         │    │   PostgreSQL    │
│   (Real-time)   │    │   (Historical)  │
└─────────────────┘    └─────────────────┘
```

## 📋 Prerequisites

- Go 1.24.4 or higher
- Docker and Docker Compose
- Kafka (provided via Docker Compose)
- Redis (provided via Docker Compose)
- PostgreSQL (provided via Docker Compose)

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
- **Redis** (port 6379)
- **PostgreSQL** (port 5432)
- **TimescaleDB** (port 5433)

### 3. Build and Run the Application

```bash
# Build the application
go build -o bin/pipeline ./cmd

# Run with configuration
./bin/pipeline -config cmd/config.yaml
```

**Note**: The main application entry point is still under development. Currently, you can run individual components for testing:

```bash
# Run tests to verify current functionality
go test ./internal/processing/...
```

## ⚙️ Configuration

The application is configured via `cmd/config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

kafka:
  brokers: ["localhost:9092"]
  topics:
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

websocket:
  binance_url: "wss://fstream.binance.com:9443/ws/btcusdt@ticker"
  coinbase_url: "wss://ws-feed.pro.coinbase.com"

metrics:
  prometheus_port: 9090
```

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
│   └── storage/          # Data storage interfaces
├── pkg/                  # Reusable packages
│   ├── database/         # Database utilities
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

## 📈 Monitoring

**Note**: Monitoring and metrics are planned but not yet implemented.

### Planned Prometheus Metrics

The application will expose Prometheus metrics at `/metrics`:

- `crypto_price_current` - Current price for each symbol
- `crypto_volume_24h` - 24-hour volume
- `crypto_trades_total` - Total number of trades processed
- `processing_duration_seconds` - Data processing latency
- `kafka_messages_consumed` - Kafka message consumption rate

### Planned Health Checks

- `GET /health` - Application health status
- `GET /health/ready` - Readiness probe
- `GET /health/live` - Liveness probe

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