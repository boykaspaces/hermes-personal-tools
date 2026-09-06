package providers

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/provider"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/providers/githubapp"
	personalsecrets "github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/secrets"
)

const providerTypesEnvironment = "CREDENTIAL_PROVIDERS"

// LoadRegistry builds the server-controlled profile registry from a compiled
// allowlist of provider adapters. Adding a provider changes this composition
// package, not the Lambda entry point or HTTP handler.
func LoadRegistry(awsConfig aws.Config) (*provider.Registry, error) {
	providerTypes := strings.Split(os.Getenv(providerTypesEnvironment), ",")
	registrations := make([]provider.Registration, 0, len(providerTypes))
	seen := make(map[string]struct{}, len(providerTypes))
	for _, rawProviderType := range providerTypes {
		providerType := strings.TrimSpace(rawProviderType)
		if providerType == "" {
			return nil, errors.New("CREDENTIAL_PROVIDERS must list at least one provider")
		}
		if _, exists := seen[providerType]; exists {
			return nil, fmt.Errorf("credential provider type %q is listed more than once", providerType)
		}
		seen[providerType] = struct{}{}

		switch providerType {
		case "githubapp":
			configuration, err := githubapp.LoadConfig()
			if err != nil {
				return nil, fmt.Errorf("load githubapp provider: %w", err)
			}
			issuer := githubapp.New(
				configuration,
				personalsecrets.NewAWSGetter(secretsmanager.NewFromConfig(awsConfig)),
				nil,
			)
			registrations = append(registrations, provider.Registration{
				Profile: configuration.ProfileID,
				Issuer:  issuer,
			})
		default:
			return nil, fmt.Errorf("credential provider type %q is not compiled into this service", providerType)
		}
	}
	return provider.NewRegistry(registrations...)
}
