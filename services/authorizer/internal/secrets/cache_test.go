package secrets

import (
	"context"
	"testing"
	"time"
)

func TestCachedString(t *testing.T) {
	calls := 0
	secret := NewCachedString(func(context.Context) (string, error) {
		calls++
		return "value", nil
	}, time.Minute)
	for range 2 {
		if got, err := secret.Get(context.Background()); err != nil || got != "value" {
			t.Fatalf("Get() = %q, %v", got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", calls)
	}
}
