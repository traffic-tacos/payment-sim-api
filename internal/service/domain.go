package service

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// PaymentIntentStatus represents the status of a payment intent
type PaymentIntentStatus string

const (
	StatusPending  PaymentIntentStatus = "PENDING"
	StatusApproved PaymentIntentStatus = "APPROVED"
	StatusFailed   PaymentIntentStatus = "FAILED"
)

// Scenario represents the payment simulation scenario
type Scenario string

const (
	ScenarioApprove Scenario = "approve"
	ScenarioFail    Scenario = "fail"
	ScenarioDelay   Scenario = "delay"
	ScenarioRandom  Scenario = "random"
)

// PaymentIntent represents a payment intent
type PaymentIntent struct {
	ID             string              `json:"payment_intent_id"`
	ReservationID  string              `json:"reservation_id"`
	Amount         int64               `json:"amount"`
	Currency       string              `json:"currency,omitempty"`
	Scenario       Scenario            `json:"scenario"`
	DelayMs        int                 `json:"delay_ms,omitempty"`
	WebhookURL     string              `json:"webhook_url"`
	Metadata       map[string]any      `json:"metadata,omitempty"`
	Status         PaymentIntentStatus `json:"status"`
	IdempotencyKey string              `json:"-"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	LastAttemptAt  *time.Time          `json:"last_attempt_at,omitempty"`
	AttemptCount   int                 `json:"attempt_count,omitempty"`
}

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
func NewPaymentIntent(req *CreatePaymentIntentRequest, idempotencyKey string) (*PaymentIntent, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id, err := generateULID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ULID: %w", err)
	}

	return &PaymentIntent{
		ID:             "pay_" + id,
		ReservationID:  req.ReservationID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Scenario:       req.Scenario,
		DelayMs:        req.DelayMs,
		WebhookURL:     req.WebhookURL,
		Metadata:       req.Metadata,
		Status:         StatusPending,
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

// WebhookPayload represents the payload sent to webhook endpoints
type WebhookPayload struct {
	Type            string         `json:"type"`
	ReservationID   string         `json:"reservation_id"`
	PaymentIntentID string         `json:"payment_intent_id"`
	Amount          int64          `json:"amount"`
	Timestamp       string         `json:"ts"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// NewWebhookPayload creates a new WebhookPayload
func (pi *PaymentIntent) NewWebhookPayload(webhookType string) *WebhookPayload {
	return &WebhookPayload{
		Type:            webhookType,
		ReservationID:   pi.ReservationID,
		PaymentIntentID: pi.ID,
		Amount:          pi.Amount,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Metadata:        pi.Metadata,
	}
}

// UpdateStatus updates the payment intent status
func (pi *PaymentIntent) UpdateStatus(status PaymentIntentStatus) {
	now := time.Now().UTC()
	pi.Status = status
	pi.UpdatedAt = now
	pi.LastAttemptAt = &now
	pi.AttemptCount++
}
