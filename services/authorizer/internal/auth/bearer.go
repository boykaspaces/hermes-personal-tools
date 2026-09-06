package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
)

const ClientTokenHeader = "X-Hermes-Gateway-Token"

var ErrInvalidBearerToken = errors.New("invalid bearer token")

type TokenSource interface {
	Get(context.Context) (string, error)
}

type StaticToken string

func (s StaticToken) Get(context.Context) (string, error) {
	if s == "" {
		return "", errors.New("static token is empty")
	}
	return string(s), nil
}

func ValidateBearer(ctx context.Context, headerValue string, source TokenSource) error {
	scheme, presented, found := strings.Cut(strings.TrimSpace(headerValue), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || presented == "" {
		return ErrInvalidBearerToken
	}
	expected, err := source.Get(ctx)
	if err != nil {
		return err
	}
	if len(presented) != len(expected) || subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
		return ErrInvalidBearerToken
	}
	return nil
}
