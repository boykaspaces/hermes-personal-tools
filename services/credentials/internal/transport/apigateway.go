package transport

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/app"
)

type APIGatewayV2Request struct {
	RawPath         string            `json:"rawPath"`
	RawQueryString  string            `json:"rawQueryString"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	RequestContext  struct {
		RequestID string `json:"requestId"`
		HTTP      struct {
			Method   string `json:"method"`
			SourceIP string `json:"sourceIp"`
		} `json:"http"`
		Authorizer struct {
			IAM struct {
				UserARN string `json:"userArn"`
			} `json:"iam"`
		} `json:"authorizer"`
	} `json:"requestContext"`
}

type APIGatewayV2Response struct {
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            string            `json:"body,omitempty"`
	IsBase64Encoded bool              `json:"isBase64Encoded"`
}

type LambdaHTTPHandler struct{ handler http.Handler }

func NewLambdaHTTPHandler(handler http.Handler) *LambdaHTTPHandler {
	return &LambdaHTTPHandler{handler: handler}
}

func (h *LambdaHTTPHandler) Handle(ctx context.Context, event APIGatewayV2Request) (APIGatewayV2Response, error) {
	body, err := eventBody(event)
	if err != nil {
		return APIGatewayV2Response{}, err
	}
	requestURL := &url.URL{Scheme: "https", Host: requestHost(event.Headers), Path: event.RawPath, RawQuery: event.RawQueryString}
	request, err := http.NewRequestWithContext(app.WithCaller(ctx, event.RequestContext.Authorizer.IAM.UserARN), event.RequestContext.HTTP.Method, requestURL.String(), body)
	if err != nil {
		return APIGatewayV2Response{}, fmt.Errorf("construct HTTP request: %w", err)
	}
	for name, value := range event.Headers {
		if strings.EqualFold(name, "Authorization") {
			continue
		}
		request.Header.Set(name, value)
	}
	request.RemoteAddr = event.RequestContext.HTTP.SourceIP
	response := &bufferedResponse{header: make(http.Header)}
	h.handler.ServeHTTP(response, request)
	return response.gatewayResponse(), nil
}

func eventBody(event APIGatewayV2Request) (io.Reader, error) {
	if !event.IsBase64Encoded {
		return strings.NewReader(event.Body), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(event.Body)
	if err != nil {
		return nil, fmt.Errorf("decode request body: %w", err)
	}
	return bytes.NewReader(decoded), nil
}

func requestHost(headers map[string]string) string {
	for name, value := range headers {
		if strings.EqualFold(name, "host") && value != "" {
			return value
		}
	}
	return "credential-lease.local"
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *bufferedResponse) Header() http.Header { return w.header }
func (w *bufferedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *bufferedResponse) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}
func (w *bufferedResponse) gatewayResponse() APIGatewayV2Response {
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	headers := make(map[string]string, len(w.header))
	for name, values := range w.header {
		headers[name] = strings.Join(values, ",")
	}
	return APIGatewayV2Response{StatusCode: status, Headers: headers, Body: w.body.String()}
}
