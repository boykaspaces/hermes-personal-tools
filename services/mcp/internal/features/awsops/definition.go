package awsops

import (
	"context"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/protocol"
	awsopsprovider "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/awsops"
	awsopstool "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/tools/awsops"
)

const Name = "awsops"

func Definition(mode config.ProviderMode, configuredTargets config.AWSOperations, recorder *audit.Recorder) features.Definition {
	targets := awsopsprovider.Targets{
		StackName:              configuredTargets.StackName,
		MCPFunctionName:        configuredTargets.MCPFunctionName,
		AuthorizerFunctionName: configuredTargets.AuthorizerFunctionName,
		HermesManagedNodeID:    configuredTargets.HermesManagedNodeID,
		LogGroupNames:          configuredTargets.LogGroupNames,
		AccountID:              configuredTargets.AccountID,
		BudgetNames:            configuredTargets.BudgetNames,
	}
	return features.Definition{
		Name: Name,
		Mode: string(mode),
		Providers: map[string]features.ProviderFactory{
			string(config.ProviderMock): func(context.Context) (protocol.Module, error) {
				return awsopstool.New(awsopsprovider.MockProvider{}, recorder), nil
			},
			string(config.ProviderAWS): func(ctx context.Context) (protocol.Module, error) {
				provider, err := awsopsprovider.NewAWSProvider(ctx, targets)
				if err != nil {
					return nil, err
				}
				return awsopstool.New(provider, recorder), nil
			},
		},
	}
}
