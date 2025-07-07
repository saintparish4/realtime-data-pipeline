# Realtime Data Pipeline Shutdown Script
Write-Host "Stopping Realtime Data Pipeline..." -ForegroundColor Green

# Stop the Go application (if running)
Write-Host "Stopping Go application..." -ForegroundColor Yellow
Get-Process -Name "go" -ErrorAction SilentlyContinue | Stop-Process -Force

# Stop Docker services
Write-Host "Stopping Docker services..." -ForegroundColor Yellow
docker-compose down

Write-Host "All services stopped." -ForegroundColor Green 