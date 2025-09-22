package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"

	awsClient "github.com/traffic-tacos/payment-sim-api/internal/aws"
	"github.com/traffic-tacos/payment-sim-api/internal/config"
)

type PaymentEventMessage struct {
	PaymentID     string `json:"payment_id"`
	ReservationID string `json:"reservation_id"`
	UserID        string `json:"user_id"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Timestamp     int64  `json:"timestamp"`
	EventType     string `json:"event_type"`
}

type ReservationWorker struct {
	sqsClient *sqs.Client
	queueURL  string
	logger    *zap.Logger
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting fake Reservation Worker for design demo")

	// Config 로드
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		logger.Fatal("Failed to process environment config", zap.Error(err))
	}

	// AWS 클라이언트 초기화
	ctx := context.Background()
	awsClients, err := awsClient.NewClients(ctx, &cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize AWS clients", zap.Error(err))
	}

	worker := &ReservationWorker{
		sqsClient: awsClients.SQS,
		queueURL:  cfg.PaymentWebhookQueueURL,
		logger:    logger,
	}

	// Graceful shutdown 설정
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		logger.Info("Received interrupt signal, shutting down...")
		cancel()
	}()

	// SQS 폴링 시작
	worker.startPolling(ctx)
}

func (w *ReservationWorker) startPolling(ctx context.Context) {
	w.logger.Info("Starting SQS polling", zap.String("queue_url", w.queueURL))

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Context cancelled, stopping polling")
			return
		default:
			w.pollMessages(ctx)
			time.Sleep(2 * time.Second) // 2초마다 폴링
		}
	}
}

func (w *ReservationWorker) pollMessages(ctx context.Context) {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(w.queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20, // Long polling
		VisibilityTimeout:   60,
	}

	result, err := w.sqsClient.ReceiveMessage(ctx, input)
	if err != nil {
		w.logger.Error("Failed to receive messages from SQS", zap.Error(err))
		return
	}

	w.logger.Info("Polled SQS",
		zap.Int("message_count", len(result.Messages)))

	for _, message := range result.Messages {
		w.processMessage(ctx, message)
	}
}

func (w *ReservationWorker) processMessage(ctx context.Context, message types.Message) {
	w.logger.Info("Processing SQS message",
		zap.String("message_id", aws.ToString(message.MessageId)))

	// EventBridge 메시지 파싱 (EventBridge -> SQS 형태)
	var eventBridgeMessage struct {
		Source     string                  `json:"source"`
		DetailType string                  `json:"detail-type"`
		Detail     PaymentEventMessage     `json:"detail"`
	}

	if err := json.Unmarshal([]byte(aws.ToString(message.Body)), &eventBridgeMessage); err != nil {
		w.logger.Error("Failed to unmarshal EventBridge message", zap.Error(err))
		w.deleteMessage(ctx, message)
		return
	}

	paymentEvent := eventBridgeMessage.Detail

	w.logger.Info("Processing payment event",
		zap.String("payment_id", paymentEvent.PaymentID),
		zap.String("reservation_id", paymentEvent.ReservationID),
		zap.String("status", paymentEvent.Status),
		zap.Int64("amount", paymentEvent.Amount))

	// 가라 예약 처리 로직 (설계 발표용)
	w.processReservation(paymentEvent)

	// 메시지 삭제 (성공적으로 처리됨)
	w.deleteMessage(ctx, message)
}

func (w *ReservationWorker) processReservation(event PaymentEventMessage) {
	// 가라 비즈니스 로직 (실제로는 예약 상태 업데이트 등)
	switch event.Status {
	case "PAYMENT_STATUS_APPROVED":
		w.logger.Info("Payment approved - updating reservation to CONFIRMED",
			zap.String("reservation_id", event.ReservationID),
			zap.String("payment_id", event.PaymentID))

		// 실제로는 여기서 reservation DB 업데이트
		// 예: reservationService.UpdateStatus(event.ReservationID, "CONFIRMED")

	case "PAYMENT_STATUS_FAILED":
		w.logger.Info("Payment failed - updating reservation to PAYMENT_FAILED",
			zap.String("reservation_id", event.ReservationID),
			zap.String("payment_id", event.PaymentID))

		// 실제로는 여기서 reservation DB 업데이트
		// 예: reservationService.UpdateStatus(event.ReservationID, "PAYMENT_FAILED")

	default:
		w.logger.Warn("Unknown payment status received",
			zap.String("status", event.Status),
			zap.String("payment_id", event.PaymentID))
	}

	// 가라 처리 시간 시뮬레이션
	time.Sleep(100 * time.Millisecond)

	w.logger.Info("Reservation processing completed",
		zap.String("reservation_id", event.ReservationID),
		zap.String("payment_status", event.Status))
}

func (w *ReservationWorker) deleteMessage(ctx context.Context, message types.Message) {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.queueURL),
		ReceiptHandle: message.ReceiptHandle,
	}

	_, err := w.sqsClient.DeleteMessage(ctx, input)
	if err != nil {
		w.logger.Error("Failed to delete message from SQS",
			zap.String("message_id", aws.ToString(message.MessageId)),
			zap.Error(err))
	} else {
		w.logger.Info("Message deleted successfully",
			zap.String("message_id", aws.ToString(message.MessageId)))
	}
}