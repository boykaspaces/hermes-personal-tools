// Package catalog is the explicit production allowlist for MCP features.
// Importing a provider or tool package elsewhere does not expose it.
package catalog

import (
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	awsopsfeature "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features/awsops"
	calendarfeature "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features/calendar"
	gmailfeature "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features/gmail"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/secrets"
)

func Definitions(cfg config.Config, credentials secrets.StringSource, recorder *audit.Recorder) []features.Definition {
	googleSecretID := cfg.GoogleWorkspace.CredentialSecretARN
	return []features.Definition{
		calendarfeature.Definition(cfg.Calendar.Mode, googleSecretID, credentials, recorder),
		gmailfeature.Definition(cfg.Gmail.Mode, googleSecretID, credentials, recorder),
		awsopsfeature.Definition(cfg.AWSOps.Mode, cfg.AWSOperations, recorder),
	}
}
