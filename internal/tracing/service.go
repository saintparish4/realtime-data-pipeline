package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// Service handles distributed tracing operations
type Service struct {
	redisClient *redis.Client
	config      *Config
}

// Config holds tracing service configuration
type Config struct {
	ServiceName       string        `yaml:"service_name"`
	SamplingRate      float64       `yaml:"sampling_rate"`
	MaxTraceDuration  time.Duration `yaml:"max_trace_duration"`
	TraceRetention    time.Duration `yaml:"trace_retention"`
	EnableCorrelation bool          `yaml:"enable_correlation"`
	EnableAggregation bool          `yaml:"enable_aggregation"`
}

// Trace represents a distributed trace
type Trace struct {
	ID            string                 `json:"id"`
	ParentID      string                 `json:"parent_id,omitempty"`
	ServiceName   string                 `json:"service_name"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration,omitempty"`
	Status        TraceStatus            `json:"status"`
	Tags          map[string]string      `json:"tags,omitempty"`
	Logs          []TraceLog             `json:"logs,omitempty"`
	Spans         []*Span                `json:"spans,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Span represents a span within a trace
type Span struct {
	ID            string                 `json:"id"`
	TraceID       string                 `json:"trace_id"`
	ParentID      string                 `json:"parent_id,omitempty"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration,omitempty"`
	Status        SpanStatus             `json:"status"`
	Tags          map[string]string      `json:"tags,omitempty"`
	Logs          []SpanLog              `json:"logs,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// TraceStatus represents the status of a trace
type TraceStatus string

const (
	TraceStatusStarted   TraceStatus = "started"
	TraceStatusCompleted TraceStatus = "completed"
	TraceStatusError     TraceStatus = "error"
	TraceStatusTimeout   TraceStatus = "timeout"
)

// SpanStatus represents the status of a span
type SpanStatus string

const (
	SpanStatusStarted   SpanStatus = "started"
	SpanStatusCompleted SpanStatus = "completed"
	SpanStatusError     SpanStatus = "error"
)

// TraceLog represents a log entry in a trace
type TraceLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// SpanLog represents a log entry in a span
type SpanLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// TraceContext holds trace context information
type TraceContext struct {
	TraceID       string
	SpanID        string
	ParentID      string
	Sampled       bool
	CorrelationID string
}

// NewService creates a new tracing service
func NewService(redisClient *redis.Client, config *Config) *Service {
	if config == nil {
		config = &Config{
			ServiceName:       "realtime-data-pipeline",
			SamplingRate:      0.1,
			MaxTraceDuration:  5 * time.Minute,
			TraceRetention:    7 * 24 * time.Hour,
			EnableCorrelation: true,
			EnableAggregation: true,
		}
	}

	return &Service{
		redisClient: redisClient,
		config:      config,
	}
}

// StartTrace starts a new trace
func (s *Service) StartTrace(ctx context.Context, operationName string, tags map[string]string) (*Trace, *TraceContext) {
	traceID := s.generateTraceID()
	spanID := s.generateSpanID()

	// Check if we should sample this trace
	sampled := s.shouldSample()

	trace := &Trace{
		ID:            traceID,
		ServiceName:   s.config.ServiceName,
		OperationName: operationName,
		StartTime:     time.Now(),
		Status:        TraceStatusStarted,
		Tags:          tags,
		Spans:         []*Span{},
		Metadata:      make(map[string]interface{}),
	}

	traceContext := &TraceContext{
		TraceID:       traceID,
		SpanID:        spanID,
		Sampled:       sampled,
		CorrelationID: s.generateCorrelationID(),
	}

	// Store trace if sampled
	if sampled {
		s.storeTrace(ctx, trace)
	}

	return trace, traceContext
}

// StartSpan starts a new span within a trace
func (s *Service) StartSpan(ctx context.Context, traceContext *TraceContext, operationName string, tags map[string]string) *Span {
	if !traceContext.Sampled {
		return nil
	}

	spanID := s.generateSpanID()
	span := &Span{
		ID:            spanID,
		TraceID:       traceContext.TraceID,
		ParentID:      traceContext.SpanID,
		OperationName: operationName,
		StartTime:     time.Now(),
		Status:        SpanStatusStarted,
		Tags:          tags,
		Logs:          []SpanLog{},
		Metadata:      make(map[string]interface{}),
	}

	// Update trace context
	traceContext.SpanID = spanID

	return span
}

// EndSpan ends a span
func (s *Service) EndSpan(ctx context.Context, span *Span, status SpanStatus, err error) {
	if span == nil {
		return
	}

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = status

	if err != nil {
		span.Tags["error"] = "true"
		span.Tags["error.message"] = err.Error()
		span.Status = SpanStatusError
	}

	// Store span
	s.storeSpan(ctx, span)
}

// EndTrace ends a trace
func (s *Service) EndTrace(ctx context.Context, trace *Trace, status TraceStatus, err error) {
	if trace == nil {
		return
	}

	trace.EndTime = time.Now()
	trace.Duration = trace.EndTime.Sub(trace.StartTime)
	trace.Status = status

	if err != nil {
		trace.Tags["error"] = "true"
		trace.Tags["error.message"] = err.Error()
		trace.Status = TraceStatusError
	}

	// Store updated trace
	s.storeTrace(ctx, trace)
}

// AddTraceLog adds a log entry to a trace
func (s *Service) AddTraceLog(ctx context.Context, trace *Trace, level, message string, fields map[string]interface{}) {
	if trace == nil {
		return
	}

	log := TraceLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	trace.Logs = append(trace.Logs, log)
}

// AddSpanLog adds a log entry to a span
func (s *Service) AddSpanLog(ctx context.Context, span *Span, level, message string, fields map[string]interface{}) {
	if span == nil {
		return
	}

	log := SpanLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	span.Logs = append(span.Logs, log)
}

// GetTrace retrieves a trace by ID
func (s *Service) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	key := fmt.Sprintf("traces:%s", traceID)

	var trace Trace
	if err := s.redisClient.Get(ctx, key, &trace); err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	return &trace, nil
}

// GetTraceSpans retrieves all spans for a trace
func (s *Service) GetTraceSpans(ctx context.Context, traceID string) ([]*Span, error) {
	pattern := fmt.Sprintf("spans:%s:*", traceID)
	keys, err := s.redisClient.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	var spans []*Span
	for _, key := range keys {
		var span Span
		if err := s.redisClient.Get(ctx, key, &span); err == nil {
			spans = append(spans, &span)
		}
	}

	return spans, nil
}

// SearchTraces searches for traces based on criteria
func (s *Service) SearchTraces(ctx context.Context, query *TraceQuery) ([]*Trace, error) {
	// Implementation would search traces based on query criteria
	// For now, return empty array
	return []*Trace{}, nil
}

// TraceQuery represents a trace search query
type TraceQuery struct {
	ServiceName   string            `json:"service_name"`
	OperationName string            `json:"operation_name"`
	Status        TraceStatus       `json:"status"`
	Tags          map[string]string `json:"tags"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	Limit         int               `json:"limit"`
}

// GetTraceStats returns statistics about traces
func (s *Service) GetTraceStats(ctx context.Context, timeRange time.Duration) (map[string]interface{}, error) {
	// Implementation would calculate trace statistics
	// For now, return basic stats
	return map[string]interface{}{
		"total_traces":  0,
		"error_traces":  0,
		"avg_duration":  0,
		"time_range":    timeRange.String(),
		"sampling_rate": s.config.SamplingRate,
	}, nil
}

// CleanupOldTraces removes traces older than the retention period
func (s *Service) CleanupOldTraces(ctx context.Context) error {
	cutoff := time.Now().Add(-s.config.TraceRetention)

	// Find old trace keys
	pattern := "traces:*"
	keys, err := s.redisClient.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	// Remove old traces
	for _, key := range keys {
		var trace Trace
		if err := s.redisClient.Get(ctx, key, &trace); err == nil {
			if trace.StartTime.Before(cutoff) {
				s.redisClient.Expire(ctx, key, 0) // Expire immediately
			}
		}
	}

	// Clean up old spans
	spanPattern := "spans:*"
	spanKeys, err := s.redisClient.Keys(ctx, spanPattern)
	if err != nil {
		return err
	}

	for _, key := range spanKeys {
		var span Span
		if err := s.redisClient.Get(ctx, key, &span); err == nil {
			if span.StartTime.Before(cutoff) {
				s.redisClient.Expire(ctx, key, 0) // Expire immediately
			}
		}
	}

	return nil
}

// storeTrace stores a trace in Redis
func (s *Service) storeTrace(ctx context.Context, trace *Trace) {
	key := fmt.Sprintf("traces:%s", trace.ID)
	s.redisClient.Set(ctx, key, trace, s.config.TraceRetention)
}

// storeSpan stores a span in Redis
func (s *Service) storeSpan(ctx context.Context, span *Span) {
	key := fmt.Sprintf("spans:%s:%s", span.TraceID, span.ID)
	s.redisClient.Set(ctx, key, span, s.config.TraceRetention)
}

// generateTraceID generates a unique trace ID
func (s *Service) generateTraceID() string {
	return fmt.Sprintf("trace_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// generateSpanID generates a unique span ID
func (s *Service) generateSpanID() string {
	return fmt.Sprintf("span_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// generateCorrelationID generates a correlation ID
func (s *Service) generateCorrelationID() string {
	return fmt.Sprintf("corr_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// shouldSample determines if a trace should be sampled
func (s *Service) shouldSample() bool {
	// Simple random sampling based on rate
	return time.Now().UnixNano()%100 < int64(s.config.SamplingRate*100)
}
