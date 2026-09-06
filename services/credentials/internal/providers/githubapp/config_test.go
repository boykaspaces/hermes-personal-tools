package githubapp

import "testing"

func setValidConfigEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_CREDENTIAL_PROFILE_ID", "github-self-management")
	t.Setenv("GITHUB_API_BASE_URL", "")
	t.Setenv("GITHUB_APP_ID", "123")
	t.Setenv("GITHUB_INSTALLATION_ID", "456")
	t.Setenv("GITHUB_REPOSITORY_ID", "789")
	t.Setenv("GITHUB_REPOSITORY_OWNER", "owner")
	t.Setenv("GITHUB_REPOSITORY_NAME", "repo")
	t.Setenv("GITHUB_PRIVATE_KEY_SECRET_ARN", "arn:aws:secretsmanager:region:account:secret:key")
}

func TestLoadConfigKeepsGitHubConfigurationInProvider(t *testing.T) {
	setValidConfigEnvironment(t)

	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.ProfileID != "github-self-management" || config.APIBaseURL != "https://api.github.com" {
		t.Fatalf("unexpected profile configuration: %+v", config)
	}
	if config.AppID != "123" || config.InstallationID != 456 || config.RepositoryID != 789 {
		t.Fatalf("unexpected GitHub identity configuration: %+v", config)
	}
	if config.RepositoryOwner != "owner" || config.RepositoryName != "repo" {
		t.Fatalf("unexpected repository configuration: %+v", config)
	}
}

func TestLoadConfigRejectsInvalidGitHubConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "non-positive installation", key: "GITHUB_INSTALLATION_ID", value: "0"},
		{name: "non-positive repository", key: "GITHUB_REPOSITORY_ID", value: "-1"},
		{name: "repository owner path", key: "GITHUB_REPOSITORY_OWNER", value: "owner/team"},
		{name: "insecure API origin", key: "GITHUB_API_BASE_URL", value: "http://api.github.test"},
		{name: "API origin with path", key: "GITHUB_API_BASE_URL", value: "https://api.github.test/v1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidConfigEnvironment(t)
			t.Setenv(test.key, test.value)
			if _, err := LoadConfig(); err == nil {
				t.Fatal("LoadConfig accepted invalid GitHub configuration")
			}
		})
	}
}
