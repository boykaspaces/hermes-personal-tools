package gmail

import (
	"context"
	"fmt"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/protocol"
	gmailprovider "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/gmail"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/secrets"
	gmailtool "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/tools/gmail"
)

const Name = "gmail"

func Definition(mode config.ProviderMode, googleCredentialSecretID string, credentials secrets.StringSource, recorder *audit.Recorder) features.Definition {
	return features.Definition{
		Name: Name,
		Mode: string(mode),
		Providers: map[string]features.ProviderFactory{
			string(config.ProviderMock): func(context.Context) (protocol.Module, error) {
				return gmailtool.New(gmailprovider.MockProvider{}, recorder), nil
			},
			string(config.ProviderGoogle): func(ctx context.Context) (protocol.Module, error) {
				secretJSON, err := credentials.GetString(ctx, googleCredentialSecretID)
				if err != nil {
					return nil, fmt.Errorf("load Google credential: %w", err)
				}
				provider, err := gmailprovider.NewGoogleProvider(ctx, secretJSON)
				if err != nil {
					return nil, fmt.Errorf("initialize Google Gmail provider: %w", err)
				}
				return gmailtool.New(provider, recorder), nil
			},
		},
	}
}
