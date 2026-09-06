package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/identity"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/awsops"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/calendar"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/gmail"
	awsopstool "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/tools/awsops"
	calendartool "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/tools/calendar"
	gmailtool "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/tools/gmail"
)

func TestLegacyProtocolStatelessLifecycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := testHandler(logger)

	initialize := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-03-26",
			"capabilities":{},
			"clientInfo":{"name":"compatibility-test","version":"1.0.0"}
		}
	}`)
	if got := nestedString(t, initialize, "result", "protocolVersion"); got != "2025-03-26" {
		t.Fatalf("negotiated protocol = %q", got)
	}

	list := postMCP(t, handler, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	tools := nestedSlice(t, list, "result", "tools")
	if len(tools) != 8 {
		t.Fatalf("tools/list returned %d tools: %#v", len(tools), list)
	}
	toolNames := make(map[string]bool, len(tools))
	for _, value := range tools {
		tool, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("unexpected tool: %#v", value)
		}
		name, _ := tool["name"].(string)
		toolNames[name] = true
	}
	if !toolNames[calendartool.ListCalendarsToolName] || !toolNames[calendartool.ListEventsToolName] || !toolNames[gmailtool.ToolName] ||
		!toolNames[awsopstool.PersonalToolsStatusToolName] || !toolNames[awsopstool.HermesStatusToolName] || !toolNames[awsopstool.RecentErrorsToolName] || !toolNames[awsopstool.CostSummaryToolName] || !toolNames[awsopstool.BudgetsToolName] ||
		toolNames["gmail_send_email"] || toolNames["host_run_command"] {
		t.Fatalf("unexpected tool registry: %#v", toolNames)
	}

	calendarListCall := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":3,
		"method":"tools/call",
		"params":{"name":"calendar_list_calendars","arguments":{}}
	}`)
	calendarListResult, ok := calendarListCall["result"].(map[string]any)
	if !ok {
		t.Fatalf("calendar list tools/call missing result: %#v", calendarListCall)
	}
	calendarListStructured, ok := calendarListResult["structuredContent"].(map[string]any)
	calendars, calendarsOK := calendarListStructured["calendars"].([]any)
	if !ok || !calendarsOK || len(calendars) != 2 || calendarListStructured["read_only"] != true {
		t.Fatalf("unexpected calendar list result: %#v", calendarListResult)
	}

	aggregateCall := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":4,
		"method":"tools/call",
		"params":{
			"name":"calendar_list_events",
			"arguments":{"all_readable_calendars":true,"max_results":5}
		}
	}`)
	aggregateResult, ok := aggregateCall["result"].(map[string]any)
	if !ok {
		t.Fatalf("aggregate tools/call missing result: %#v", aggregateCall)
	}
	aggregateStructured, ok := aggregateResult["structuredContent"].(map[string]any)
	aggregateEvents, eventsOK := aggregateStructured["events"].([]any)
	if !ok || !eventsOK || len(aggregateEvents) != 2 || aggregateStructured["all_readable_calendars"] != true || aggregateStructured["calendars_queried"] != float64(2) {
		t.Fatalf("unexpected aggregate result: %#v", aggregateResult)
	}

	call := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":5,
		"method":"tools/call",
		"params":{
			"name":"calendar_list_events",
			"arguments":{"calendar_id":"primary","max_results":5}
		}
	}`)
	result, ok := call["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/call missing result: %#v", call)
	}
	if isError, _ := result["isError"].(bool); isError {
		t.Fatalf("tools/call returned an error: %#v", call)
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok || structured["provider"] != "mock" || structured["read_only"] != true {
		t.Fatalf("unexpected structured result: %#v", result)
	}

	draftCall := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":6,
		"method":"tools/call",
		"params":{
			"name":"gmail_create_draft",
			"arguments":{"to":["alice@example.com"],"subject":"Review me","body_text":"Draft only"}
		}
	}`)
	draftResult, ok := draftCall["result"].(map[string]any)
	if !ok {
		t.Fatalf("draft tools/call missing result: %#v", draftCall)
	}
	draftStructured, ok := draftResult["structuredContent"].(map[string]any)
	if !ok || draftStructured["status"] != "draft" || draftStructured["sent"] != false {
		t.Fatalf("unexpected draft result: %#v", draftResult)
	}

	costCall := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":7,
		"method":"tools/call",
		"params":{"name":"awsops_get_cost_summary","arguments":{"lookback_months":2,"max_services":5}}
	}`)
	costResult, ok := costCall["result"].(map[string]any)
	costStructured, structuredOK := costResult["structuredContent"].(map[string]any)
	if !ok || !structuredOK || costStructured["provider"] != "mock" || costStructured["read_only"] != true {
		t.Fatalf("unexpected cost result: %#v", costCall)
	}

	budgetCall := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":8,
		"method":"tools/call",
		"params":{"name":"awsops_get_budgets","arguments":{}}
	}`)
	budgetResult, ok := budgetCall["result"].(map[string]any)
	budgetStructured, structuredOK := budgetResult["structuredContent"].(map[string]any)
	budgets, budgetsOK := budgetStructured["budgets"].([]any)
	if !ok || !structuredOK || !budgetsOK || len(budgets) != 2 || budgetStructured["read_only"] != true {
		t.Fatalf("unexpected budget result: %#v", budgetCall)
	}
}

func TestToolCallRequiresAuthenticatedCaller(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := testHandler(logger)
	request := newMCPRequest(t, `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"tools/call",
		"params":{"name":"calendar_list_events","arguments":{}}
	}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	result, _ := payload["result"].(map[string]any)
	if isError, _ := result["isError"].(bool); !isError {
		t.Fatalf("expected MCP tool error, got %#v", payload)
	}
}

func TestSendEmailToolIsNotExposed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := testHandler(logger)
	response := postMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"tools/call",
		"params":{"name":"gmail_send_email","arguments":{}}
	}`)
	errorObject, _ := response["error"].(map[string]any)
	message, _ := errorObject["message"].(string)
	if !strings.Contains(message, "unknown tool") {
		t.Fatalf("unregistered send tool must fail: %#v", response)
	}
}

func testHandler(logger *slog.Logger) http.Handler {
	recorder := audit.New(logger)
	return NewHandler([]Module{
		calendartool.New(calendar.MockProvider{}, recorder),
		gmailtool.New(gmail.MockProvider{}, recorder),
		awsopstool.New(awsops.MockProvider{}, recorder),
	}, logger)
}

func postMCP(t *testing.T, handler http.Handler, body string) map[string]any {
	t.Helper()
	request := newMCPRequest(t, body)
	request = request.WithContext(identity.WithCaller(context.Background(), identity.New("hermes")))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, response.Body.String())
	}
	return payload
}

func newMCPRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, "https://personal-tools.test/mcp", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	return request
}

func nestedString(t *testing.T, value map[string]any, keys ...string) string {
	t.Helper()
	current := any(value)
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("%q is not an object in %#v", key, current)
		}
		current = object[key]
	}
	result, ok := current.(string)
	if !ok {
		t.Fatalf("value is not a string: %#v", current)
	}
	return result
}

func nestedSlice(t *testing.T, value map[string]any, keys ...string) []any {
	t.Helper()
	current := any(value)
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("%q is not an object in %#v", key, current)
		}
		current = object[key]
	}
	result, ok := current.([]any)
	if !ok {
		t.Fatalf("value is not a slice: %#v", current)
	}
	return result
}
