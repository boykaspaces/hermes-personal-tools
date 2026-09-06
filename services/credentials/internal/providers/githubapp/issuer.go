package githubapp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/lease"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/secrets"
)

const maxGitHubResponseBytes = 1 << 20

const providerID = "github-app"

// Config contains only GitHub App provider configuration. Generic HTTP and
// lease concerns remain outside this package.
type Config struct {
	ProfileID           string
	APIBaseURL          string
	AppID               string
	InstallationID      int64
	RepositoryID        int64
	RepositoryOwner     string
	RepositoryName      string
	PrivateKeySecretARN string
}

type Issuer struct {
	config  Config
	secrets secrets.Getter
	client  *http.Client
	now     func() time.Time
}

func New(config Config, secretGetter secrets.Getter, client *http.Client) *Issuer {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &Issuer{config: config, secrets: secretGetter, client: client, now: time.Now}
}

func (i *Issuer) Issue(ctx context.Context) (lease.Lease, error) {
	secretValue, err := i.secrets.Get(ctx, i.config.PrivateKeySecretARN)
	if err != nil {
		return lease.Lease{}, fmt.Errorf("read GitHub App private key: %w", err)
	}
	privateKeyPEM, err := decodePrivateKeySecret(secretValue)
	if err != nil {
		return lease.Lease{}, err
	}
	privateKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return lease.Lease{}, err
	}
	jwt, err := signAppJWT(privateKey, i.config.AppID, i.now())
	if err != nil {
		return lease.Lease{}, fmt.Errorf("sign GitHub App JWT: %w", err)
	}

	body, err := json.Marshal(struct {
		RepositoryIDs []int64           `json:"repository_ids"`
		Permissions   map[string]string `json:"permissions"`
	}{
		RepositoryIDs: []int64{i.config.RepositoryID},
		Permissions: map[string]string{
			"contents":      "write",
			"pull_requests": "write",
		},
	})
	if err != nil {
		return lease.Lease{}, fmt.Errorf("encode GitHub token request: %w", err)
	}
	endpoint := strings.TrimRight(i.config.APIBaseURL, "/") + fmt.Sprintf("/app/installations/%d/access_tokens", i.config.InstallationID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return lease.Lease{}, fmt.Errorf("create GitHub token request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+jwt)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "personal-tools-credential-lease")
	request.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	response, err := i.client.Do(request)
	if err != nil {
		return lease.Lease{}, fmt.Errorf("request GitHub installation token: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusCreated {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxGitHubResponseBytes))
		return lease.Lease{}, fmt.Errorf("GitHub installation token request returned status %d", response.StatusCode)
	}
	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxGitHubResponseBytes))
	if err := decoder.Decode(&tokenResponse); err != nil {
		return lease.Lease{}, fmt.Errorf("decode GitHub installation token response: %w", err)
	}
	if tokenResponse.Token == "" || tokenResponse.ExpiresAt.IsZero() || !tokenResponse.ExpiresAt.After(i.now()) {
		return lease.Lease{}, errors.New("GitHub returned an invalid or expired installation token")
	}
	leaseIDBytes := make([]byte, 16)
	if _, err := rand.Read(leaseIDBytes); err != nil {
		return lease.Lease{}, fmt.Errorf("generate lease identifier: %w", err)
	}
	return lease.Lease{
		LeaseID:  hex.EncodeToString(leaseIDBytes),
		Profile:  i.config.ProfileID,
		Provider: providerID,
		Credential: lease.Credential{
			Type:     lease.CredentialHTTPBasic,
			Username: "x-access-token",
			Secret:   tokenResponse.Token,
		},
		Targets: []lease.Target{{
			Protocol:   "https",
			Host:       "github.com",
			PathPrefix: i.config.RepositoryOwner + "/" + i.config.RepositoryName,
		}},
		ExpiresAt: tokenResponse.ExpiresAt.UTC(),
	}, nil
}

func decodePrivateKeySecret(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "-----BEGIN") {
		return trimmed, nil
	}
	var document struct {
		PrivateKeyPEM string `json:"private_key_pem"`
	}
	if err := json.Unmarshal([]byte(trimmed), &document); err != nil || strings.TrimSpace(document.PrivateKeyPEM) == "" {
		return "", errors.New("GitHub App secret must be a PEM key or JSON containing private_key_pem")
	}
	return strings.TrimSpace(document.PrivateKeyPEM), nil
}

func parseRSAPrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("decode GitHub App private key PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("parse GitHub App private key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("GitHub App private key must be RSA")
	}
	return key, nil
}

func signAppJWT(key *rsa.PrivateKey, appID string, now time.Time) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appID,
	})
	encode := base64.RawURLEncoding.EncodeToString
	unsigned := encode(header) + "." + encode(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + encode(signature), nil
}
