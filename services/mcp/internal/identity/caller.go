package identity

import (
	"context"
	"errors"
)

type contextKey struct{}

// Caller is the authenticated identity supplied by API Gateway after the
// dedicated Authorizer has accepted the request. Provider credentials are
// deliberately never attached to this value.
type Caller struct {
	ID string
}

func New(id string) Caller {
	return Caller{ID: id}
}

func WithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, contextKey{}, caller)
}

func FromContext(ctx context.Context) (Caller, bool) {
	caller, ok := ctx.Value(contextKey{}).(Caller)
	return caller, ok
}

var ErrUnauthenticated = errors.New("caller is not authenticated")

func Require(ctx context.Context) (Caller, error) {
	caller, ok := FromContext(ctx)
	if !ok || caller.ID == "" {
		return Caller{}, ErrUnauthenticated
	}
	return caller, nil
}
