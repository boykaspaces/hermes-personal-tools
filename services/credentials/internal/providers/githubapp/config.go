package githubapp

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// LoadConfig reads and validates the deployment-owned configuration for the
// GitHub App credential provider.
func LoadConfig() (Config, error) {
	installationID, err := positiveInt64("GITHUB_INSTALLATION_ID")
	if err != nil {
		return Config{}, err
	}
	repositoryID, err := positiveInt64("GITHUB_REPOSITORY_ID")
	if err != nil {
		return Config{}, err
	}
	config := Config{
		ProfileID:           strings.TrimSpace(os.Getenv("GITHUB_CREDENTIAL_PROFILE_ID")),
		APIBaseURL:          strings.TrimSpace(os.Getenv("GITHUB_API_BASE_URL")),
		AppID:               strings.TrimSpace(os.Getenv("GITHUB_APP_ID")),
		InstallationID:      installationID,
		RepositoryID:        repositoryID,
		RepositoryOwner:     strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY_OWNER")),
		RepositoryName:      strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY_NAME")),
		PrivateKeySecretARN: strings.TrimSpace(os.Getenv("GITHUB_PRIVATE_KEY_SECRET_ARN")),
	}
	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://api.github.com"
	}
	for name, value := range map[string]string{
		"GITHUB_CREDENTIAL_PROFILE_ID":  config.ProfileID,
		"GITHUB_APP_ID":                 config.AppID,
		"GITHUB_REPOSITORY_OWNER":       config.RepositoryOwner,
		"GITHUB_REPOSITORY_NAME":        config.RepositoryName,
		"GITHUB_PRIVATE_KEY_SECRET_ARN": config.PrivateKeySecretARN,
	} {
		if value == "" {
			return Config{}, fmt.Errorf("%s is required", name)
		}
	}
	if strings.ContainsAny(config.RepositoryOwner, "/\\") || strings.ContainsAny(config.RepositoryName, "/\\") {
		return Config{}, errors.New("GitHub repository owner and name must be single path segments")
	}
	parsed, err := url.Parse(config.APIBaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Config{}, errors.New("GITHUB_API_BASE_URL must be an HTTPS origin")
	}
	return config, nil
}

func positiveInt64(name string) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}
