package server

import (
	"context"

	"go.uber.org/zap"

	paymentv1 "github.com/traffic-tacos/proto-contracts/gen/go/payment/v1"
	"github.com/traffic-tacos/payment-sim-api/internal/service"
)

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	paymentService *service.PaymentService
	logger         *zap.Logger
}

func NewPaymentServer(paymentService *service.PaymentService, logger *zap.Logger) *PaymentServer {
	return &PaymentServer{
		paymentService: paymentService,
		logger:         logger,
	}
}

func (s *PaymentServer) CreatePaymentIntent(ctx context.Context, req *paymentv1.CreatePaymentIntentRequest) (*paymentv1.CreatePaymentIntentResponse, error) {
	s.logger.Info("gRPC CreatePaymentIntent called",
		zap.String("reservation_id", req.ReservationId),
		zap.String("user_id", req.UserId))

	return s.paymentService.CreatePaymentIntent(ctx, req)
}

func (s *PaymentServer) GetPaymentStatus(ctx context.Context, req *paymentv1.GetPaymentStatusRequest) (*paymentv1.GetPaymentStatusResponse, error) {
	s.logger.Info("gRPC GetPaymentStatus called",
		zap.String("payment_intent_id", req.PaymentIntentId))

	return s.paymentService.GetPaymentStatus(ctx, req)
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	s.logger.Info("gRPC ProcessPayment called",
		zap.String("payment_intent_id", req.PaymentIntentId))

	return s.paymentService.ProcessPayment(ctx, req)
}