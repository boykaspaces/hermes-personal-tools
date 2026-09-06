package provisioner

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type staticCredentials struct{}

func (staticCredentials) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "secret", SessionToken: "session"}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestRefreshSignsAndAtomicallyStoresLeaseWithoutLoggingSecret(t *testing.T) {
	now := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	outputPath := filepath.Join(t.TempDir(), "credentials", "github.json")
	var logs bytes.Buffer
	client, err := New(Config{
		Endpoint: "https://example.execute-api.ap-southeast-1.amazonaws.com/v1/credential-leases",
		Region:   "ap-southeast-1", Profile: "github-self-management", OutputPath: outputPath,
	}, staticCredentials{}, slog.New(slog.NewJSONHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	client.now = func() time.Time { return now }
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(request.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Errorf("request was not SigV4 signed")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"lease_id":"lease-1",
				"profile":"github-self-management",
				"provider":"github-app",
				"credential":{"type":"http-basic","username":"x-access-token","secret":"sensitive-token"},
				"targets":[{"protocol":"https","host":"github.com","path_prefix":"owner/repo"}],
				"expires_at":"2026-09-06T11:00:00Z"
			}`)),
			Request: request,
		}, nil
	})
	issued, err := client.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if issued.LeaseID != "lease-1" {
		t.Fatalf("lease ID = %q", issued.LeaseID)
	}
	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(contents, []byte("sensitive-token")) {
		t.Fatal("stored lease is missing token")
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("lease mode = %o", got)
	}
	if strings.Contains(logs.String(), "sensitive-token") {
		t.Fatal("provisioner log exposed token")
	}
}

func TestNewRejectsNonLeaseEndpoint(t *testing.T) {
	for _, endpoint := range []string{
		"http://example.test/v1/credential-leases",
		"https://example.test/other",
		"https://example.test/v1/credential-leases?target=other",
	} {
		_, err := New(Config{
			Endpoint: endpoint, Region: "ap-southeast-1", Profile: "profile",
			OutputPath: filepath.Join(t.TempDir(), "lease.json"),
		}, staticCredentials{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err == nil {
			t.Errorf("expected endpoint %q to be rejected", endpoint)
		}
	}
}
