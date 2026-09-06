package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"strings"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/googleauth"
	googlegmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GoogleProvider struct {
	service *googlegmail.Service
}

func NewGoogleProvider(ctx context.Context, secretJSON string) (*GoogleProvider, error) {
	httpClient, err := googleauth.NewHTTPClient(ctx, secretJSON, googlegmail.GmailComposeScope)
	if err != nil {
		return nil, err
	}
	service, err := googlegmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("initialize Gmail client: %w", err)
	}
	return &GoogleProvider{service: service}, nil
}

func (p *GoogleProvider) Name() string { return "google_gmail" }

func (p *GoogleProvider) CreateDraft(ctx context.Context, request CreateDraftRequest) (Draft, error) {
	raw := base64.RawURLEncoding.EncodeToString(buildPlainTextMessage(request))
	created, err := p.service.Users.Drafts.Create("me", &googlegmail.Draft{
		Message: &googlegmail.Message{Raw: raw},
	}).Context(ctx).Do()
	if err != nil {
		return Draft{}, fmt.Errorf("create Gmail draft: %w", err)
	}

	draft := Draft{DraftID: created.Id}
	if created.Message != nil {
		draft.MessageID = created.Message.Id
		draft.ThreadID = created.Message.ThreadId
	}
	return draft, nil
}

func buildPlainTextMessage(request CreateDraftRequest) []byte {
	headers := []string{
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"To: " + strings.Join(request.To, ", "),
	}
	if len(request.CC) > 0 {
		headers = append(headers, "Cc: "+strings.Join(request.CC, ", "))
	}
	if request.Subject != "" {
		headers = append(headers, "Subject: "+mime.QEncoding.Encode("UTF-8", request.Subject))
	}
	body := strings.ReplaceAll(strings.ReplaceAll(request.BodyText, "\r\n", "\n"), "\r", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body)
}
