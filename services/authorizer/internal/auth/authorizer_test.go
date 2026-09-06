package auth

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestLambdaAuthorizer(t *testing.T) {
	authorizer := NewLambdaAuthorizer(StaticToken("secret"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := APIGatewayAuthorizerRequest{Headers: map[string]string{"x-hermes-gateway-token": "Bearer secret"}}
	response, err := authorizer.Handle(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !response.IsAuthorized || response.Context["caller_id"] != "hermes" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if _, exists := response.Context["scopes"]; exists {
		t.Fatalf("authorizer must not issue scopes: %#v", response.Context)
	}

	request.Headers["x-hermes-gateway-token"] = "Bearer wrong"
	response, err = authorizer.Handle(context.Background(), request)
	if err != nil || response.IsAuthorized {
		t.Fatalf("denied response = %#v, %v", response, err)
	}

	request.Headers = map[string]string{"authorization": "Bearer secret"}
	response, err = authorizer.Handle(context.Background(), request)
	if err != nil || response.IsAuthorized {
		t.Fatalf("standard Authorization header must be ignored: %#v, %v", response, err)
	}
}
