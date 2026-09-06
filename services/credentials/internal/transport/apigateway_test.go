package transport

import (
	"context"
	"net/http"
	"testing"

	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/app"
)

func TestIAMIdentityIsPropagatedAndAuthorizationHeaderIsRemoved(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		caller, ok := app.CallerFromContext(request.Context())
		if !ok || caller != "arn:aws:sts::123456789012:assumed-role/hermes/session" {
			t.Errorf("caller = %q, ok = %v", caller, ok)
		}
		if got := request.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization reached application: %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	adapter := NewLambdaHTTPHandler(handler)
	event := APIGatewayV2Request{
		RawPath: "/v1/credential-leases",
		Headers: map[string]string{"host": "example.execute-api.amazonaws.com", "authorization": "sensitive-sigv4"},
	}
	event.RequestContext.HTTP.Method = http.MethodPost
	event.RequestContext.Authorizer.IAM.UserARN = "arn:aws:sts::123456789012:assumed-role/hermes/session"
	response, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", response.StatusCode)
	}
}
