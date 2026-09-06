package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/identity"
	calendarprovider "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/calendar"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ListCalendarsToolName = "calendar_list_calendars"
	ListEventsToolName    = "calendar_list_events"
)

type Module struct {
	provider calendarprovider.Provider
	recorder *audit.Recorder
}

func New(provider calendarprovider.Provider, recorder *audit.Recorder) *Module {
	return &Module{provider: provider, recorder: recorder}
}

func (m *Module) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        ListCalendarsToolName,
		Description: "List all calendars for which the Google account can read event content. Use a returned calendar id with calendar_list_events. This tool is read-only and runs automatically.",
	}, m.listCalendars)
	mcp.AddTool(server, &mcp.Tool{
		Name:        ListEventsToolName,
		Description: "List events from primary, one readable calendar, or every readable calendar when all_readable_calendars=true. The all-calendar mode merges, sorts, and applies max_results globally. This tool is read-only and runs automatically.",
	}, m.listEvents)
}

func (m *Module) listCalendars(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, calendarprovider.ListCalendarsOutput, error) {
	started := time.Now()
	caller, err := identity.Require(ctx)
	if err != nil {
		m.recorder.ToolCall(ctx, "unknown", ListCalendarsToolName, "denied", started, 0)
		return nil, calendarprovider.ListCalendarsOutput{}, err
	}

	calendarList, err := m.provider.ListCalendars(ctx)
	if err != nil {
		m.recorder.ToolCall(ctx, caller.ID, ListCalendarsToolName, "provider_error", started, 0)
		return nil, calendarprovider.ListCalendarsOutput{}, fmt.Errorf("calendar provider failed: %w", err)
	}

	output := calendarprovider.ListCalendarsOutput{
		Calendars:             calendarList.Calendars,
		Truncated:             calendarList.Truncated,
		ReadOnly:              true,
		Provider:              m.provider.Name(),
		UntrustedExternalData: m.provider.Name() != "mock",
	}
	m.recorder.ToolCall(ctx, caller.ID, ListCalendarsToolName, "success", started, len(calendarList.Calendars))
	return nil, output, nil
}

func (m *Module) listEvents(ctx context.Context, _ *mcp.CallToolRequest, input calendarprovider.ListEventsInput) (*mcp.CallToolResult, calendarprovider.ListEventsOutput, error) {
	started := time.Now()
	caller, err := identity.Require(ctx)
	if err != nil {
		m.recorder.ToolCall(ctx, "unknown", ListEventsToolName, "denied", started, 0)
		return nil, calendarprovider.ListEventsOutput{}, err
	}

	request, err := calendarprovider.ValidateListInput(input, time.Now())
	if err != nil {
		m.recorder.ToolCall(ctx, caller.ID, ListEventsToolName, "invalid", started, 0)
		return nil, calendarprovider.ListEventsOutput{}, fmt.Errorf("invalid calendar query: %w", err)
	}
	events, calendarsQueried, err := m.queryEvents(ctx, request)
	if err != nil {
		m.recorder.ToolCall(ctx, caller.ID, ListEventsToolName, "provider_error", started, 0)
		return nil, calendarprovider.ListEventsOutput{}, fmt.Errorf("calendar provider failed: %w", err)
	}

	output := calendarprovider.ListEventsOutput{
		CalendarID:            request.CalendarID,
		AllReadableCalendars:  request.AllReadableCalendars,
		CalendarsQueried:      calendarsQueried,
		Events:                events,
		ReadOnly:              true,
		Provider:              m.provider.Name(),
		UntrustedExternalData: m.provider.Name() != "mock",
	}
	m.recorder.ToolCall(ctx, caller.ID, ListEventsToolName, "success", started, len(events))
	return nil, output, nil
}
