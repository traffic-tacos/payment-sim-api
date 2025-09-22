package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"go.uber.org/zap"

	"github.com/traffic-tacos/payment-sim-api/internal/config"
)

type PaymentEvent struct {
	PaymentID     string `json:"payment_id"`
	ReservationID string `json:"reservation_id"`
	UserID        string `json:"user_id"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Timestamp     int64  `json:"timestamp"`
	EventType     string `json:"event_type"`
}

type Publisher struct {
	eventBridge *eventbridge.Client
	config      *config.Config
	logger      *zap.Logger
}

func NewPublisher(eventBridge *eventbridge.Client, config *config.Config, logger *zap.Logger) *Publisher {
	return &Publisher{
		eventBridge: eventBridge,
		config:      config,
		logger:      logger,
	}
}

func (p *Publisher) PublishPaymentEvent(ctx context.Context, event PaymentEvent) error {
	event.Timestamp = time.Now().Unix()
	event.EventType = "payment.status_updated"

	detail, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal payment event", zap.Error(err))
		return err
	}

	entry := types.PutEventsRequestEntry{
		Source:     aws.String(p.config.EventSource),
		DetailType: aws.String("Payment Status Updated"),
		Detail:     aws.String(string(detail)),
		EventBusName: aws.String(p.config.EventBusName),
	}

	input := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{entry},
	}

	p.logger.Info("Publishing payment event to EventBridge",
		zap.String("payment_id", event.PaymentID),
		zap.String("status", event.Status),
		zap.String("event_bus", p.config.EventBusName))

	result, err := p.eventBridge.PutEvents(ctx, input)
	if err != nil {
		p.logger.Error("Failed to publish event to EventBridge", zap.Error(err))
		return err
	}

	// Check for failed entries
	if len(result.Entries) > 0 && result.Entries[0].ErrorCode != nil {
		p.logger.Error("EventBridge entry failed",
			zap.String("error_code", *result.Entries[0].ErrorCode),
			zap.String("error_message", aws.ToString(result.Entries[0].ErrorMessage)))
		return err
	}

	p.logger.Info("Payment event published successfully",
		zap.String("payment_id", event.PaymentID),
		zap.String("event_id", aws.ToString(result.Entries[0].EventId)))

	return nil
}