package auth

import (
	"context"
	"errors"
	"log/slog"
)

type APIGatewayAuthorizerRequest struct {
	Headers        map[string]string `json:"headers"`
	IdentitySource []string          `json:"identitySource"`
	RequestContext struct {
		RequestID string `json:"requestId"`
	} `json:"requestContext"`
}

type APIGatewayAuthorizerResponse struct {
	IsAuthorized bool           `json:"isAuthorized"`
	Context      map[string]any `json:"context,omitempty"`
}

type LambdaAuthorizer struct {
	tokens TokenSource
	logger *slog.Logger
}

func NewLambdaAuthorizer(tokens TokenSource, logger *slog.Logger) *LambdaAuthorizer {
	return &LambdaAuthorizer{tokens: tokens, logger: logger}
}

func (a *LambdaAuthorizer) Handle(ctx context.Context, request APIGatewayAuthorizerRequest) (APIGatewayAuthorizerResponse, error) {
	clientToken := header(request.Headers, ClientTokenHeader)
	if err := ValidateBearer(ctx, clientToken, a.tokens); err != nil {
		reason := "token_source_unavailable"
		if errors.Is(err, ErrInvalidBearerToken) {
			reason = "invalid_token"
		}
		a.logger.WarnContext(ctx, "authorization_denied",
			"request_id", request.RequestContext.RequestID,
			"reason", reason,
		)
		return APIGatewayAuthorizerResponse{IsAuthorized: false}, nil
	}

	return APIGatewayAuthorizerResponse{
		IsAuthorized: true,
		Context: map[string]any{
			"caller_id": "hermes",
		},
	}, nil
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if equalFoldASCII(key, name) {
			return value
		}
	}
	return ""
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
