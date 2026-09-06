package features

import (
	"context"
	"errors"
	"testing"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/protocol"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testModule struct{}

func (testModule) Register(*mcp.Server) {}

func TestBuildResolvesIndependentModes(t *testing.T) {
	features, err := Build(context.Background(), []Definition{
		{Name: "calendar", Mode: DisabledMode},
		{
			Name: "gmail",
			Mode: "mock",
			Providers: map[string]ProviderFactory{
				"mock": func(context.Context) (protocol.Module, error) {
					return testModule{}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if features[0].Module != nil || features[1].Module == nil {
		t.Fatalf("unexpected features: %#v", features)
	}
}

func TestBuildRejectsInvalidDefinitions(t *testing.T) {
	tests := [][]Definition{
		{{Name: "", Mode: DisabledMode}},
		{{Name: "calendar", Mode: DisabledMode}, {Name: "calendar", Mode: DisabledMode}},
		{{Name: "calendar", Mode: "unknown"}},
		{{
			Name: "calendar",
			Mode: "mock",
			Providers: map[string]ProviderFactory{
				"mock": func(context.Context) (protocol.Module, error) { return nil, nil },
			},
		}},
	}
	for _, definitions := range tests {
		if _, err := Build(context.Background(), definitions); err == nil {
			t.Fatalf("expected error for %#v", definitions)
		}
	}
}

func TestBuildWrapsFactoryError(t *testing.T) {
	want := errors.New("provider failed")
	_, err := Build(context.Background(), []Definition{{
		Name: "calendar",
		Mode: "google",
		Providers: map[string]ProviderFactory{
			"google": func(context.Context) (protocol.Module, error) { return nil, want },
		},
	}})
	if !errors.Is(err, want) {
		t.Fatalf("Build() error = %v, want wrapped provider error", err)
	}
}
