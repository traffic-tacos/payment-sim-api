package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traffic-tacos/payment-sim-api/internal/store"
)

// mockDispatcher implements webhook dispatcher interface for testing
type mockDispatcher struct{}

func (m *mockDispatcher) ScheduleWebhook(ctx context.Context, payload *store.WebhookPayload, targetURL, paymentIntentID string) {
	// No-op for testing
}

func (m *mockDispatcher) Stop() {
	// No-op for testing
}

func TestCreatePaymentIntentRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         CreatePaymentIntentRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			req: CreatePaymentIntentRequest{
				ReservationID: "rsv_123",
				Amount:        10000,
				Scenario:      ScenarioApprove,
				WebhookURL:    "https://example.com/webhook",
			},
			expectError: false,
		},
		{
			name: "missing reservation_id",
			req: CreatePaymentIntentRequest{
				Amount:     10000,
				Scenario:   ScenarioApprove,
				WebhookURL: "https://example.com/webhook",
			},
			expectError: true,
			errorMsg:    "reservation_id is required",
		},
		{
			name: "zero amount",
			req: CreatePaymentIntentRequest{
				ReservationID: "rsv_123",
				Amount:        0,
				Scenario:      ScenarioApprove,
				WebhookURL:    "https://example.com/webhook",
			},
			expectError: true,
			errorMsg:    "amount must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
