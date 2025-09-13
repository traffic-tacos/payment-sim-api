package store

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// PaymentIntent represents a payment intent stored in memory
type PaymentIntent struct {
	ID             string
	ReservationID  string
	Amount         int64
	Currency       string
	Scenario       string
	DelayMs        int
	WebhookURL     string
	Metadata       map[string]any
	Status         string
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastAttemptAt  *time.Time
	AttemptCount   int
}

// GetID returns the payment intent ID
func (pi *PaymentIntent) GetID() string {
	return pi.ID
}

// GetIdempotencyKey returns the idempotency key
func (pi *PaymentIntent) GetIdempotencyKey() string {
	return pi.IdempotencyKey
}

// UpdateStatus updates the payment intent status
func (pi *PaymentIntent) UpdateStatus(status string) {
	now := time.Now().UTC()
	pi.Status = status
	pi.UpdatedAt = now
	pi.LastAttemptAt = &now
	pi.AttemptCount++
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

// WebhookPayload represents the payload sent to webhook endpoints
type WebhookPayload struct {
	Type            string         `json:"type"`
	ReservationID   string         `json:"reservation_id"`
	PaymentIntentID string         `json:"payment_intent_id"`
	Amount          int64          `json:"amount"`
	Timestamp       string         `json:"ts"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// MemoryStore implements an in-memory store with TTL for payment intents
type MemoryStore struct {
	mu              sync.RWMutex
	intents         map[string]*PaymentIntent
	idempotencyKeys map[string]string // idempotency_key -> payment_intent_id
	cleanupInterval time.Duration
	ttl             time.Duration
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		intents:         make(map[string]*PaymentIntent),
		idempotencyKeys: make(map[string]string),
		cleanupInterval: 5 * time.Minute, // Run cleanup every 5 minutes
		ttl:             24 * time.Hour,  // Keep intents for 24 hours
	}

	// Start cleanup goroutine
	go store.startCleanup()

	return store
}

// startCleanup starts the periodic cleanup of expired entries
func (s *MemoryStore) startCleanup() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes expired entries
func (s *MemoryStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	var expiredKeys []string
	var expiredIdempotencyKeys []string

	for id, intent := range s.intents {
		if now.Sub(intent.CreatedAt) > s.ttl {
			expiredKeys = append(expiredKeys, id)
			if intent.IdempotencyKey != "" {
				expiredIdempotencyKeys = append(expiredIdempotencyKeys, intent.IdempotencyKey)
			}
		}
	}

	for _, key := range expiredKeys {
		delete(s.intents, key)
	}

	for _, key := range expiredIdempotencyKeys {
		delete(s.idempotencyKeys, key)
	}
}

// GetIntent retrieves a payment intent by ID
func (s *MemoryStore) GetIntent(id string) (*PaymentIntent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	intent, exists := s.intents[id]
	if !exists {
		return nil, fmt.Errorf("payment intent not found: %s", id)
	}

	return intent, nil
}

// SaveIntent saves a payment intent
func (s *MemoryStore) SaveIntent(intent *PaymentIntent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.intents[intent.ID] = intent
	if intent.IdempotencyKey != "" {
		s.idempotencyKeys[intent.IdempotencyKey] = intent.ID
	}

	return nil
}

// CheckIdempotency checks if an idempotency key exists and returns the associated payment intent ID
func (s *MemoryStore) CheckIdempotency(idempotencyKey string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paymentIntentID, exists := s.idempotencyKeys[idempotencyKey]
	return paymentIntentID, exists
}

// GenerateIdempotencyHash generates a hash for idempotency checking based on request content
func GenerateIdempotencyHash(reservationID string, amount int64, scenario string, delayMs int, webhookURL string, metadataLen int) string {
	// Create a deterministic string representation of the request
	content := fmt.Sprintf("%s:%d:%s:%d:%s:%d",
		reservationID,
		amount,
		scenario,
		delayMs,
		webhookURL,
		metadataLen, // Simple metadata length check
	)

	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// Store defines the interface for payment intent storage
type Store interface {
	GetIntent(id string) (*PaymentIntent, error)
	SaveIntent(intent *PaymentIntent) error
	CheckIdempotency(idempotencyKey string) (string, bool)
}
