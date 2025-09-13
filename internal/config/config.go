package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port string

	// Webhook
	WebhookSecret         string
	DefaultApproveDelayMs int
	DefaultFailDelayMs    int
	DefaultDelayDelayMs   int
	RandomApproveRate     float64
	WebhookTimeoutMs      int
	WebhookMaxRetries     int
	WebhookBackoffMs      int
	WebhookMaxRPS         int

	// Observability
	OTLPEndpoint string
	LogLevel     string
}

func Load() (*Config, error) {
	cfg := &Config{
		// Server defaults
		Port: getEnvOrDefault("PORT", "8080"),

		// Webhook defaults
		WebhookSecret:         getEnvOrDefault("WEBHOOK_SECRET", ""),
		DefaultApproveDelayMs: getEnvIntOrDefault("DEFAULT_APPROVE_DELAY_MS", 200),
		DefaultFailDelayMs:    getEnvIntOrDefault("DEFAULT_FAIL_DELAY_MS", 100),
		DefaultDelayDelayMs:   getEnvIntOrDefault("DEFAULT_DELAY_DELAY_MS", 3000),
		RandomApproveRate:     getEnvFloatOrDefault("RANDOM_APPROVE_RATE", 0.8),
		WebhookTimeoutMs:      getEnvIntOrDefault("WEBHOOK_TIMEOUT_MS", 1000),
		WebhookMaxRetries:     getEnvIntOrDefault("WEBHOOK_MAX_RETRIES", 5),
		WebhookBackoffMs:      getEnvIntOrDefault("WEBHOOK_BACKOFF_MS", 1000),
		WebhookMaxRPS:         getEnvIntOrDefault("WEBHOOK_MAX_RPS", 500),

		// Observability defaults
		OTLPEndpoint: getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4317"),
		LogLevel:     getEnvOrDefault("LOG_LEVEL", "info"),
	}

	// Validate required fields
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("WEBHOOK_SECRET is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
