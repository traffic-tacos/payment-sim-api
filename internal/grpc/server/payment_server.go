package server

import (
	"context"

	"go.uber.org/zap"

	paymentv1 "github.com/traffic-tacos/payment-sim-api/gen/go/proto/payment/v1"
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

func (s *PaymentServer) GetPaymentIntent(ctx context.Context, req *paymentv1.GetPaymentIntentRequest) (*paymentv1.GetPaymentIntentResponse, error) {
	s.logger.Info("gRPC GetPaymentIntent called",
		zap.String("payment_id", req.PaymentId))

	return s.paymentService.GetPaymentIntent(ctx, req)
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	s.logger.Info("gRPC ProcessPayment called",
		zap.String("payment_id", req.PaymentId))

	return s.paymentService.ProcessPayment(ctx, req)
}