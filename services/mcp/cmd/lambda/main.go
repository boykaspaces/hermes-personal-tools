package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/app"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/catalog"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/secrets"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/transport"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.FromEnvironment(os.Getenv)
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}

	var awsSecretSource secrets.StringSource
	if cfg.UsesGoogle() {
		awsSecretSource, err = secrets.NewAWSStringSource(ctx)
		if err != nil {
			logger.Error("initialize Secret source", "error", err)
			os.Exit(1)
		}
	}
	recorder := audit.New(logger)
	credentials := secrets.NewCachedStringSource(awsSecretSource)
	configuredFeatures, err := features.Build(ctx, catalog.Definitions(cfg, credentials, recorder))
	if err != nil {
		logger.Error("initialize features", "error", err)
		os.Exit(1)
	}

	application := app.New(configuredFeatures, logger)
	lambda.Start(transport.NewLambdaHTTPHandler(application).Handle)
}
