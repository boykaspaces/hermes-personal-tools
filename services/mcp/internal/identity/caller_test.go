package identity

import (
	"context"
	"errors"
	"testing"
)

func TestCallerContextRoundTrip(t *testing.T) {
	want := New("hermes")
	got, ok := FromContext(WithCaller(context.Background(), want))
	if !ok || got.ID != want.ID {
		t.Fatalf("FromContext() = %#v, %v", got, ok)
	}
}

func TestRequireRejectsMissingCaller(t *testing.T) {
	if _, err := Require(context.Background()); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Require() error = %v, want ErrUnauthenticated", err)
	}
}
