package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/boykaspaces/hermes-personal-tools/services/authorizer/internal/auth"
	personalsecrets "github.com/boykaspaces/hermes-personal-tools/services/authorizer/internal/secrets"
)

const clientTokenCacheTTL = time.Minute

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	secretARN := os.Getenv("CLIENT_TOKEN_SECRET_ARN")
	if secretARN == "" {
		logger.Error("CLIENT_TOKEN_SECRET_ARN is required")
		os.Exit(1)
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("load AWS configuration", "error", err)
		os.Exit(1)
	}
	client := secretsmanager.NewFromConfig(awsConfig)
	tokens := personalsecrets.NewCachedString(func(ctx context.Context) (string, error) {
		output, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretARN),
		})
		if err != nil {
			return "", err
		}
		if output.SecretString == nil {
			return "", errors.New("client token secret must contain a string")
		}
		return *output.SecretString, nil
	}, clientTokenCacheTTL)

	lambda.Start(auth.NewLambdaAuthorizer(tokens, logger).Handle)
}
