package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/saintparish4/realtime-data-pipeline/internal/models"
	"github.com/saintparish4/realtime-data-pipeline/pkg/redis"
)

// Service handles all notification operations including email, Slack, and webhooks
type Service struct {
	redisClient *redis.Client
	config      *Config
	rateLimiter *RateLimiter
}

// Config holds notification service configuration
type Config struct {
	Email struct {
		SMTPHost    string `yaml:"smtp_host"`
		SMTPPort    int    `yaml:"smtp_port"`
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
		FromAddress string `yaml:"from_address"`
		FromName    string `yaml:"from_name"`
		UseTLS      bool   `yaml:"use_tls"`
	} `yaml:"email"`

	Slack struct {
		WebhookURL string `yaml:"webhook_url"`
		Channel    string `yaml:"channel"`
		Username   string `yaml:"username"`
		IconEmoji  string `yaml:"icon_emoji"`
	} `yaml:"slack"`

	Webhooks []WebhookConfig `yaml:"webhooks"`

	RateLimit struct {
		MaxPerMinute int           `yaml:"max_per_minute"`
		MaxPerHour   int           `yaml:"max_per_hour"`
		Window       time.Duration `yaml:"window"`
	} `yaml:"rate_limit"`

	Retry struct {
		MaxAttempts int           `yaml:"max_attempts"`
		Backoff     time.Duration `yaml:"backoff"`
	} `yaml:"retry"`
}

// WebhookConfig defines a webhook endpoint configuration
type WebhookConfig struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Timeout time.Duration     `yaml:"timeout"`
	Enabled bool              `yaml:"enabled"`
}

// NotificationRequest represents a notification to be sent
type NotificationRequest struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Recipients  []string               `json:"recipients"`
	Subject     string                 `json:"subject"`
	Message     string                 `json:"message"`
	Alert       *models.Alert          `json:"alert,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Priority    Priority               `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail   NotificationType = "email"
	NotificationTypeSlack   NotificationType = "slack"
	NotificationTypeWebhook NotificationType = "webhook"
)

// Priority represents notification priority
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// NotificationResult represents the result of sending a notification
type NotificationResult struct {
	ID       string        `json:"id"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	SentAt   time.Time     `json:"sent_at"`
	Attempts int           `json:"attempts"`
	Duration time.Duration `json:"duration"`
}

// NewService creates a new notification service
func NewService(redisClient *redis.Client, config *Config) *Service {
	if config == nil {
		config = &Config{}
	}

	// Set defaults
	if config.RateLimit.MaxPerMinute == 0 {
		config.RateLimit.MaxPerMinute = 60
	}
	if config.RateLimit.MaxPerHour == 0 {
		config.RateLimit.MaxPerHour = 1000
	}
	if config.RateLimit.Window == 0 {
		config.RateLimit.Window = time.Minute
	}
	if config.Retry.MaxAttempts == 0 {
		config.Retry.MaxAttempts = 3
	}
	if config.Retry.Backoff == 0 {
		config.Retry.Backoff = time.Second * 5
	}

	return &Service{
		redisClient: redisClient,
		config:      config,
		rateLimiter: NewRateLimiter(redisClient, config.RateLimit),
	}
}

// SendNotification sends a notification based on the request type
func (s *Service) SendNotification(ctx context.Context, req *NotificationRequest) (*NotificationResult, error) {
	startTime := time.Now()

	// Check rate limits
	if err := s.rateLimiter.CheckLimit(ctx, req.Type, req.Priority); err != nil {
		return &NotificationResult{
			ID:      req.ID,
			Success: false,
			Error:   fmt.Sprintf("rate limit exceeded: %v", err),
			SentAt:  time.Now(),
		}, err
	}

	// Send based on type
	var err error
	var attempts int

	for attempts = 1; attempts <= s.config.Retry.MaxAttempts; attempts++ {
		switch req.Type {
		case NotificationTypeEmail:
			err = s.sendEmail(ctx, req)
		case NotificationTypeSlack:
			err = s.sendSlack(ctx, req)
		case NotificationTypeWebhook:
			err = s.sendWebhook(ctx, req)
		default:
			err = fmt.Errorf("unsupported notification type: %s", req.Type)
		}

		if err == nil {
			break
		}

		// Wait before retry
		if attempts < s.config.Retry.MaxAttempts {
			time.Sleep(s.config.Retry.Backoff * time.Duration(attempts))
		}
	}

	result := &NotificationResult{
		ID:      req.ID,
		Success: err == nil,
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		SentAt:   time.Now(),
		Attempts: attempts,
		Duration: time.Since(startTime),
	}

	// Store result
	s.storeNotificationResult(ctx, result)

	return result, err
}

// sendEmail sends an email notification
func (s *Service) sendEmail(ctx context.Context, req *NotificationRequest) error {
	if s.config.Email.SMTPHost == "" {
		return fmt.Errorf("SMTP host not configured")
	}

	// Prepare email content
	emailContent, err := s.prepareEmailContent(req)
	if err != nil {
		return fmt.Errorf("failed to prepare email content: %w", err)
	}

	// Setup SMTP authentication
	auth := smtp.PlainAuth("", s.config.Email.Username, s.config.Email.Password, s.config.Email.SMTPHost)

	// Send email to each recipient
	for _, recipient := range req.Recipients {
		to := []string{recipient}
		msg := []byte(emailContent)

		addr := fmt.Sprintf("%s:%d", s.config.Email.SMTPHost, s.config.Email.SMTPPort)
		if err := smtp.SendMail(addr, auth, s.config.Email.FromAddress, to, msg); err != nil {
			return fmt.Errorf("failed to send email to %s: %w", recipient, err)
		}
	}

	return nil
}

// sendSlack sends a Slack notification
func (s *Service) sendSlack(ctx context.Context, req *NotificationRequest) error {
	if s.config.Slack.WebhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}

	// Prepare Slack message
	slackMessage := s.prepareSlackMessage(req)

	// Send to Slack
	jsonData, err := json.Marshal(slackMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	resp, err := http.Post(s.config.Slack.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status: %d", resp.StatusCode)
	}

	return nil
}

// sendWebhook sends a webhook notification
func (s *Service) sendWebhook(ctx context.Context, req *NotificationRequest) error {
	for _, webhook := range s.config.Webhooks {
		if !webhook.Enabled {
			continue
		}

		if err := s.sendSingleWebhook(ctx, req, webhook); err != nil {
			return fmt.Errorf("failed to send webhook %s: %w", webhook.Name, err)
		}
	}

	return nil
}

// sendSingleWebhook sends a notification to a single webhook endpoint
func (s *Service) sendSingleWebhook(ctx context.Context, req *NotificationRequest, webhook WebhookConfig) error {
	// Prepare webhook payload
	payload := map[string]interface{}{
		"id":        req.ID,
		"type":      string(req.Type),
		"subject":   req.Subject,
		"message":   req.Message,
		"priority":  string(req.Priority),
		"timestamp": req.CreatedAt,
		"alert":     req.Alert,
		"metadata":  req.Metadata,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range webhook.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set timeout
	client := &http.Client{
		Timeout: webhook.Timeout,
	}

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

// prepareEmailContent prepares the email content with HTML template
func (s *Service) prepareEmailContent(req *NotificationRequest) (string, error) {
	tmpl := `
From: {{.FromName}} <{{.FromAddress}}>
To: {{.Recipients}}
Subject: {{.Subject}}
MIME-Version: 1.0
Content-Type: text/html; charset=UTF-8

<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
</head>
<body>
    <h2>{{.Subject}}</h2>
    <p><strong>Priority:</strong> {{.Priority}}</p>
    <p><strong>Time:</strong> {{.Timestamp}}</p>
    
    <div style="margin: 20px 0;">
        {{.Message}}
    </div>
    
    {{if .Alert}}
    <div style="background-color: #f8f9fa; padding: 15px; border-left: 4px solid #007bff; margin: 20px 0;">
        <h3>Alert Details</h3>
        <p><strong>Symbol:</strong> {{.Alert.Symbol}}</p>
        <p><strong>Severity:</strong> {{.Alert.Severity}}</p>
        <p><strong>Condition:</strong> {{.Alert.Condition}}</p>
        <p><strong>Current Value:</strong> {{.Alert.CurrentValue}}</p>
        <p><strong>Threshold:</strong> {{.Alert.Threshold}}</p>
    </div>
    {{end}}
    
    <hr>
    <p style="color: #6c757d; font-size: 12px;">
        This notification was sent by the Real-Time Data Pipeline system.
    </p>
</body>
</html>`

	data := struct {
		FromName    string
		FromAddress string
		Recipients  string
		Subject     string
		Message     string
		Priority    string
		Timestamp   string
		Alert       *models.Alert
	}{
		FromName:    s.config.Email.FromName,
		FromAddress: s.config.Email.FromAddress,
		Recipients:  strings.Join(req.Recipients, ", "),
		Subject:     req.Subject,
		Message:     req.Message,
		Priority:    string(req.Priority),
		Timestamp:   req.CreatedAt.Format(time.RFC3339),
		Alert:       req.Alert,
	}

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// prepareSlackMessage prepares the Slack message
func (s *Service) prepareSlackMessage(req *NotificationRequest) map[string]interface{} {
	// Determine color based on priority
	color := "#36a64f" // green
	switch req.Priority {
	case PriorityHigh:
		color = "#ff9500" // orange
	case PriorityCritical:
		color = "#ff0000" // red
	}

	// Create Slack message structure
	message := map[string]interface{}{
		"channel":    s.config.Slack.Channel,
		"username":   s.config.Slack.Username,
		"icon_emoji": s.config.Slack.IconEmoji,
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     req.Subject,
				"text":      req.Message,
				"fields":    []map[string]interface{}{},
				"timestamp": req.CreatedAt.Unix(),
			},
		},
	}

	// Add fields
	fields := message["attachments"].([]map[string]interface{})[0]["fields"].([]map[string]interface{})

	fields = append(fields, map[string]interface{}{
		"title": "Priority",
		"value": string(req.Priority),
		"short": true,
	})

	if req.Alert != nil {
		fields = append(fields, map[string]interface{}{
			"title": "Symbol",
			"value": req.Alert.Symbol,
			"short": true,
		})
		fields = append(fields, map[string]interface{}{
			"title": "Severity",
			"value": string(req.Alert.Severity),
			"short": true,
		})
	}

	message["attachments"].([]map[string]interface{})[0]["fields"] = fields

	return message
}

// storeNotificationResult stores the notification result in Redis
func (s *Service) storeNotificationResult(ctx context.Context, result *NotificationResult) {
	key := fmt.Sprintf("notifications:results:%s", result.ID)
	s.redisClient.Set(ctx, key, result, 24*time.Hour)
}

// GetNotificationHistory retrieves notification history
func (s *Service) GetNotificationHistory(ctx context.Context, limit int) ([]*NotificationResult, error) {
	pattern := "notifications:results:*"
	keys, err := s.redisClient.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	var results []*NotificationResult
	for i, key := range keys {
		if i >= limit {
			break
		}

		var result NotificationResult
		if err := s.redisClient.Get(ctx, key, &result); err == nil {
			results = append(results, &result)
		}
	}

	return results, nil
}

// TestNotification sends a test notification
func (s *Service) TestNotification(ctx context.Context, notificationType NotificationType) error {
	testReq := &NotificationRequest{
		ID:         fmt.Sprintf("test_%d", time.Now().Unix()),
		Type:       notificationType,
		Recipients: []string{"test@example.com"},
		Subject:    "Test Notification",
		Message:    "This is a test notification from the Real-Time Data Pipeline system.",
		Priority:   PriorityNormal,
		CreatedAt:  time.Now(),
	}

	_, err := s.SendNotification(ctx, testReq)
	return err
}
