# Realtime Data Pipeline Startup Script
Write-Host "Starting Realtime Data Pipeline..." -ForegroundColor Green

# Step 1: Start Docker services
Write-Host "Starting Docker services..." -ForegroundColor Yellow
docker-compose up -d

# Step 2: Wait for services to initialize
Write-Host "Waiting for services to initialize..." -ForegroundColor Yellow
Start-Sleep -Seconds 15

# Step 3: Check if services are running
Write-Host "Checking service status..." -ForegroundColor Yellow
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Step 4: Test database connections
Write-Host "Testing database connections..." -ForegroundColor Yellow
docker exec realtime-data-pipeline-postgres-1 psql -U postgres -d pipeline -c "SELECT 'PostgreSQL OK' as status;" 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "PostgreSQL: Connected" -ForegroundColor Green
} else {
    Write-Host "PostgreSQL: Failed" -ForegroundColor Red
}

docker exec realtime-data-pipeline-timescaledb-1 psql -U postgres -d timeseries -c "SELECT 'TimescaleDB OK' as status;" 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "TimescaleDB: Connected" -ForegroundColor Green
} else {
    Write-Host "TimescaleDB: Failed" -ForegroundColor Red
}

# Step 5: Start the Go application
Write-Host "Starting Go application..." -ForegroundColor Yellow
Write-Host "Application will be available at:" -ForegroundColor Cyan
Write-Host "  - API: http://localhost:8080/api/v1/" -ForegroundColor Cyan
Write-Host "  - Dashboard: http://localhost:8080/dashboard" -ForegroundColor Cyan
Write-Host "  - Health: http://localhost:8080/health" -ForegroundColor Cyan
Write-Host ""
Write-Host "Press Ctrl+C to stop the application" -ForegroundColor Yellow

go run ./cmd -config cmd/config.yaml 