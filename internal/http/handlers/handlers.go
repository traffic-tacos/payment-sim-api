package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/traffic-tacos/payment-sim-api/internal/observability"
	"github.com/traffic-tacos/payment-sim-api/internal/service"
	"go.uber.org/zap"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	service *service.Service
	logger  *zap.Logger
	metrics *observability.Metrics
}

// NewHandlers creates new handlers
func NewHandlers(svc *service.Service, logger *zap.Logger, metrics *observability.Metrics) *Handlers {
	return &Handlers{
		service: svc,
		logger:  logger,
		metrics: metrics,
	}
}

// CreatePaymentIntent handles POST /v1/sim/intent
func (h *Handlers) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req service.CreatePaymentIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON payload", err)
		return
	}

	// Extract idempotency key from header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey != "" {
		if _, err := uuid.Parse(idempotencyKey); err != nil {
			h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid Idempotency-Key format", err)
			return
		}
	}

	// Create payment intent
	intent, err := h.service.CreatePaymentIntent(r.Context(), &req, idempotencyKey)
	if err != nil {
		h.logger.Error("Failed to create payment intent", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create payment intent", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"payment_intent_id": intent.ID,
		"status":            intent.Status,
		"next":              "webhook",
	})
}

// TestWebhook handles POST /v1/sim/webhook/test
func (h *Handlers) TestWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PaymentIntentID string `json:"payment_intent_id"`
		Type            string `json:"type"`
		WebhookURL      string `json:"webhook_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON payload", err)
		return
	}

	// Validate required fields
	if req.PaymentIntentID == "" || req.Type == "" || req.WebhookURL == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Missing required fields", nil)
		return
	}

	// Validate webhook type
	if req.Type != "payment.approved" && req.Type != "payment.failed" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid webhook type", nil)
		return
	}

	// Trigger test webhook
	err := h.service.TestWebhook(r.Context(), req.PaymentIntentID, req.Type, req.WebhookURL)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Payment intent not found", err)
		return
	}

	h.writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"sent": true,
	})
}

// GetPaymentIntent handles GET /v1/sim/intents/{paymentIntentId}
func (h *Handlers) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {
	paymentIntentID := chi.URLParam(r, "paymentIntentId")
	if paymentIntentID == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Missing payment_intent_id", nil)
		return
	}

	intent, err := h.service.GetPaymentIntent(r.Context(), paymentIntentID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Payment intent not found", err)
		return
	}

	h.writeJSON(w, http.StatusOK, intent)
}

// HealthCheck handles GET /healthz
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ReadinessCheck handles GET /readyz
func (h *Handlers) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// In a real application, you might check database connectivity, etc.
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		TraceID string `json:"trace_id,omitempty"`
	} `json:"error"`
}

// writeError writes an error response
func (h *Handlers) writeError(w http.ResponseWriter, status int, code, message string, err error) {
	w.WriteHeader(status)

	response := ErrorResponse{}
	response.Error.Code = code
	response.Error.Message = message

	// Add trace ID if available (simplified for now)
	response.Error.TraceID = ""

	if err != nil {
		h.logger.Error("HTTP error response",
			zap.Int("status", status),
			zap.String("code", code),
			zap.String("message", message),
			zap.Error(err))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeJSON writes a JSON response
func (h *Handlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
		// Fallback error response
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"Failed to encode response"}}`))
	}
}
