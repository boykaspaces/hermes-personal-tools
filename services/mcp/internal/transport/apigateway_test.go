package transport

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/app"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/catalog"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/identity"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/secrets"
)

func TestLambdaHTTPHandlerInjectsCaller(t *testing.T) {
	handler := NewLambdaHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get(ClientTokenHeader) != "" || request.Header.Get("Authorization") != "" {
			http.Error(w, "sensitive authentication header reached application", http.StatusInternalServerError)
			return
		}
		caller, ok := identity.FromContext(request.Context())
		if !ok || caller.ID != "hermes" {
			http.Error(w, "missing caller", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	event := APIGatewayV2Request{
		RawPath: "/mcp",
		Headers: map[string]string{
			ClientTokenHeader: "Bearer secret",
			"Authorization":   "Bearer must-also-be-removed",
		},
	}
	event.RequestContext.HTTP.Method = http.MethodPost
	event.RequestContext.Authorizer.Lambda = map[string]any{
		"caller_id": "hermes",
	}
	response, err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || response.Body != `{"ok":true}` {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestLambdaHTTPHandlerMCPInitialize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{
		Calendar: config.Feature{Mode: config.ProviderMock},
		Gmail:    config.Feature{Mode: config.ProviderDisabled},
		AWSOps:   config.Feature{Mode: config.ProviderDisabled},
	}
	configuredFeatures, err := features.Build(context.Background(), catalog.Definitions(
		cfg,
		secrets.NewCachedStringSource(nil),
		audit.New(logger),
	))
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLambdaHTTPHandler(app.New(configuredFeatures, logger))
	event := APIGatewayV2Request{
		RawPath: "/mcp",
		Headers: map[string]string{
			"content-type": "application/json",
			"accept":       "application/json, text/event-stream",
			"host":         "example.execute-api.ap-southeast-1.amazonaws.com",
		},
		Body: `{
			"jsonrpc":"2.0",
			"id":1,
			"method":"initialize",
			"params":{
				"protocolVersion":"2025-03-26",
				"capabilities":{},
				"clientInfo":{"name":"lambda-transport-test","version":"1"}
			}
		}`,
	}
	event.RequestContext.HTTP.Method = http.MethodPost
	event.RequestContext.Authorizer.Lambda = map[string]any{
		"caller_id": "hermes",
	}

	response, err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.StatusCode, response.Body)
	}
	var payload struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(response.Body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Result.ProtocolVersion != "2025-03-26" {
		t.Fatalf("protocol version = %q", payload.Result.ProtocolVersion)
	}
}
