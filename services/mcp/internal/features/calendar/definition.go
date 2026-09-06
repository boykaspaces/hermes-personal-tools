package calendar

import (
	"context"
	"fmt"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/protocol"
	calendarprovider "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/calendar"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/secrets"
	calendartool "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/tools/calendar"
)

const Name = "calendar"

func Definition(mode config.ProviderMode, googleCredentialSecretID string, credentials secrets.StringSource, recorder *audit.Recorder) features.Definition {
	return features.Definition{
		Name: Name,
		Mode: string(mode),
		Providers: map[string]features.ProviderFactory{
			string(config.ProviderMock): func(context.Context) (protocol.Module, error) {
				return calendartool.New(calendarprovider.MockProvider{}, recorder), nil
			},
			string(config.ProviderGoogle): func(ctx context.Context) (protocol.Module, error) {
				secretJSON, err := credentials.GetString(ctx, googleCredentialSecretID)
				if err != nil {
					return nil, fmt.Errorf("load Google credential: %w", err)
				}
				provider, err := calendarprovider.NewGoogleProvider(ctx, secretJSON)
				if err != nil {
					return nil, fmt.Errorf("initialize Google Calendar provider: %w", err)
				}
				return calendartool.New(provider, recorder), nil
			},
		},
	}
}
