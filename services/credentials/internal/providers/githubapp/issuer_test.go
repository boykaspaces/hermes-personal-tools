package githubapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type staticSecret string

func (s staticSecret) Get(context.Context, string) (string, error) { return string(s), nil }

func TestIssueUsesFixedRepositoryAndPermissions(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	secret := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	expiresAt := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/app/installations/456/access_tokens" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing app JWT")
		}
		var body struct {
			RepositoryIDs []int64           `json:"repository_ids"`
			Permissions   map[string]string `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.RepositoryIDs) != 1 || body.RepositoryIDs[0] != 789 {
			t.Errorf("repository_ids = %v", body.RepositoryIDs)
		}
		if body.Permissions["contents"] != "write" || body.Permissions["pull_requests"] != "write" {
			t.Errorf("permissions = %v", body.Permissions)
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"token":"installation-token","expires_at":"2026-09-06T12:00:00Z"}`)),
			Request:    r,
		}, nil
	})}

	issuer := New(Config{
		ProfileID: "github-hermes", APIBaseURL: "https://api.github.test", AppID: "123", InstallationID: 456,
		RepositoryID: 789, RepositoryOwner: "owner", RepositoryName: "repo", PrivateKeySecretARN: "secret-arn",
	}, staticSecret(secret), client)
	issuer.now = func() time.Time { return expiresAt.Add(-time.Hour) }
	issued, err := issuer.Issue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if issued.Credential.Secret != "installation-token" {
		t.Fatal("wrong token")
	}
	if got := issued.Targets[0].PathPrefix; got != "owner/repo" {
		t.Fatalf("path prefix = %q", got)
	}
	if issued.ExpiresAt != expiresAt {
		t.Fatalf("expires = %s", issued.ExpiresAt)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestDecodePrivateKeySecretJSON(t *testing.T) {
	got, err := decodePrivateKeySecret(`{"private_key_pem":"-----BEGIN TEST-----\\nkey\\n-----END TEST-----"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "-----BEGIN TEST-----") {
		t.Fatalf("got %q", got)
	}
}
