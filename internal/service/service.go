package service

import (
	"context"
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/traffic-tacos/payment-sim-api/internal/config"
	"github.com/traffic-tacos/payment-sim-api/internal/observability"
	"github.com/traffic-tacos/payment-sim-api/internal/store"
	"github.com/traffic-tacos/payment-sim-api/internal/webhook"
	"go.uber.org/zap"
)

// Scenario represents the payment simulation scenario
type Scenario string

const (
	ScenarioApprove Scenario = "approve"
	ScenarioFail    Scenario = "fail"
	ScenarioDelay   Scenario = "delay"
	ScenarioRandom  Scenario = "random"
)

// CreatePaymentIntentRequest represents the request to create a payment intent
type CreatePaymentIntentRequest struct {
	ReservationID string         `json:"reservation_id" validate:"required"`
	Amount        int64          `json:"amount" validate:"required,min=1"`
	Currency      string         `json:"currency,omitempty"`
	Scenario      Scenario       `json:"scenario" validate:"required,oneof=approve fail delay random"`
	DelayMs       int            `json:"delay_ms,omitempty"`
	WebhookURL    string         `json:"webhook_url" validate:"required,url"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// Validate validates the CreatePaymentIntentRequest
func (r *CreatePaymentIntentRequest) Validate() error {
	if strings.TrimSpace(r.ReservationID) == "" {
		return fmt.Errorf("reservation_id is required")
	}
	if r.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if r.Scenario == "" {
		return fmt.Errorf("scenario is required")
	}
	if !isValidScenario(r.Scenario) {
		return fmt.Errorf("invalid scenario: %s", r.Scenario)
	}
	if strings.TrimSpace(r.WebhookURL) == "" {
		return fmt.Errorf("webhook_url is required")
	}
	if _, err := url.ParseRequestURI(r.WebhookURL); err != nil {
		return fmt.Errorf("invalid webhook_url: %w", err)
	}
	return nil
}

func isValidScenario(s Scenario) bool {
	switch s {
	case ScenarioApprove, ScenarioFail, ScenarioDelay, ScenarioRandom:
		return true
	default:
		return false
	}
}

// NewPaymentIntent creates a new PaymentIntent
func NewPaymentIntent(req *CreatePaymentIntentRequest, idempotencyKey string) (*store.PaymentIntent, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id, err := generateULID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ULID: %w", err)
	}

	return &store.PaymentIntent{
		ID:             "pay_" + id,
		ReservationID:  req.ReservationID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Scenario:       string(req.Scenario),
		DelayMs:        req.DelayMs,
		WebhookURL:     req.WebhookURL,
		Metadata:       req.Metadata,
		Status:         "PENDING",
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// generateULID generates a new ULID
func generateULID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// Service handles the business logic for payment simulation
type Service struct {
	cfg        *config.Config
	logger     *zap.Logger
	store      store.Store
	dispatcher *webhook.Dispatcher
	metrics    *observability.Metrics
}

// NewService creates a new service
func NewService(cfg *config.Config, logger *zap.Logger, store store.Store, dispatcher *webhook.Dispatcher, metrics *observability.Metrics) *Service {
	return &Service{
		cfg:        cfg,
		logger:     logger,
		store:      store,
		dispatcher: dispatcher,
		metrics:    metrics,
	}
}

// CreatePaymentIntent creates a new payment intent
func (s *Service) CreatePaymentIntent(ctx context.Context, req *CreatePaymentIntentRequest, idempotencyKey string) (*store.PaymentIntent, error) {
	// Generate idempotency key if not provided (for internal consistency)
	if idempotencyKey == "" {
		idempotencyKey = store.GenerateIdempotencyHash(
			req.ReservationID,
			req.Amount,
			string(req.Scenario),
			req.DelayMs,
			req.WebhookURL,
			len(req.Metadata),
		)
	}

	// Check idempotency
	if existingID, exists := s.store.CheckIdempotency(idempotencyKey); exists {
		s.logger.Info("Idempotency hit", zap.String("idempotency_key", idempotencyKey), zap.String("existing_id", existingID))
		s.metrics.IdempotencyHitsTotal.Inc()

		// Return existing intent
		return s.store.GetIntent(existingID)
	}

	// Create new payment intent
	intent, err := NewPaymentIntent(req, idempotencyKey)
	if err != nil {
		s.logger.Error("Failed to create payment intent", zap.Error(err))
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	// Save to store
	if err := s.store.SaveIntent(intent); err != nil {
		s.logger.Error("Failed to save payment intent", zap.Error(err), zap.String("id", intent.ID))
		return nil, fmt.Errorf("failed to save payment intent: %w", err)
	}

	// Record scenario metric
	s.metrics.ScenarioCounterTotal.WithLabelValues(string(req.Scenario)).Inc()

	// Schedule webhook delivery
	go s.scheduleWebhookDelivery(ctx, intent)

	s.logger.Info("Payment intent created",
		zap.String("id", intent.ID),
		zap.String("scenario", string(req.Scenario)),
		zap.String("webhook_url", req.WebhookURL))

	return intent, nil
}

// scheduleWebhookDelivery schedules webhook delivery based on scenario
func (s *Service) scheduleWebhookDelivery(ctx context.Context, intent *store.PaymentIntent) {
	var delay time.Duration
	var webhookType string

	switch intent.Scenario {
	case string(ScenarioApprove):
		delay = time.Duration(s.cfg.DefaultApproveDelayMs) * time.Millisecond
		webhookType = "payment.approved"
		intent.UpdateStatus("APPROVED")

	case string(ScenarioFail):
		delay = time.Duration(s.cfg.DefaultFailDelayMs) * time.Millisecond
		webhookType = "payment.failed"
		intent.UpdateStatus("FAILED")

	case string(ScenarioDelay):
		if intent.DelayMs > 0 {
			delay = time.Duration(intent.DelayMs) * time.Millisecond
		} else {
			delay = time.Duration(s.cfg.DefaultDelayDelayMs) * time.Millisecond
		}
		webhookType = "payment.approved" // Delay scenario still approves
		intent.UpdateStatus("APPROVED")

	case string(ScenarioRandom):
		if mrand.Float64() < s.cfg.RandomApproveRate {
			delay = time.Duration(s.cfg.DefaultApproveDelayMs) * time.Millisecond
			webhookType = "payment.approved"
			intent.UpdateStatus("APPROVED")
		} else {
			delay = time.Duration(s.cfg.DefaultFailDelayMs) * time.Millisecond
			webhookType = "payment.failed"
			intent.UpdateStatus("FAILED")
		}

	default:
		s.logger.Error("Unknown scenario", zap.String("scenario", intent.Scenario))
		return
	}

	// Update intent in store
	if err := s.store.SaveIntent(intent); err != nil {
		s.logger.Error("Failed to update payment intent status", zap.Error(err), zap.String("id", intent.ID))
	}

	// Schedule webhook
	time.AfterFunc(delay, func() {
		payload := intent.NewWebhookPayload(webhookType)
		s.dispatcher.ScheduleWebhook(ctx, payload, intent.WebhookURL, intent.ID)
	})

	s.logger.Info("Webhook delivery scheduled",
		zap.String("payment_intent_id", intent.ID),
		zap.String("scenario", string(intent.Scenario)),
		zap.String("webhook_type", webhookType),
		zap.Duration("delay", delay))
}

// GetPaymentIntent retrieves a payment intent by ID
func (s *Service) GetPaymentIntent(ctx context.Context, id string) (*store.PaymentIntent, error) {
	intent, err := s.store.GetIntent(id)
	if err != nil {
		s.logger.Error("Failed to get payment intent", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("payment intent not found: %w", err)
	}

	return intent, nil
}

// TestWebhook manually triggers a webhook for testing
func (s *Service) TestWebhook(ctx context.Context, paymentIntentID, webhookType, targetURL string) error {
	// Get the payment intent
	intent, err := s.store.GetIntent(paymentIntentID)
	if err != nil {
		s.logger.Error("Payment intent not found for test webhook", zap.Error(err), zap.String("id", paymentIntentID))
		return fmt.Errorf("payment intent not found: %w", err)
	}

	// Create payload
	payload := intent.NewWebhookPayload(webhookType)

	// Schedule webhook immediately
	s.dispatcher.ScheduleWebhook(ctx, payload, targetURL, paymentIntentID)

	s.logger.Info("Test webhook scheduled",
		zap.String("payment_intent_id", paymentIntentID),
		zap.String("webhook_type", webhookType),
		zap.String("target_url", targetURL))

	return nil
}
