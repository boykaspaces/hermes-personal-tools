package calendar

import (
	"context"
	"fmt"
	"time"
)

// MockProvider makes protocol, feature and deployment testing possible
// before a Google OAuth credential is stored in Secrets Manager.
type MockProvider struct{}

func (MockProvider) Name() string { return "mock" }

func (MockProvider) ListCalendars(context.Context) (CalendarList, error) {
	return CalendarList{
		Calendars: mockCalendars(),
	}, nil
}

func (MockProvider) ListEvents(_ context.Context, request ListRequest) ([]Event, error) {
	calendarEntry, ok := mockCalendar(request.CalendarID)
	if !ok {
		return nil, fmt.Errorf("unsupported calendar %q", request.CalendarID)
	}
	start := request.TimeMin.Add(2 * time.Hour).UTC()
	events := []Event{
		{
			ID:              "mock-event-1",
			CalendarID:      calendarEntry.ID,
			CalendarSummary: calendarEntry.Summary,
			Summary:         "Personal Tools MCP compatibility check",
			Description:     "Mock event description",
			Location:        "Mock location",
			EventType:       "default",
			Start:           start.Format(time.RFC3339),
			End:             start.Add(30 * time.Minute).Format(time.RFC3339),
			Status:          "confirmed",
		},
	}
	if request.MaxResults < len(events) {
		events = events[:request.MaxResults]
	}
	return events, nil
}

func mockCalendars() []Calendar {
	return []Calendar{
		{
			ID:         "primary",
			Summary:    "Mock primary calendar",
			TimeZone:   "UTC",
			AccessRole: "owner",
			Primary:    true,
			Selected:   true,
		},
		{
			ID:         "birthdays@mock.invalid",
			Summary:    "Mock birthdays calendar",
			TimeZone:   "UTC",
			AccessRole: "reader",
			Selected:   true,
		},
	}
}

func mockCalendar(calendarID string) (Calendar, bool) {
	for _, calendarEntry := range mockCalendars() {
		if calendarEntry.ID == calendarID {
			return calendarEntry, true
		}
	}
	return Calendar{}, false
}
