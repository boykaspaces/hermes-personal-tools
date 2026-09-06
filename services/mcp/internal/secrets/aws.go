package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type StringSource interface {
	GetString(context.Context, string) (string, error)
}

type AWSStringSource struct {
	client *secretsmanager.Client
}

func NewAWSStringSource(ctx context.Context) (*AWSStringSource, error) {
	config, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	return &AWSStringSource{client: secretsmanager.NewFromConfig(config)}, nil
}

func (s *AWSStringSource) GetString(ctx context.Context, secretARN string) (string, error) {
	output, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretARN),
	})
	if err != nil {
		return "", err
	}
	if output.SecretString == nil || *output.SecretString == "" {
		return "", errors.New("secret must contain a non-empty string value")
	}
	return *output.SecretString, nil
}
