package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/saintparish4/realtime-data-pipeline/internal/config"
	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/internal/observability"
	"github.com/saintparish4/realtime-data-pipeline/internal/processing"
	"github.com/saintparish4/realtime-data-pipeline/pkg/database"
	"github.com/saintparish4/realtime-data-pipeline/pkg/kafka"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// Server represents the main API server
type Server struct {
	config        *config.Config
	router        *gin.Engine
	httpServer    *http.Server
	redisClient   *redis.Client
	postgresDB    *database.PostgreSQL
	timescaleDB   *database.TimescaleDB
	kafkaProducer *kafka.Producer
	kafkaConsumer *kafka.Consumer
	alertManager  *processing.AlertManager
	metricsEngine *processing.MetricsEngine
	observability *observability.Service
	upgrader      websocket.Upgrader
	wsConnections map[string]*websocket.Conn

	// Multiplexing and batching fields
	subscriptions map[*websocket.Conn]map[string]bool // client -> set of symbols
	subMu         sync.RWMutex
	batchInterval time.Duration
	batchChan     chan *priceUpdate
	shutdownChan  chan struct{}

	clientSendChans map[*websocket.Conn]chan []*priceUpdate // NEW: per-client send channel
}

type subscriptionRequest struct {
	Action  string   `json:"action"`
	Symbols []string `json:"symbols"`
}

type priceUpdate struct {
	Symbol    string      `json:"symbol"`
	Price     interface{} `json:"price"`
	Volume    interface{} `json:"volume"`
	Timestamp interface{} `json:"timestamp"`
	Source    string      `json:"source"`
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config, redisClient *redis.Client, postgresDB *database.PostgreSQL, timescaleDB *database.TimescaleDB, kafkaProducer *kafka.Producer, kafkaConsumer *kafka.Consumer, obsService *observability.Service, metricsEngine *processing.MetricsEngine, alertManager *processing.AlertManager) (*Server, error) {

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Setup WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	server := &Server{
		config:          cfg,
		router:          router,
		redisClient:     redisClient,
		postgresDB:      postgresDB,
		timescaleDB:     timescaleDB,
		kafkaProducer:   kafkaProducer,
		kafkaConsumer:   kafkaConsumer,
		alertManager:    alertManager,
		metricsEngine:   metricsEngine,
		observability:   obsService,
		upgrader:        upgrader,
		wsConnections:   make(map[string]*websocket.Conn),
		subscriptions:   make(map[*websocket.Conn]map[string]bool),
		subMu:           sync.RWMutex{},
		batchInterval:   50 * time.Millisecond, // tune as needed
		batchChan:       make(chan *priceUpdate, 10000),
		shutdownChan:    make(chan struct{}),
		clientSendChans: make(map[*websocket.Conn]chan []*priceUpdate), // NEW
	}

	// Setup routes
	server.setupRoutes()

	// Setup HTTP server
	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start Kafka consumer batcher
	go server.startKafkaBatcher(kafkaConsumer)

	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Health check endpoint
		v1.GET("/health", s.healthCheck)

		// Metrics endpoints
		metrics := v1.Group("/metrics")
		{
			metrics.GET("/:symbol", s.getMetrics)
			metrics.GET("/:symbol/aggregated", s.getAggregatedMetrics)
			metrics.GET("/:symbol/anomalies", s.getAnomalies)
			metrics.GET("/:symbol/rollups", s.getMetricRollups)
		}

		// Alerts endpoints
		alerts := v1.Group("/alerts")
		{
			alerts.GET("/", s.getAlerts)
			alerts.GET("/:id", s.getAlert)
			alerts.POST("/", s.createAlert)
			alerts.PUT("/:id", s.updateAlert)
			alerts.DELETE("/:id", s.deleteAlert)
			alerts.POST("/:id/resolve", s.resolveAlert)
		}

		// Alert rules endpoints
		rules := v1.Group("/rules")
		{
			rules.GET("/", s.getAlertRules)
			rules.GET("/:id", s.getAlertRule)
			rules.POST("/", s.createAlertRule)
			rules.PUT("/:id", s.updateAlertRule)
			rules.DELETE("/:id", s.deleteAlertRule)
		}

		// Notifications endpoints
		notifications := v1.Group("/notifications")
		{
			notifications.GET("/", s.getNotifications)
			notifications.POST("/test", s.testNotification)
			notifications.PUT("/settings", s.updateNotificationSettings)
		}

		// Observability endpoints
		obs := v1.Group("/observability")
		{
			obs.GET("/traces", s.getTraces)
			obs.GET("/logs", s.getLogs)
			obs.GET("/events", s.getEvents)
		}
	}

	// WebSocket endpoint for real-time data
	s.router.GET("/ws", s.handleWebSocket)

	// Dashboard static files
	s.router.Static("/dashboard", "./web/dashboard")
	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/dashboard")
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s:%d", s.config.Server.Host, s.config.Server.Port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down API server...")

	// Close WebSocket connections
	for _, conn := range s.wsConnections {
		conn.Close()
	}

	// Close database connections
	if s.postgresDB != nil {
		s.postgresDB.Close()
	}
	if s.timescaleDB != nil {
		s.timescaleDB.Close()
	}

	// Close Kafka connections
	if s.kafkaProducer != nil {
		s.kafkaProducer.Close()
	}
	if s.kafkaConsumer != nil {
		s.kafkaConsumer.Close()
	}

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"services": map[string]string{
			"redis":     "connected",
			"postgres":  "connected",
			"timescale": "connected",
			"kafka":     "connected",
		},
	}

	c.JSON(http.StatusOK, health)
}

// ===== METRICS ENDPOINTS =====

func (s *Server) getMetrics(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol parameter is required"})
		return
	}

	// Get metrics from observability service
	query := &models.TimeSeriesQuery{
		Symbol: symbol,
		Limit:  100,
	}

	metrics, err := s.observability.GetMetrics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure we always return an array, even if it's empty
	if metrics == nil {
		metrics = []*models.TimeSeriesMetric{}
	}

	// Force JSON marshaling to treat empty slice as array, not null
	if len(metrics) == 0 {
		c.JSON(http.StatusOK, []*models.TimeSeriesMetric{})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getAggregatedMetrics(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol parameter is required"})
		return
	}

	// Get aggregated metrics
	query := &models.MetricQuery{
		Symbol: symbol,
		Limit:  100,
	}

	metrics, err := s.observability.GetAggregatedMetrics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure we always return an array, even if it's empty
	if metrics == nil {
		metrics = []*models.MetricAggregation{}
	}

	// Force JSON marshaling to treat empty slice as array, not null
	if len(metrics) == 0 {
		c.JSON(http.StatusOK, []*models.MetricAggregation{})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getAnomalies(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol parameter is required"})
		return
	}

	// Get anomalies from Redis
	key := fmt.Sprintf("anomalies:%s:*", symbol)
	keys, err := s.redisClient.Keys(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var anomalies []models.AnomalyDetectionResult
	for range keys {
		// Implementation would fetch anomaly data from Redis
		// For now, return empty array
	}

	c.JSON(http.StatusOK, anomalies)
}

func (s *Server) getMetricRollups(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol parameter is required"})
		return
	}

	// Get metric rollups from Redis
	key := fmt.Sprintf("rollups:%s:*", symbol)
	keys, err := s.redisClient.Keys(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var rollups []models.MetricRollup
	for range keys {
		// Implementation would fetch rollup data from Redis
		// For now, return empty array
	}

	c.JSON(http.StatusOK, rollups)
}

// ===== ALERTS ENDPOINTS =====

func (s *Server) getAlerts(c *gin.Context) {
	alerts, err := s.observability.GetActiveAlerts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure we always return an array, even if it's empty
	if alerts == nil {
		alerts = []*models.Alert{}
	}

	// Force JSON marshaling to treat empty slice as array, not null
	if len(alerts) == 0 {
		c.JSON(http.StatusOK, []*models.Alert{})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

func (s *Server) getAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alert ID is required"})
		return
	}

	alert, err := s.redisClient.GetAlert(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (s *Server) createAlert(c *gin.Context) {
	var alert models.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := alert.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.redisClient.StoreAlert(c.Request.Context(), &alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

func (s *Server) updateAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alert ID is required"})
		return
	}

	var alert models.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert.ID = alertID
	if err := s.redisClient.StoreAlert(c.Request.Context(), &alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (s *Server) deleteAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alert ID is required"})
		return
	}

	if err := s.redisClient.DeleteAlert(c.Request.Context(), alertID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert deleted"})
}

func (s *Server) resolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alert ID is required"})
		return
	}

	if err := s.observability.ResolveAlert(c.Request.Context(), alertID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert resolved"})
}

// ===== ALERT RULES ENDPOINTS =====

func (s *Server) getAlertRules(c *gin.Context) {
	rules := s.observability.GetAlertRules()

	// Ensure we always return an array, even if it's empty
	if rules == nil {
		rules = []*observability.AlertRule{}
	}

	// Force JSON marshaling to treat empty slice as array, not null
	if len(rules) == 0 {
		c.JSON(http.StatusOK, []*observability.AlertRule{})
		return
	}

	c.JSON(http.StatusOK, rules)
}

func (s *Server) getAlertRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule ID is required"})
		return
	}

	rules := s.observability.GetAlertRules()
	for _, rule := range rules {
		if rule.ID == ruleID {
			c.JSON(http.StatusOK, rule)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
}

func (s *Server) createAlertRule(c *gin.Context) {
	var rule observability.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.observability.AddAlertRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

func (s *Server) updateAlertRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule ID is required"})
		return
	}

	var rule observability.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = ruleID
	// Remove old rule and add new one
	s.observability.RemoveAlertRule(ruleID)
	if err := s.observability.AddAlertRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (s *Server) deleteAlertRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule ID is required"})
		return
	}

	if err := s.observability.RemoveAlertRule(ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule deleted"})
}

// ===== NOTIFICATIONS ENDPOINTS =====

func (s *Server) getNotifications(c *gin.Context) {
	// Implementation would fetch notification history
	c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) testNotification(c *gin.Context) {
	// Implementation would send a test notification
	c.JSON(http.StatusOK, gin.H{"message": "test notification sent"})
}

func (s *Server) updateNotificationSettings(c *gin.Context) {
	// Implementation would update notification settings
	c.JSON(http.StatusOK, gin.H{"message": "notification settings updated"})
}

// ===== OBSERVABILITY ENDPOINTS =====

func (s *Server) getTraces(c *gin.Context) {
	// Implementation would fetch distributed traces
	c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) getLogs(c *gin.Context) {
	// Implementation would fetch logs
	c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) getEvents(c *gin.Context) {
	// Implementation would fetch events
	c.JSON(http.StatusOK, []interface{}{})
}

// ===== WEBSOCKET HANDLER =====

// WebSocket handler with multiplexing and batching
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	connID := fmt.Sprintf("ws_%d", time.Now().UnixNano())
	s.wsConnections[connID] = conn

	s.subMu.Lock()
	s.subscriptions[conn] = make(map[string]bool)
	clientSend := make(chan []*priceUpdate, 10)
	s.clientSendChans[conn] = clientSend // NEW
	s.subMu.Unlock()

	go s.clientWriter(conn, clientSend)

	defer func() {
		s.subMu.Lock()
		delete(s.subscriptions, conn)
		delete(s.clientSendChans, conn) // NEW
		s.subMu.Unlock()
		delete(s.wsConnections, connID)
		close(clientSend)
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		// Parse subscription request
		var req subscriptionRequest
		if err := json.Unmarshal(message, &req); err != nil {
			log.Printf("Invalid subscription message: %v", err)
			continue
		}
		s.handleSubscription(conn, req)
	}
}

func (s *Server) handleSubscription(conn *websocket.Conn, req subscriptionRequest) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	if req.Action == "subscribe" {
		if s.subscriptions[conn] == nil {
			s.subscriptions[conn] = make(map[string]bool)
		}
		for _, sym := range req.Symbols {
			s.subscriptions[conn][sym] = true
		}
	} else if req.Action == "unsubscribe" {
		for _, sym := range req.Symbols {
			delete(s.subscriptions[conn], sym)
		}
	}
}

// Batcher: consumes Kafka, batches, and broadcasts
func (s *Server) startKafkaBatcher(consumer *kafka.Consumer) {
	ctx := context.Background()
	batch := make([]*priceUpdate, 0, 100)
	ticker := time.NewTicker(s.batchInterval)
	defer ticker.Stop()

	// Start Kafka consumer in background
	go func() {
		consumer.ConsumeEvents(ctx, func(_ context.Context, event *kafka.ObservabilityEvent) error {
			if event.Type != "metric" && event.Type != "raw" {
				return nil
			}
			// Unmarshal event.Data as RawCryptoData
			var raw models.RawCryptoData
			dataBytes, err := json.Marshal(event.Data)
			if err != nil {
				return nil
			}
			if err := json.Unmarshal(dataBytes, &raw); err != nil {
				return nil
			}
			update := &priceUpdate{
				Symbol:    raw.Symbol,
				Price:     raw.Price,
				Volume:    raw.Volume,
				Timestamp: raw.Timestamp,
				Source:    raw.Source,
			}
			s.batchChan <- update
			return nil
		})
	}()

	for {
		select {
		case <-s.shutdownChan:
			return
		case <-ticker.C:
			if len(batch) > 0 {
				s.broadcastBatch(batch)
				batch = batch[:0]
			}
		case upd := <-s.batchChan:
			batch = append(batch, upd)
			if len(batch) >= 1000 {
				s.broadcastBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// Broadcast batch to all clients based on their subscriptions
func (s *Server) broadcastBatch(batch []*priceUpdate) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()
	if len(batch) == 0 {
		return
	}
	// Group updates by symbol
	symbolMap := make(map[string][]*priceUpdate)
	for _, upd := range batch {
		symbolMap[upd.Symbol] = append(symbolMap[upd.Symbol], upd)
	}
	for conn, subs := range s.subscriptions {
		var clientBatch []*priceUpdate
		for sym := range subs {
			clientBatch = append(clientBatch, symbolMap[sym]...)
		}
		if len(clientBatch) > 0 {
			// Non-blocking send
			select {
			case s.getClientSendChan(conn) <- clientBatch:
			default:
				// Drop if client is slow
			}
		}
	}
}

// Helper to get the send channel for a client
func (s *Server) getClientSendChan(conn *websocket.Conn) chan []*priceUpdate {
	s.subMu.RLock()
	defer s.subMu.RUnlock()
	return s.clientSendChans[conn]
}

// Client writer goroutine
func (s *Server) clientWriter(conn *websocket.Conn, send <-chan []*priceUpdate) {
	for batch := range send {
		if len(batch) == 0 {
			continue
		}
		msg, err := json.Marshal(batch)
		if err != nil {
			continue
		}
		conn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}
