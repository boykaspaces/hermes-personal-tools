package providers

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func setGitHubProviderEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("CREDENTIAL_PROVIDERS", "githubapp")
	t.Setenv("GITHUB_CREDENTIAL_PROFILE_ID", "github-self-management")
	t.Setenv("GITHUB_API_BASE_URL", "")
	t.Setenv("GITHUB_APP_ID", "123")
	t.Setenv("GITHUB_INSTALLATION_ID", "456")
	t.Setenv("GITHUB_REPOSITORY_ID", "789")
	t.Setenv("GITHUB_REPOSITORY_OWNER", "owner")
	t.Setenv("GITHUB_REPOSITORY_NAME", "repo")
	t.Setenv("GITHUB_PRIVATE_KEY_SECRET_ARN", "arn:aws:secretsmanager:region:account:secret:key")
}

func TestLoadRegistryLoadsConfiguredProvider(t *testing.T) {
	setGitHubProviderEnvironment(t)
	registry, err := LoadRegistry(aws.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Lookup("github-self-management"); !ok {
		t.Fatal("GitHub profile was not registered")
	}
}

func TestLoadRegistryRejectsUnknownAndDuplicateProviderTypes(t *testing.T) {
	for _, providerTypes := range []string{"unknown", "githubapp,githubapp", ""} {
		t.Run(providerTypes, func(t *testing.T) {
			setGitHubProviderEnvironment(t)
			t.Setenv("CREDENTIAL_PROVIDERS", providerTypes)
			if _, err := LoadRegistry(aws.Config{}); err == nil {
				t.Fatal("LoadRegistry accepted invalid provider types")
			}
		})
	}
}
