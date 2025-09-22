package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/traffic-tacos/payment-sim-api/internal/config"
	"github.com/traffic-tacos/payment-sim-api/internal/events"
)

type WebhookPayload struct {
	PaymentID     string `json:"payment_id"`
	ReservationID string `json:"reservation_id"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Timestamp     int64  `json:"timestamp"`
	EventType     string `json:"event_type"`
}

type Dispatcher struct {
	logger     *zap.Logger
	config     *config.Config
	httpClient *http.Client
	publisher  *events.Publisher
}

func NewDispatcher(logger *zap.Logger, config *config.Config, publisher *events.Publisher) *Dispatcher {
	return &Dispatcher{
		logger:    logger,
		config:    config,
		publisher: publisher,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (d *Dispatcher) SendPaymentWebhookAsync(paymentID, reservationID, finalStatus, webhookURL string, amount int64, currency string, delaySeconds int) {
	// 비동기 실행 (실제 PG사처럼 지연 후 webhook 발송)
	go func() {
		// 가라 지연 (실제 PG 처리 시뮬레이션)
		time.Sleep(time.Duration(delaySeconds) * time.Second)

		payload := WebhookPayload{
			PaymentID:     paymentID,
			ReservationID: reservationID,
			Status:        finalStatus,
			Amount:        amount,
			Currency:      currency,
			Timestamp:     time.Now().Unix(),
			EventType:     "payment.status_updated",
		}

		// EventBridge로 실제 이벤트 발송 (SQS로 라우팅됨)
		if d.publisher != nil {
			event := events.PaymentEvent{
				PaymentID:     paymentID,
				ReservationID: reservationID,
				Status:        finalStatus,
				Amount:        amount,
				Currency:      currency,
			}

			ctx := context.Background()
			if err := d.publisher.PublishPaymentEvent(ctx, event); err != nil {
				d.logger.Error("Failed to publish payment event to EventBridge",
					zap.String("payment_id", paymentID),
					zap.Error(err))
			}
		}

		// HTTP Webhook도 여전히 발송 (기존 시스템 호환성)
		err := d.sendWebhook(payload, webhookURL)
		if err != nil {
			d.logger.Error("Failed to send webhook",
				zap.String("payment_id", paymentID),
				zap.String("webhook_url", webhookURL),
				zap.Error(err))
		} else {
			d.logger.Info("Webhook sent successfully",
				zap.String("payment_id", paymentID),
				zap.String("status", finalStatus),
				zap.String("webhook_url", webhookURL))
		}
	}()
}

func (d *Dispatcher) sendWebhook(payload WebhookPayload, webhookURL string) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// HTTP 헤더 설정 (실제 PG사 방식)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PaymentSim/1.0")

	// HMAC 서명 추가 (보안)
	if d.config.WebhookSecret != "" {
		signature := d.generateSignature(jsonData, d.config.WebhookSecret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	d.logger.Info("Sending webhook",
		zap.String("payment_id", payload.PaymentID),
		zap.String("webhook_url", webhookURL),
		zap.String("status", payload.Status))

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (d *Dispatcher) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}