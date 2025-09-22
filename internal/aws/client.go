package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

	// Load AWS config with tacos profile
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.AWSRegion),
		config.WithSharedConfigProfile(cfg.AWSProfile),
	)
	if err != nil {
		logger.Error("Failed to load AWS config", zap.Error(err))
		return nil, err
	}

	logger.Info("AWS config loaded successfully",
		zap.String("region", awsConfig.Region))

	return &Clients{
		EventBridge: eventbridge.NewFromConfig(awsConfig),
		SQS:         sqs.NewFromConfig(awsConfig),
		Config:      awsConfig,
	}, nil
}