package gitcredential

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/clients/credential-agent/internal/lease"
)

func TestGetReturnsOnlyMatchingCredential(t *testing.T) {
	directory := t.TempDir()
	item := lease.Lease{
		LeaseID: "1", Profile: "github-hermes", Provider: "github-app",
		Credential: lease.Credential{Type: lease.CredentialHTTPBasic, Username: "x-access-token", Secret: "token-value"},
		Targets:    []lease.Target{{Protocol: "https", Host: "github.com", PathPrefix: "owner/repo"}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	data, _ := json.Marshal(item)
	if err := os.WriteFile(filepath.Join(directory, "lease.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	input := strings.NewReader("protocol=https\nhost=github.com\npath=owner/repo.git\n\n")
	if err := Run("get", directory, input, &output, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "username=x-access-token\npassword=token-value\n\n" {
		t.Fatalf("output = %q", got)
	}

	output.Reset()
	input = strings.NewReader("protocol=https\nhost=github.com\npath=owner/other.git\n\n")
	if err := Run("get", directory, input, &output, time.Now()); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected output %q", output.String())
	}
}
