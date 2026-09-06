package features

import (
	"context"
	"errors"
	"fmt"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/protocol"
)

const DisabledMode = "disabled"

type ProviderFactory func(context.Context) (protocol.Module, error)

// Definition declares one independently configurable MCP feature and the
// provider implementations that are allowed to activate it.
type Definition struct {
	Name      string
	Mode      string
	Providers map[string]ProviderFactory
}

type Feature struct {
	Name   string
	Mode   string
	Module protocol.Module
}

// Build resolves declarative feature definitions into the modules exposed by
// the MCP server. Definitions are the production allowlist: provider code is
// unreachable until it is registered here under an enabled mode.
func Build(ctx context.Context, definitions []Definition) ([]Feature, error) {
	features := make([]Feature, 0, len(definitions))
	names := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if definition.Name == "" {
			return nil, errors.New("feature name is required")
		}
		if _, duplicate := names[definition.Name]; duplicate {
			return nil, fmt.Errorf("duplicate feature %q", definition.Name)
		}
		names[definition.Name] = struct{}{}

		feature := Feature{Name: definition.Name, Mode: definition.Mode}
		if definition.Mode == DisabledMode {
			features = append(features, feature)
			continue
		}
		factory, supported := definition.Providers[definition.Mode]
		if !supported {
			return nil, fmt.Errorf("feature %q does not support provider mode %q", definition.Name, definition.Mode)
		}
		module, err := factory(ctx)
		if err != nil {
			return nil, fmt.Errorf("initialize feature %q in %q mode: %w", definition.Name, definition.Mode, err)
		}
		if module == nil {
			return nil, fmt.Errorf("feature %q provider %q returned a nil module", definition.Name, definition.Mode)
		}
		feature.Module = module
		features = append(features, feature)
	}
	return features, nil
}
