package googleauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Credentials is the shared OAuth credential used by the explicitly enabled
// Google Workspace providers. The refresh token must have been granted every
// scope requested by those providers.
type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	TokenURI     string `json:"token_uri,omitempty"`
}

func Parse(secretJSON string) (Credentials, error) {
	var credentials Credentials
	if err := json.Unmarshal([]byte(secretJSON), &credentials); err != nil {
		return Credentials{}, fmt.Errorf("google credential secret is not valid JSON: %w", err)
	}
	if credentials.ClientID == "" || credentials.ClientSecret == "" || credentials.RefreshToken == "" {
		return Credentials{}, errors.New("google credential secret requires client_id, client_secret and refresh_token")
	}
	return credentials, nil
}

func NewHTTPClient(ctx context.Context, secretJSON string, scopes ...string) (*http.Client, error) {
	credentials, err := Parse(secretJSON)
	if err != nil {
		return nil, err
	}
	endpoint := google.Endpoint
	if credentials.TokenURI != "" {
		endpoint.TokenURL = credentials.TokenURI
	}
	config := &oauth2.Config{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
		Endpoint:     endpoint,
		Scopes:       scopes,
	}
	token := &oauth2.Token{RefreshToken: credentials.RefreshToken}
	return config.Client(ctx, token), nil
}
