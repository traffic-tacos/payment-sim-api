package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	paymentv1 "github.com/traffic-tacos/proto-contracts/gen/go/payment/v1"
	awsClient "github.com/traffic-tacos/payment-sim-api/internal/aws"
	"github.com/traffic-tacos/payment-sim-api/internal/config"
	"github.com/traffic-tacos/payment-sim-api/internal/events"
	"github.com/traffic-tacos/payment-sim-api/internal/grpc/server"
	"github.com/traffic-tacos/payment-sim-api/internal/observability"
	"github.com/traffic-tacos/payment-sim-api/internal/service"
	"github.com/traffic-tacos/payment-sim-api/internal/webhook"
)

func main() {
	// Load configuration
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := observability.NewLogger(cfg.Environment)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize AWS clients
	ctx := context.Background()
	awsClients, err := awsClient.NewClients(ctx, &cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize AWS clients", zap.Error(err))
	}

	// Initialize EventBridge publisher
	eventPublisher := events.NewPublisher(awsClients.EventBridge, &cfg, logger)

	// Initialize services
	webhookDispatcher := webhook.NewDispatcher(logger, &cfg, eventPublisher)
	paymentService := service.NewPaymentService(logger, &cfg, webhookDispatcher, eventPublisher)

	// Setup gRPC server
	grpcServer := grpc.NewServer()
	paymentGRPCServer := server.NewPaymentServer(paymentService, logger)
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentGRPCServer)

	// Enable gRPC reflection for grpcui
	reflection.Register(grpcServer)

	// Setup metrics and health check server (like inventory-api)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthHandler)

	metricsServer := &http.Server{
		Addr:    ":8031",
		Handler: mux,
	}

	// Start servers
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()

		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			logger.Fatal("Failed to listen for gRPC", zap.Error(err))
		}

		logger.Info("Starting gRPC server", zap.Int("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("gRPC server failed", zap.Error(err))
		}
	}()

	// Start metrics server
	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info("Starting metrics server", zap.Int("port", 8031))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Handle shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	logger.Info("Shutting down servers...")
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Metrics server shutdown failed", zap.Error(err))
	}

	grpcServer.GracefulStop()
	wg.Wait()

	logger.Info("Servers stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"payment-sim-api"}`))
}