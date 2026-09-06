package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/app"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/providers"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/transport"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	awsConfig, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("load AWS configuration", "error", err)
		os.Exit(1)
	}
	registry, err := providers.LoadRegistry(awsConfig)
	if err != nil {
		logger.Error("load credential provider registry", "error", err)
		os.Exit(1)
	}
	handler := transport.NewLambdaHTTPHandler(app.New(registry, logger))
	lambda.Start(handler.Handle)
}
