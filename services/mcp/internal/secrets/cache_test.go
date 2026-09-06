package secrets

import (
	"context"
	"testing"
)

type countingSource struct {
	calls int
}

func (s *countingSource) GetString(context.Context, string) (string, error) {
	s.calls++
	return "secret", nil
}

func TestCachedStringSourceReadsEachSecretOnce(t *testing.T) {
	source := &countingSource{}
	cache := NewCachedStringSource(source)
	for range 2 {
		value, err := cache.GetString(context.Background(), "arn:example")
		if err != nil || value != "secret" {
			t.Fatalf("GetString() = %q, %v", value, err)
		}
	}
	if source.calls != 1 {
		t.Fatalf("source calls = %d, want 1", source.calls)
	}
}

func TestCachedStringSourceRequiresConfiguration(t *testing.T) {
	cache := NewCachedStringSource(nil)
	if _, err := cache.GetString(context.Background(), "arn:example"); err == nil {
		t.Fatal("expected missing source error")
	}
	if _, err := cache.GetString(context.Background(), ""); err == nil {
		t.Fatal("expected missing secret ID error")
	}
}
