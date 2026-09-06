package gmail

import (
	"strings"
	"testing"
)

func TestValidateCreateDraftInput(t *testing.T) {
	got, err := ValidateCreateDraftInput(CreateDraftInput{
		To:       []string{"Alice <alice@example.com>"},
		Subject:  "Hello",
		BodyText: "Draft body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.To[0], "alice@example.com") || got.Subject != "Hello" {
		t.Fatalf("unexpected request: %#v", got)
	}
}

func TestValidateCreateDraftInputRejectsUnsafeValues(t *testing.T) {
	tests := []CreateDraftInput{
		{Subject: "missing recipient"},
		{To: []string{"alice@example.com\r\nBcc: attacker@example.com"}},
		{To: []string{"not-an-address"}},
		{To: []string{"alice@example.com"}, Subject: "hello\r\nBcc: attacker@example.com"},
		{To: []string{"alice@example.com"}, BodyText: strings.Repeat("x", maxBodyBytes+1)},
	}
	for _, input := range tests {
		if _, err := ValidateCreateDraftInput(input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}

func TestBuildPlainTextMessage(t *testing.T) {
	message := string(buildPlainTextMessage(CreateDraftRequest{
		To:       []string{"alice@example.com"},
		Subject:  "你好",
		BodyText: "line 1\nline 2",
	}))
	if !strings.Contains(message, "Subject: =?UTF-8?q?") || !strings.Contains(message, "\r\n\r\nline 1\r\nline 2") {
		t.Fatalf("unexpected MIME message: %q", message)
	}
}
