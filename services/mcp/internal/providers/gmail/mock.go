package gmail

import "context"

// MockProvider verifies protocol and safety behavior without creating any
// external side effect.
type MockProvider struct{}

func (MockProvider) Name() string { return "mock" }

func (MockProvider) CreateDraft(context.Context, CreateDraftRequest) (Draft, error) {
	return Draft{
		DraftID:   "mock-draft-1",
		MessageID: "mock-message-1",
		ThreadID:  "mock-thread-1",
	}, nil
}
