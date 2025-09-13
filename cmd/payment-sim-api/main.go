package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	"github.com/traffic-tacos/payment-sim-api/internal/config"
	"github.com/traffic-tacos/payment-sim-api/internal/http/handlers"
	"github.com/traffic-tacos/payment-sim-api/internal/http/middleware"
	"github.com/traffic-tacos/payment-sim-api/internal/observability"
	"github.com/traffic-tacos/payment-sim-api/internal/service"
	"github.com/traffic-tacos/payment-sim-api/internal/store"
	"github.com/traffic-tacos/payment-sim-api/internal/webhook"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := observability.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize OpenTelemetry
	shutdownTracer, err := observability.InitTracer(context.Background(), cfg.OTLPEndpoint)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			logger.Error("Failed to shutdown tracer", zap.Error(err))
		}
	}()

	// Initialize Prometheus metrics
	metrics := observability.NewMetrics()

	// Initialize components
	memStore := store.NewMemoryStore()
	webhookDispatcher := webhook.NewDispatcher(cfg, logger, metrics)
	paymentService := service.NewService(cfg, logger, memStore, webhookDispatcher, metrics)

	// Initialize HTTP server
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.OTelTracer())
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Metrics(metrics))

	// Initialize handlers
	h := handlers.NewHandlers(paymentService, logger, metrics)

	// Register routes
	r.Route("/v1/sim", func(r chi.Router) {
		r.Post("/intent", h.CreatePaymentIntent)
		r.Post("/webhook/test", h.TestWebhook)
		r.Get("/intents/{paymentIntentId}", h.GetPaymentIntent)
	})

	// Health and metrics endpoints
	r.Get("/healthz", h.HealthCheck)
	r.Get("/readyz", h.ReadinessCheck)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      otelhttp.NewHandler(r, "payment-sim-api"),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting payment-sim-api server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
