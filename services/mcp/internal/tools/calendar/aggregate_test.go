package calendar

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	calendarprovider "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/calendar"
)

func TestQueryAllReadableCalendarsSortsAndAppliesGlobalLimit(t *testing.T) {
	provider := &aggregateTestProvider{
		calendarList: calendarprovider.CalendarList{Calendars: []calendarprovider.Calendar{
			{ID: "work", Summary: "Work"},
			{ID: "family", Summary: "Family"},
		}},
		events: map[string][]calendarprovider.Event{
			"work": {
				{ID: "work-later", CalendarID: "work", Start: "2026-09-03T12:00:00Z"},
			},
			"family": {
				{ID: "family-first", CalendarID: "family", Start: "2026-09-03T08:00:00Z"},
				{ID: "family-second", CalendarID: "family", Start: "2026-09-03T10:00:00Z"},
			},
		},
	}
	module := New(provider, audit.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	events, calendarsQueried, err := module.queryEvents(context.Background(), calendarprovider.ListRequest{
		AllReadableCalendars: true,
		TimeMin:              time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
		TimeMax:              time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC),
		MaxResults:           2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calendarsQueried != 2 || len(events) != 2 {
		t.Fatalf("queried=%d events=%#v", calendarsQueried, events)
	}
	if events[0].ID != "family-first" || events[1].ID != "family-second" {
		t.Fatalf("events are not globally sorted and limited: %#v", events)
	}
	for _, request := range provider.Requests() {
		if request.AllReadableCalendars || request.MaxResults != 2 || request.CalendarID == "" {
			t.Fatalf("unexpected provider request: %#v", request)
		}
	}
}

func TestQueryAllReadableCalendarsRejectsTruncatedDiscovery(t *testing.T) {
	provider := &aggregateTestProvider{
		calendarList: calendarprovider.CalendarList{
			Calendars: []calendarprovider.Calendar{{ID: "primary"}},
			Truncated: true,
		},
	}
	module := New(provider, audit.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	_, _, err := module.queryEvents(context.Background(), calendarprovider.ListRequest{
		AllReadableCalendars: true,
		MaxResults:           10,
	})
	if err == nil || !strings.Contains(err.Error(), "incomplete aggregate") {
		t.Fatalf("expected truncated discovery error, got %v", err)
	}
	if len(provider.Requests()) != 0 {
		t.Fatal("event queries must not start from a truncated calendar list")
	}
}

func TestQueryAllReadableCalendarsIsAllOrNothing(t *testing.T) {
	provider := &aggregateTestProvider{
		calendarList: calendarprovider.CalendarList{Calendars: []calendarprovider.Calendar{
			{ID: "primary"},
			{ID: "revoked"},
		}},
		events: map[string][]calendarprovider.Event{
			"primary": {{ID: "event", CalendarID: "primary", Start: "2026-09-03"}},
		},
		errors: map[string]error{
			"revoked": errors.New("access revoked"),
		},
	}
	module := New(provider, audit.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	events, calendarsQueried, err := module.queryEvents(context.Background(), calendarprovider.ListRequest{
		AllReadableCalendars: true,
		MaxResults:           10,
	})
	if err == nil || !strings.Contains(err.Error(), "access revoked") {
		t.Fatalf("expected provider error, got %v", err)
	}
	if events != nil || calendarsQueried != 0 {
		t.Fatalf("partial aggregate must not be returned: queried=%d events=%#v", calendarsQueried, events)
	}
}

type aggregateTestProvider struct {
	calendarList calendarprovider.CalendarList
	events       map[string][]calendarprovider.Event
	errors       map[string]error

	mutex    sync.Mutex
	requests []calendarprovider.ListRequest
}

func (provider *aggregateTestProvider) Name() string { return "aggregate_test" }

func (provider *aggregateTestProvider) ListCalendars(context.Context) (calendarprovider.CalendarList, error) {
	return provider.calendarList, nil
}

func (provider *aggregateTestProvider) ListEvents(_ context.Context, request calendarprovider.ListRequest) ([]calendarprovider.Event, error) {
	provider.mutex.Lock()
	provider.requests = append(provider.requests, request)
	provider.mutex.Unlock()
	if err := provider.errors[request.CalendarID]; err != nil {
		return nil, err
	}
	return provider.events[request.CalendarID], nil
}

func (provider *aggregateTestProvider) Requests() []calendarprovider.ListRequest {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()
	return append([]calendarprovider.ListRequest(nil), provider.requests...)
}
