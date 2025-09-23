package aws

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"go.uber.org/zap"

	appconfig "github.com/traffic-tacos/payment-sim-api/internal/config"
)

type Clients struct {
	EventBridge *eventbridge.Client
	SQS         *sqs.Client
	Config      aws.Config
}

func NewClients(ctx context.Context, cfg *appconfig.Config, logger *zap.Logger) (*Clients, error) {
	logger.Info("Initializing AWS clients",
		zap.String("profile", cfg.AWSProfile),
		zap.String("region", cfg.AWSRegion))

	// Use static credentials from environment variables to avoid profile issues
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	endpointURL := os.Getenv("AWS_ENDPOINT_URL")

	if accessKey == "" {
		accessKey = "test"
	}
	if secretKey == "" {
		secretKey = "test"
	}

	logger.Info("Using static AWS credentials",
		zap.String("access_key", accessKey),
		zap.String("endpoint_url", endpointURL))

	// Create AWS config with static credentials
	awsConfig := aws.Config{
		Region:      cfg.AWSRegion,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	// Set custom endpoint if provided (for LocalStack)
	if endpointURL != "" {
		awsConfig.BaseEndpoint = aws.String(endpointURL)
	}

	logger.Info("AWS config loaded successfully",
		zap.String("region", awsConfig.Region))

	return &Clients{
		EventBridge: eventbridge.NewFromConfig(awsConfig),
		SQS:         sqs.NewFromConfig(awsConfig),
		Config:      awsConfig,
	}, nil
}