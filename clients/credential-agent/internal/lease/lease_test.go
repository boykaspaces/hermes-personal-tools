package lease

import (
	"testing"
	"time"
)

func TestMatchesExactRepositoryOnly(t *testing.T) {
	item := Lease{Targets: []Target{{Protocol: "https", Host: "github.com", PathPrefix: "owner/repo"}}}
	for _, path := range []string{"owner/repo", "/owner/repo.git", "owner/repo/info/refs"} {
		if !item.Matches("HTTPS", "GITHUB.COM", path) {
			t.Errorf("expected match for %q", path)
		}
	}
	for _, path := range []string{"owner/repository", "other/repo", "owner"} {
		if item.Matches("https", "github.com", path) {
			t.Errorf("unexpected match for %q", path)
		}
	}
}

func TestValidateRejectsNearExpiry(t *testing.T) {
	now := time.Now()
	item := Lease{
		LeaseID: "1", Profile: "p", Provider: "provider",
		Credential: Credential{Type: CredentialHTTPBasic, Username: "u", Secret: "s"},
		Targets:    []Target{{Protocol: "https", Host: "github.com", PathPrefix: "owner/repo"}},
		ExpiresAt:  now.Add(20 * time.Second),
	}
	if err := item.Validate(now); err == nil {
		t.Fatal("expected near-expiry lease rejection")
	}
}
