package secrets

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Getter interface {
	Get(ctx context.Context, secretARN string) (string, error)
}

type AWSGetter struct {
	client *secretsmanager.Client
}

func NewAWSGetter(client *secretsmanager.Client) *AWSGetter {
	return &AWSGetter{client: client}
}

func (g *AWSGetter) Get(ctx context.Context, secretARN string) (string, error) {
	output, err := g.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretARN)})
	if err != nil {
		return "", err
	}
	if output.SecretString == nil {
		return "", errors.New("credential secret must contain a string")
	}
	return *output.SecretString, nil
}
