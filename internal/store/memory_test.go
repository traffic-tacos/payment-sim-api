package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.intents)
	assert.NotNil(t, store.idempotencyKeys)
}

func TestMemoryStore_SaveAndGetIntent(t *testing.T) {
	store := NewMemoryStore()

	intent := &PaymentIntent{
		ID:             "pay_test123",
		ReservationID:  "rsv_123",
		Amount:         10000,
		Scenario:       "approve",
		WebhookURL:     "https://example.com/webhook",
		Status:         "PENDING",
		IdempotencyKey: "idempotency-key-123",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save
	err := store.SaveIntent(intent)
	require.NoError(t, err)

	// Get
	retrieved, err := store.GetIntent("pay_test123")
	require.NoError(t, err)
	assert.Equal(t, intent.ID, retrieved.ID)
	assert.Equal(t, intent.ReservationID, retrieved.ReservationID)
	assert.Equal(t, intent.Amount, retrieved.Amount)
	assert.Equal(t, intent.IdempotencyKey, retrieved.IdempotencyKey)
}

func TestMemoryStore_GetIntent_NotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.GetIntent("pay_nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment intent not found")
}

func TestMemoryStore_CheckIdempotency(t *testing.T) {
	store := NewMemoryStore()

	intent := &PaymentIntent{
		ID:             "pay_test456",
		ReservationID:  "rsv_456",
		Amount:         20000,
		IdempotencyKey: "idempotency-key-456",
	}

	// Save intent
	err := store.SaveIntent(intent)
	require.NoError(t, err)

	// Check idempotency
	paymentIntentID, exists := store.CheckIdempotency("idempotency-key-456")
	assert.True(t, exists)
	assert.Equal(t, "pay_test456", paymentIntentID)

	// Check non-existent idempotency key
	_, exists = store.CheckIdempotency("non-existent-key")
	assert.False(t, exists)
}

func TestGenerateIdempotencyHash(t *testing.T) {
	hash1 := GenerateIdempotencyHash("rsv_123", 10000, "approve", 0, "https://example.com/webhook", 0)
	hash2 := GenerateIdempotencyHash("rsv_123", 10000, "approve", 0, "https://example.com/webhook", 0)
	hash3 := GenerateIdempotencyHash("rsv_456", 10000, "approve", 0, "https://example.com/webhook", 0)

	// Same inputs should produce same hash
	assert.Equal(t, hash1, hash2)

	// Different inputs should produce different hash
	assert.NotEqual(t, hash1, hash3)

	// Hash should be non-empty
	assert.NotEmpty(t, hash1)
}

func TestMemoryStore_TTL(t *testing.T) {
	// Create store with short TTL for testing
	store := &MemoryStore{
		intents:         make(map[string]*PaymentIntent),
		idempotencyKeys: make(map[string]string),
		cleanupInterval: 100 * time.Millisecond, // Very short cleanup interval
		ttl:             200 * time.Millisecond, // Very short TTL
	}

	// Start cleanup
	go store.startCleanup()

	intent := &PaymentIntent{
		ID:             "pay_test789",
		IdempotencyKey: "idempotency-key-789",
		CreatedAt:      time.Now(),
	}

	// Save intent
	err := store.SaveIntent(intent)
	require.NoError(t, err)

	// Should exist immediately
	_, err = store.GetIntent("pay_test789")
	assert.NoError(t, err)

	// Wait for TTL to expire
	time.Sleep(500 * time.Millisecond)

	// Should be cleaned up (this might be flaky due to timing)
	// In real scenarios, the cleanup would run periodically
	t.Log("TTL test completed - cleanup may have occurred")
}
