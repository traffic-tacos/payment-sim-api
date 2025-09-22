package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonv1 "github.com/traffic-tacos/payment-sim-api/gen/go/proto/common/v1"
	paymentv1 "github.com/traffic-tacos/payment-sim-api/gen/go/proto/payment/v1"
	"github.com/traffic-tacos/payment-sim-api/internal/config"
	"github.com/traffic-tacos/payment-sim-api/internal/events"
)

type PaymentIntent struct {
	ID            string
	ReservationID string
	UserID        string
	Amount        *commonv1.Money
	Status        paymentv1.PaymentStatus
	Scenario      string
	WebhookURL    string
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

type PaymentService struct {
	logger    *zap.Logger
	config    *config.Config
	intents   map[string]*PaymentIntent
	intentsMu sync.RWMutex
	webhook   WebhookSender
	publisher *events.Publisher
}

type WebhookSender interface {
	SendPaymentWebhookAsync(paymentID, reservationID, finalStatus, webhookURL string, amount int64, currency string, delaySeconds int)
}

func NewPaymentService(logger *zap.Logger, config *config.Config, webhook WebhookSender, publisher *events.Publisher) *PaymentService {
	return &PaymentService{
		logger:    logger,
		config:    config,
		intents:   make(map[string]*PaymentIntent),
		webhook:   webhook,
		publisher: publisher,
	}
}

func (s *PaymentService) CreatePaymentIntent(ctx context.Context, req *paymentv1.CreatePaymentIntentRequest) (*paymentv1.CreatePaymentIntentResponse, error) {
	s.logger.Info("Creating payment intent",
		zap.String("reservation_id", req.ReservationId),
		zap.String("user_id", req.UserId),
		zap.String("scenario", req.Scenario))

	intent := &PaymentIntent{
		ID:            uuid.New().String(),
		ReservationID: req.ReservationId,
		UserID:        req.UserId,
		Amount:        req.Amount,
		Status:        paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING, // 실제 PG사처럼 PENDING
		Scenario:      req.Scenario,
		WebhookURL:    req.WebhookUrl,
		CreatedAt:     time.Now(),
	}

	s.intentsMu.Lock()
	s.intents[intent.ID] = intent
	s.intentsMu.Unlock()

	// 실제 PG사처럼 비동기 webhook 발송 시작
	if intent.WebhookURL != "" && s.webhook != nil {
		finalStatus := s.determineFinalStatus(intent.Scenario)
		delaySeconds := s.config.DefaultDelayMs / 1000 // ms를 초로 변환
		if delaySeconds == 0 {
			delaySeconds = 2 // 기본 2초
		}

		s.webhook.SendPaymentWebhookAsync(
			intent.ID,
			intent.ReservationID,
			finalStatus,
			intent.WebhookURL,
			intent.Amount.Amount,
			intent.Amount.Currency,
			delaySeconds,
		)
	}

	return &paymentv1.CreatePaymentIntentResponse{
		PaymentId: intent.ID,
		Status:    intent.Status, // PENDING 상태로 즉시 응답
		CreatedAt: intent.CreatedAt.Unix(),
	}, nil
}

func (s *PaymentService) GetPaymentIntent(ctx context.Context, req *paymentv1.GetPaymentIntentRequest) (*paymentv1.GetPaymentIntentResponse, error) {
	s.intentsMu.RLock()
	intent, exists := s.intents[req.PaymentId]
	s.intentsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("payment intent not found: %s", req.PaymentId)
	}

	response := &paymentv1.GetPaymentIntentResponse{
		PaymentId:     intent.ID,
		ReservationId: intent.ReservationID,
		UserId:        intent.UserID,
		Amount:        intent.Amount,
		Status:        intent.Status,
		CreatedAt:     intent.CreatedAt.Unix(),
	}

	if intent.ProcessedAt != nil {
		response.ProcessedAt = intent.ProcessedAt.Unix()
	}

	return response, nil
}

func (s *PaymentService) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	s.intentsMu.RLock()
	intent, exists := s.intents[req.PaymentId]
	s.intentsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("payment intent not found: %s", req.PaymentId)
	}

	// Manual trigger - 즉시 상태 변경
	finalStatus := s.determineFinalStatus(intent.Scenario)
	if finalStatus == "PAYMENT_STATUS_APPROVED" {
		intent.Status = paymentv1.PaymentStatus_PAYMENT_STATUS_APPROVED
	} else {
		intent.Status = paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED
	}
	now := time.Now()
	intent.ProcessedAt = &now

	return &paymentv1.ProcessPaymentResponse{
		PaymentId: intent.ID,
		Status:    intent.Status,
	}, nil
}

// 시나리오에 따른 최종 상태 결정 (가라 데이터)
func (s *PaymentService) determineFinalStatus(scenario string) string {
	if scenario == "" {
		scenario = s.config.DefaultScenario
	}

	switch scenario {
	case "approve", "success":
		return "PAYMENT_STATUS_APPROVED"
	case "fail", "failed":
		return "PAYMENT_STATUS_FAILED"
	case "random":
		if time.Now().UnixNano()%2 == 0 {
			return "PAYMENT_STATUS_APPROVED"
		} else {
			return "PAYMENT_STATUS_FAILED"
		}
	default:
		return "PAYMENT_STATUS_APPROVED"
	}
}

// REST API helper methods
func (s *PaymentService) CreatePaymentIntentREST(ctx context.Context, intent *PaymentIntent, amount int64, currency string) (string, error) {
	intent.ID = uuid.New().String()
	intent.Amount = &commonv1.Money{
		Amount:   amount,
		Currency: currency,
	}
	intent.Status = paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING // 실제 PG사처럼 PENDING
	intent.CreatedAt = time.Now()

	s.intentsMu.Lock()
	s.intents[intent.ID] = intent
	s.intentsMu.Unlock()

	// 실제 PG사처럼 비동기 webhook 발송 시작
	if intent.WebhookURL != "" && s.webhook != nil {
		finalStatus := s.determineFinalStatus(intent.Scenario)
		delaySeconds := s.config.DefaultDelayMs / 1000
		if delaySeconds == 0 {
			delaySeconds = 2
		}

		s.webhook.SendPaymentWebhookAsync(
			intent.ID,
			intent.ReservationID,
			finalStatus,
			intent.WebhookURL,
			intent.Amount.Amount,
			intent.Amount.Currency,
			delaySeconds,
		)
	}

	return intent.ID, nil
}

func (s *PaymentService) GetPaymentIntentREST(ctx context.Context, paymentID string) (*PaymentIntent, error) {
	s.intentsMu.RLock()
	intent, exists := s.intents[paymentID]
	s.intentsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("payment intent not found: %s", paymentID)
	}

	return intent, nil
}