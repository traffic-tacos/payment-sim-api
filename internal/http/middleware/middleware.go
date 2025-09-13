package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/traffic-tacos/payment-sim-api/internal/observability"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RequestID adds a request ID to the request context
func RequestID(next http.Handler) http.Handler {
	return middleware.RequestID(next)
}

// Logger logs HTTP requests
func Logger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				requestID := middleware.GetReqID(r.Context())
				status := ww.Status()
				duration := time.Since(start)

				logger.Info("HTTP request",
					zap.String("request_id", requestID),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("query", r.URL.RawQuery),
					zap.String("user_agent", r.UserAgent()),
					zap.String("remote_addr", r.RemoteAddr),
					zap.Int("status", status),
					zap.Duration("latency", duration),
					zap.String("trace_id", getTraceID(r.Context())),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// OTelTracer adds OpenTelemetry tracing
func OTelTracer() func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware("payment-sim-api")
}

// Recover recovers from panics
func Recover(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					requestID := middleware.GetReqID(r.Context())

					logger.Error("Panic recovered",
						zap.String("request_id", requestID),
						zap.Any("panic", rvr),
						zap.String("trace_id", getTraceID(r.Context())),
					)

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Metrics collects HTTP metrics
func Metrics(metrics *observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				duration := time.Since(start)
				status := strconv.Itoa(ww.Status())

				metrics.HTTPRequestTotal.WithLabelValues(r.URL.Path, status).Inc()
				metrics.HTTPRequestDuration.WithLabelValues(r.URL.Path, r.Method, status).Observe(duration.Seconds())
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// getTraceID extracts trace ID from context
func getTraceID(ctx context.Context) string {
	if span := trace.SpanFromContext(ctx); span != nil {
		return span.SpanContext().TraceID().String()
	}
	return ""
}
