package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/boykaspaces/hermes-personal-tools/clients/credential-agent/internal/provisioner"
)

func main() {
	var configuration provisioner.Config
	var once bool
	flag.StringVar(&configuration.Endpoint, "endpoint", "", "HTTPS credential lease endpoint")
	flag.StringVar(&configuration.Region, "region", "", "AWS region for SigV4")
	flag.StringVar(&configuration.Profile, "profile", "", "server-defined credential profile")
	flag.StringVar(&configuration.OutputPath, "output", "", "lease output file")
	flag.DurationVar(&configuration.RefreshBefore, "refresh-before", 10*time.Minute, "refresh lead time")
	flag.BoolVar(&once, "once", false, "fetch one lease and exit")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(configuration.Region))
	if err != nil {
		logger.Error("load AWS configuration", "error", err)
		os.Exit(1)
	}
	client, err := provisioner.New(configuration, awsConfig.Credentials, logger)
	if err != nil {
		logger.Error("load provisioner configuration", "error", err)
		os.Exit(1)
	}
	if once {
		if _, err := client.Refresh(ctx); err != nil {
			logger.Error("credential lease refresh failed", "error", err)
			os.Exit(1)
		}
		return
	}
	if err := client.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("credential provisioner stopped", "error", err)
		os.Exit(1)
	}
}
