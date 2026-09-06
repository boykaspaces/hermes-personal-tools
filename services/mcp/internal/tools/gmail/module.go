package gmail

import (
	"context"
	"fmt"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/identity"
	gmailprovider "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/gmail"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const ToolName = "gmail_create_draft"

type Module struct {
	provider gmailprovider.Provider
	recorder *audit.Recorder
}

func New(provider gmailprovider.Provider, recorder *audit.Recorder) *Module {
	return &Module{provider: provider, recorder: recorder}
}

func (m *Module) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolName,
		Description: "Create a Gmail draft for human review. This tool never sends email and returns only draft identifiers.",
	}, m.createDraft)
}

func (m *Module) createDraft(ctx context.Context, _ *mcp.CallToolRequest, input gmailprovider.CreateDraftInput) (*mcp.CallToolResult, gmailprovider.CreateDraftOutput, error) {
	started := time.Now()
	caller, err := identity.Require(ctx)
	if err != nil {
		m.recorder.ToolCall(ctx, "unknown", ToolName, "denied", started, 0)
		return nil, gmailprovider.CreateDraftOutput{}, err
	}

	request, err := gmailprovider.ValidateCreateDraftInput(input)
	if err != nil {
		m.recorder.ToolCall(ctx, caller.ID, ToolName, "invalid", started, 0)
		return nil, gmailprovider.CreateDraftOutput{}, fmt.Errorf("invalid Gmail draft: %w", err)
	}
	draft, err := m.provider.CreateDraft(ctx, request)
	if err != nil {
		m.recorder.ToolCall(ctx, caller.ID, ToolName, "provider_error", started, 0)
		return nil, gmailprovider.CreateDraftOutput{}, fmt.Errorf("gmail provider failed: %w", err)
	}

	output := gmailprovider.CreateDraftOutput{
		DraftID:   draft.DraftID,
		MessageID: draft.MessageID,
		ThreadID:  draft.ThreadID,
		Status:    "draft",
		Sent:      false,
		Provider:  m.provider.Name(),
	}
	m.recorder.ToolCall(ctx, caller.ID, ToolName, "draft_created", started, 1)
	return nil, output, nil
}
