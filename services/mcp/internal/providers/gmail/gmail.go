package gmail

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	maxRecipients   = 20
	maxSubjectRunes = 200
	maxBodyBytes    = 100_000
)

type CreateDraftInput struct {
	To       []string `json:"to" jsonschema:"recipient email addresses; at least one and at most 20"`
	CC       []string `json:"cc,omitempty" jsonschema:"optional copy recipient email addresses"`
	Subject  string   `json:"subject" jsonschema:"email subject; at most 200 characters"`
	BodyText string   `json:"body_text" jsonschema:"plain-text email body; at most 100000 bytes"`
}

type CreateDraftRequest struct {
	To       []string
	CC       []string
	Subject  string
	BodyText string
}

type Draft struct {
	DraftID   string
	MessageID string
	ThreadID  string
}

type CreateDraftOutput struct {
	DraftID   string `json:"draft_id"`
	MessageID string `json:"message_id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty"`
	Status    string `json:"status"`
	Sent      bool   `json:"sent"`
	Provider  string `json:"provider"`
}

type Provider interface {
	CreateDraft(context.Context, CreateDraftRequest) (Draft, error)
	Name() string
}

func ValidateCreateDraftInput(input CreateDraftInput) (CreateDraftRequest, error) {
	if len(input.To) == 0 {
		return CreateDraftRequest{}, errors.New("at least one to recipient is required")
	}
	if len(input.To)+len(input.CC) > maxRecipients {
		return CreateDraftRequest{}, fmt.Errorf("recipient count cannot exceed %d", maxRecipients)
	}
	to, err := validateAddresses(input.To)
	if err != nil {
		return CreateDraftRequest{}, fmt.Errorf("invalid to recipient: %w", err)
	}
	cc, err := validateAddresses(input.CC)
	if err != nil {
		return CreateDraftRequest{}, fmt.Errorf("invalid cc recipient: %w", err)
	}

	subject := strings.TrimSpace(input.Subject)
	if strings.ContainsAny(subject, "\r\n") {
		return CreateDraftRequest{}, errors.New("subject cannot contain line breaks")
	}
	if utf8.RuneCountInString(subject) > maxSubjectRunes {
		return CreateDraftRequest{}, fmt.Errorf("subject cannot exceed %d characters", maxSubjectRunes)
	}
	if len(input.BodyText) > maxBodyBytes {
		return CreateDraftRequest{}, fmt.Errorf("body_text cannot exceed %d bytes", maxBodyBytes)
	}

	return CreateDraftRequest{
		To:       to,
		CC:       cc,
		Subject:  subject,
		BodyText: input.BodyText,
	}, nil
}

func validateAddresses(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.ContainsAny(value, "\r\n") {
			return nil, errors.New("address cannot contain line breaks")
		}
		address, err := mail.ParseAddress(strings.TrimSpace(value))
		if err != nil || address.Address == "" {
			return nil, fmt.Errorf("%q is not a valid email address", value)
		}
		result = append(result, address.String())
	}
	return result, nil
}
