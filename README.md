# Real-time Cryptocurrency Data Pipeline

A real-time cryptocurrency data pipeline that processes price/volume data, calculates metrics, detects anomalies, and provides live streaming APIs.

## Features

- **Real-time Processing**: Kafka-based event streaming with moving averages and anomaly detection
- **Multi-Database Storage**: PostgreSQL (metadata), TimescaleDB (time-series), Redis (caching)
- **Live APIs**: REST endpoints and WebSocket streaming
- **Alert System**: Configurable rules with email/Slack notifications
- **Observability**: Metrics, tracing, and monitoring
- **Docker Ready**: Complete containerized setup

## Quick Start

```bash
# Clone and start
git clone https://github.com/saintparish4/realtime-data-pipeline.git
cd realtime-data-pipeline
docker-compose up -d

# Generate mock data and run
go run scripts/generate-mock-data.go
go run ./cmd -config cmd/config.yaml
```

**Access:**
- Dashboard: http://localhost:8080/dashboard
- API: http://localhost:8080/api/v1/
- Health: http://localhost:8080/health

## Architecture

```
Exchanges → WebSocket → Kafka → Processing Engine → APIs
                                    ↓
                            Redis + TimescaleDB + PostgreSQL
```


## Current Status

### ✅ Complete
- Backend processing pipeline with Kafka streaming
- Multi-database storage architecture
- REST API with comprehensive endpoints
- Alert system with notifications
- Docker containerization
- Mock data generation system
- Basic observability and monitoring

### 🚧 In Progress / Needs Work
- **Frontend Dashboard**: Currently static HTML placeholder
- **Real Data Sources**: Uses mock data, needs live API integration
- **Testing**: Limited test coverage
- **Documentation**: API docs need expansion

## API Endpoints

**Metrics:**
- `GET /api/v1/metrics/{symbol}` - Current metrics
- `GET /api/v1/metrics/{symbol}/anomalies` - Detected anomalies

**Alerts:**
- `GET /api/v1/alerts` - List alerts
- `POST /api/v1/alerts` - Create alert
- `POST /api/v1/alerts/{id}/resolve` - Resolve alert

**WebSocket:**
- `GET /ws` - Real-time data streaming

## Technologies

- **Backend**: Go, Gin, Gorilla WebSocket
- **Databases**: PostgreSQL, TimescaleDB, Redis
- **Streaming**: Apache Kafka, Zookeeper
- **Infrastructure**: Docker, Docker Compose
- **Monitoring**: Custom observability system

## Improvement Roadmap

### High Priority (1-2 weeks)
1. **Modern Dashboard**: Replace static HTML with React/Next.js
2. **Real Data Integration**: Connect to live crypto APIs (Binance, CoinGecko)
3. **Testing**: Add unit and integration tests
4. **API Documentation**: Swagger/OpenAPI specs

### Medium Priority (2-4 weeks)
1. **Advanced Analytics**: ML models, technical indicators
2. **Authentication**: JWT auth, API keys
3. **CI/CD**: GitHub Actions pipeline
4. **Monitoring**: Prometheus/Grafana integration

### Low Priority (1-2 months)
1. **Kubernetes**: Production deployment manifests
2. **Security**: Data encryption, rate limiting
3. **Performance**: Caching optimization, load balancing

## Prerequisites

- Go 1.24.4+
- Docker & Docker Compose
- 4GB+ RAM (for all services)

## License

MIT License