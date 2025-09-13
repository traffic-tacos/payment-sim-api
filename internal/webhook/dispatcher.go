package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/traffic-tacos/payment-sim-api/internal/config"
	"github.com/traffic-tacos/payment-sim-api/internal/observability"
	"github.com/traffic-tacos/payment-sim-api/internal/store"
	"go.uber.org/zap"
)

// Dispatcher handles webhook delivery with retry logic and rate limiting
type Dispatcher struct {
	cfg     *config.Config
	logger  *zap.Logger
	metrics *observability.Metrics
	client  *http.Client
	queue   chan *webhookJob
	wg      sync.WaitGroup
	stopCh  chan struct{}
}

// webhookJob represents a webhook delivery job
type webhookJob struct {
	payload         *store.WebhookPayload
	targetURL       string
	paymentIntentID string
	attempt         int
}

// NewDispatcher creates a new webhook dispatcher
func NewDispatcher(cfg *config.Config, logger *zap.Logger, metrics *observability.Metrics) *Dispatcher {
	// Configure HTTP client with optimized settings
	client := &http.Client{
		Timeout: time.Duration(cfg.WebhookTimeoutMs) * time.Millisecond,
		Transport: &http.Transport{
			MaxIdleConns:        2000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	dispatcher := &Dispatcher{
		cfg:     cfg,
		logger:  logger,
		metrics: metrics,
		client:  client,
		queue:   make(chan *webhookJob, 10000), // Buffer size for high throughput
		stopCh:  make(chan struct{}),
	}

	// Start worker goroutines
	numWorkers := 8 // Can be made configurable
	for i := 0; i < numWorkers; i++ {
		dispatcher.wg.Add(1)
		go dispatcher.worker()
	}

	return dispatcher
}

// ScheduleWebhook schedules a webhook delivery
func (d *Dispatcher) ScheduleWebhook(ctx context.Context, payload *store.WebhookPayload, targetURL, paymentIntentID string) {
	select {
	case d.queue <- &webhookJob{
		payload:         payload,
		targetURL:       targetURL,
		paymentIntentID: paymentIntentID,
		attempt:         1,
	}:
		d.logger.Info("Webhook scheduled",
			zap.String("payment_intent_id", paymentIntentID),
			zap.String("target_url", targetURL),
			zap.String("type", payload.Type))
	default:
		d.logger.Error("Webhook queue full, dropping webhook",
			zap.String("payment_intent_id", paymentIntentID))
		d.metrics.WebhookDeliveryTotal.WithLabelValues("dropped").Inc()
	}
}

// worker processes webhook jobs from the queue
func (d *Dispatcher) worker() {
	defer d.wg.Done()

	for {
		select {
		case job := <-d.queue:
			d.processWebhook(job)
		case <-d.stopCh:
			return
		}
	}
}

// processWebhook processes a single webhook job
func (d *Dispatcher) processWebhook(job *webhookJob) {
	start := time.Now()

	// Create signed request
	req, err := d.createSignedRequest(job)
	if err != nil {
		d.logger.Error("Failed to create webhook request",
			zap.Error(err),
			zap.String("payment_intent_id", job.paymentIntentID),
			zap.Int("attempt", job.attempt))
		d.metrics.WebhookDeliveryTotal.WithLabelValues("error").Inc()
		return
	}

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		d.handleFailure(job, "network_error", err)
		return
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		duration := time.Since(start).Seconds()
		d.metrics.WebhookDeliveryTotal.WithLabelValues("success").Inc()
		d.metrics.WebhookLatency.Observe(duration)

		d.logger.Info("Webhook delivered successfully",
			zap.String("payment_intent_id", job.paymentIntentID),
			zap.String("target_url", job.targetURL),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("latency", time.Since(start)),
			zap.Int("attempt", job.attempt))

		// Read response body for debugging
		body, _ := io.ReadAll(resp.Body)
		d.logger.Debug("Webhook response",
			zap.String("payment_intent_id", job.paymentIntentID),
			zap.String("response_body", string(body)))
	} else {
		d.handleFailure(job, fmt.Sprintf("http_%d", resp.StatusCode), fmt.Errorf("HTTP %d", resp.StatusCode))
	}
}

// createSignedRequest creates a signed HTTP request
func (d *Dispatcher) createSignedRequest(job *webhookJob) (*http.Request, error) {
	// Serialize payload
	payloadBytes, err := json.Marshal(job.payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create timestamp
	timestamp := time.Now().UnixMilli()
	timestampStr := strconv.FormatInt(timestamp, 10)

	// Create signature
	signature := d.createSignature(payloadBytes, timestampStr)

	// Create request
	req, err := http.NewRequest("POST", job.targetURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Id", job.payload.PaymentIntentID+"_"+strconv.Itoa(job.attempt))
	req.Header.Set("X-Timestamp", timestampStr)
	req.Header.Set("X-Signature", "sha256="+signature)

	return req, nil
}

// createSignature creates HMAC signature
func (d *Dispatcher) createSignature(payload []byte, timestamp string) string {
	message := string(payload) + timestamp
	h := hmac.New(sha256.New, []byte(d.cfg.WebhookSecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// handleFailure handles webhook delivery failure
func (d *Dispatcher) handleFailure(job *webhookJob, reason string, err error) {
	d.logger.Warn("Webhook delivery failed",
		zap.String("payment_intent_id", job.paymentIntentID),
		zap.String("target_url", job.targetURL),
		zap.String("reason", reason),
		zap.Error(err),
		zap.Int("attempt", job.attempt))

	d.metrics.WebhookDeliveryTotal.WithLabelValues("failure").Inc()

	// Check if we should retry
	if job.attempt < d.cfg.WebhookMaxRetries {
		// Schedule retry with exponential backoff
		backoffDelay := d.calculateBackoff(job.attempt)
		job.attempt++

		time.AfterFunc(backoffDelay, func() {
			select {
			case d.queue <- job:
				d.logger.Info("Webhook retry scheduled",
					zap.String("payment_intent_id", job.paymentIntentID),
					zap.Int("attempt", job.attempt),
					zap.Duration("delay", backoffDelay))
			default:
				d.logger.Error("Webhook retry queue full",
					zap.String("payment_intent_id", job.paymentIntentID))
			}
		})
	} else {
		d.logger.Error("Webhook delivery abandoned after max retries",
			zap.String("payment_intent_id", job.paymentIntentID),
			zap.Int("attempt", job.attempt))
		d.metrics.WebhookDeliveryTotal.WithLabelValues("abandoned").Inc()
	}
}

// calculateBackoff calculates exponential backoff delay
func (d *Dispatcher) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: base_delay * 2^(attempt-1)
	baseDelay := time.Duration(d.cfg.WebhookBackoffMs) * time.Millisecond
	delay := baseDelay * time.Duration(math.Pow(2, float64(attempt-1)))

	// Cap at 30 seconds
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	return delay
}

// Stop stops the dispatcher and waits for all workers to finish
func (d *Dispatcher) Stop() {
	close(d.stopCh)
	d.wg.Wait()
}
