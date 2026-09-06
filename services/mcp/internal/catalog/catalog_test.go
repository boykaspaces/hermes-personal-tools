package catalog

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/secrets"
)

type googleCredentialSource struct {
	calls int
}

func (s *googleCredentialSource) GetString(context.Context, string) (string, error) {
	s.calls++
	return `{
		"client_id":"client",
		"client_secret":"secret",
		"refresh_token":"refresh"
	}`, nil
}

func TestDefinitionsShareCredentialCache(t *testing.T) {
	source := &googleCredentialSource{}
	cfg := config.Config{
		Calendar: config.Feature{Mode: config.ProviderGoogle},
		Gmail:    config.Feature{Mode: config.ProviderGoogle},
		AWSOps:   config.Feature{Mode: config.ProviderDisabled},
		GoogleWorkspace: config.GoogleWorkspace{
			CredentialSecretARN: "arn:example",
		},
	}
	configured, err := features.Build(
		context.Background(),
		Definitions(cfg, secrets.NewCachedStringSource(source), testRecorder()),
	)
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 1 {
		t.Fatalf("credential source calls = %d, want 1", source.calls)
	}
	if configured[0].Module == nil || configured[1].Module == nil || configured[2].Module != nil {
		t.Fatalf("Google features were not configured: %#v", configured)
	}
}

func TestDefinitionsKeepDisabledFeatureInvisible(t *testing.T) {
	cfg := config.Config{
		Calendar: config.Feature{Mode: config.ProviderMock},
		Gmail:    config.Feature{Mode: config.ProviderDisabled},
		AWSOps:   config.Feature{Mode: config.ProviderDisabled},
	}
	configured, err := features.Build(
		context.Background(),
		Definitions(cfg, secrets.NewCachedStringSource(nil), testRecorder()),
	)
	if err != nil {
		t.Fatal(err)
	}
	if configured[0].Module == nil || configured[1].Module != nil || configured[2].Module != nil {
		t.Fatalf("unexpected feature visibility: %#v", configured)
	}
}

func testRecorder() *audit.Recorder {
	return audit.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
}
