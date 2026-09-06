package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/lease"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/provider"
)

type fakeIssuer struct{ issued lease.Lease }

func (f fakeIssuer) Issue(context.Context) (lease.Lease, error) { return f.issued, nil }

func testRegistry(t *testing.T, registrations ...provider.Registration) *provider.Registry {
	t.Helper()
	registry, err := provider.NewRegistry(registrations...)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestIssueLease(t *testing.T) {
	logOutput := &bytes.Buffer{}
	issued := lease.Lease{
		LeaseID: "lease-1", Profile: "github-hermes", Provider: "test-provider",
		Credential: lease.Credential{Type: lease.CredentialHTTPBasic, Username: "x-access-token", Secret: "sensitive-token"},
		Targets:    []lease.Target{{Protocol: "https", Host: "github.com", PathPrefix: "owner/repo"}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	registry := testRegistry(t, provider.Registration{Profile: "github-hermes", Issuer: fakeIssuer{issued: issued}})
	handler := New(registry, slog.New(slog.NewJSONHandler(logOutput, nil)))
	request := httptest.NewRequest(http.MethodPost, "/v1/credential-leases", strings.NewReader(`{"profile":"github-hermes"}`))
	request = request.WithContext(WithCaller(request.Context(), "arn:aws:iam::123456789012:role/hermes"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if !strings.Contains(response.Body.String(), "sensitive-token") {
		t.Fatal("lease response did not contain credential")
	}
	if strings.Contains(logOutput.String(), "sensitive-token") {
		t.Fatal("audit log exposed credential")
	}
}

func TestIssueLeaseRejectsMissingIdentityAndUnknownFields(t *testing.T) {
	registry := testRegistry(t, provider.Registration{Profile: "github-hermes", Issuer: fakeIssuer{}})
	handler := New(registry, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tests := []struct {
		name   string
		body   string
		caller string
		status int
	}{
		{name: "missing identity", body: `{"profile":"github-hermes"}`, status: http.StatusUnauthorized},
		{name: "unknown field", body: `{"profile":"github-hermes","target":"other"}`, caller: "caller", status: http.StatusBadRequest},
		{name: "unknown profile", body: `{"profile":"other"}`, caller: "caller", status: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/credential-leases", strings.NewReader(test.body))
			if test.caller != "" {
				request = request.WithContext(WithCaller(request.Context(), test.caller))
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestIssueLeaseRoutesMultipleProfilesThroughOneHandler(t *testing.T) {
	githubLease := lease.Lease{LeaseID: "github-lease", Profile: "github-hermes", Provider: "github-app"}
	otherLease := lease.Lease{LeaseID: "other-lease", Profile: "other-profile", Provider: "test-provider"}
	registry := testRegistry(t,
		provider.Registration{Profile: "github-hermes", Issuer: fakeIssuer{issued: githubLease}},
		provider.Registration{Profile: "other-profile", Issuer: fakeIssuer{issued: otherLease}},
	)
	handler := New(registry, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/v1/credential-leases", strings.NewReader(`{"profile":"other-profile"}`))
	request = request.WithContext(WithCaller(request.Context(), "caller"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "other-lease") || strings.Contains(response.Body.String(), "github-lease") {
		t.Fatalf("handler routed to the wrong provider: %s", response.Body.String())
	}
}
