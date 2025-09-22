package config

type Config struct {
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	GRPCPort    int    `envconfig:"GRPC_PORT" default:"8003"`

	// AWS Configuration
	AWSRegion      string `envconfig:"AWS_REGION" default:"ap-northeast-2"`
	AWSProfile     string `envconfig:"AWS_PROFILE" default:"tacos"`
	EventBusName   string `envconfig:"EVENT_BUS_NAME" default:"ticket-reservation-events"`
	EventSource    string `envconfig:"PAYMENT_EVENT_SOURCE" default:"payment-sim-api"`

	// Real AWS SQS queues
	PaymentWebhookQueueURL string `envconfig:"PAYMENT_WEBHOOK_QUEUE_URL"`
	PaymentWebhookDLQURL   string `envconfig:"PAYMENT_WEBHOOK_DLQ_URL"`

	// Webhook configuration (실제 PG사 시뮬레이션용)
	WebhookSecret string `envconfig:"WEBHOOK_SECRET" default:"payment-sim-secret"`

	// Simulation settings
	DefaultDelayMs   int    `envconfig:"DEFAULT_DELAY_MS" default:"2000"`
	DefaultScenario  string `envconfig:"DEFAULT_SCENARIO" default:"approve"`
}