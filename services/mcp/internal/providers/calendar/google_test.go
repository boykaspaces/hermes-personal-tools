package calendar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestGoogleProviderListCalendarsReturnsEventReadableCalendars(t *testing.T) {
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/calendar/v3/users/me/calendarList" {
			http.NotFound(response, request)
			return
		}
		if request.URL.Query().Get("minAccessRole") != "reader" || request.URL.Query().Get("showHidden") != "true" {
			t.Errorf("unexpected query: %s", request.URL.RawQuery)
		}
		writeGoogleJSON(t, response, map[string]any{
			"nextPageToken": "more",
			"items": []map[string]any{
				{
					"id":              "primary@example.com",
					"summary":         "Primary",
					"summaryOverride": "My calendar",
					"timeZone":        "Asia/Singapore",
					"accessRole":      "owner",
					"primary":         true,
					"selected":        true,
				},
				{
					"id":         "birthdays@example.com",
					"summary":    "Birthdays",
					"accessRole": "reader",
				},
				{
					"id":         "freebusy@example.com",
					"summary":    "Free/busy only",
					"accessRole": "freeBusyReader",
				},
			},
		})
	})

	provider := newTestGoogleProvider(t, handler)
	result, err := provider.ListCalendars(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Calendars) != 2 || !result.Truncated {
		t.Fatalf("unexpected calendar list: %#v", result)
	}
	if result.Calendars[0].Summary != "My calendar" || result.Calendars[1].ID != "birthdays@example.com" {
		t.Fatalf("unexpected calendars: %#v", result.Calendars)
	}
}

func TestGoogleProviderListEventsRechecksReaderAccess(t *testing.T) {
	eventsCalled := false
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case strings.HasPrefix(request.URL.Path, "/calendar/v3/users/me/calendarList/"):
			writeGoogleJSON(t, response, map[string]any{
				"id":         "birthdays@example.com",
				"summary":    "Birthdays",
				"accessRole": "reader",
			})
		case strings.HasPrefix(request.URL.Path, "/calendar/v3/calendars/") && strings.HasSuffix(request.URL.Path, "/events"):
			eventsCalled = true
			writeGoogleJSON(t, response, map[string]any{
				"items": []map[string]any{
					{
						"id":          "birthday-1",
						"summary":     "Birthday",
						"description": "Remember the cake",
						"location":    "Singapore",
						"eventType":   "birthday",
						"birthdayProperties": map[string]string{
							"contact":        "people/c12345",
							"type":           "custom",
							"customTypeName": "Lunar birthday",
						},
						"start":  map[string]string{"date": "2026-09-03"},
						"end":    map[string]string{"date": "2026-09-04"},
						"status": "confirmed",
					},
				},
			})
		default:
			http.NotFound(response, request)
		}
	})

	provider := newTestGoogleProvider(t, handler)
	events, err := provider.ListEvents(context.Background(), ListRequest{
		CalendarID: "birthdays@example.com",
		TimeMin:    time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
		TimeMax:    time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC),
		MaxResults: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !eventsCalled || len(events) != 1 || events[0].Start != "2026-09-03" || events[0].CalendarID != "birthdays@example.com" || events[0].CalendarSummary != "Birthdays" {
		t.Fatalf("unexpected events: %#v", events)
	}
	if events[0].Description != "Remember the cake" || events[0].Location != "Singapore" || events[0].EventType != "birthday" {
		t.Fatalf("missing event details: %#v", events[0])
	}
	if events[0].BirthdayProperties == nil || events[0].BirthdayProperties.Contact != "people/c12345" || events[0].BirthdayProperties.Type != "custom" || events[0].BirthdayProperties.CustomTypeName != "Lunar birthday" {
		t.Fatalf("missing birthday properties: %#v", events[0].BirthdayProperties)
	}
}

func TestGoogleProviderListEventsRejectsFreeBusyOnlyCalendar(t *testing.T) {
	eventsCalled := false
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case strings.HasPrefix(request.URL.Path, "/calendar/v3/users/me/calendarList/"):
			writeGoogleJSON(t, response, map[string]any{
				"id":         "room@example.com",
				"accessRole": "freeBusyReader",
			})
		case strings.HasPrefix(request.URL.Path, "/calendar/v3/calendars/"):
			eventsCalled = true
			http.Error(response, "must not be called", http.StatusInternalServerError)
		default:
			http.NotFound(response, request)
		}
	})

	provider := newTestGoogleProvider(t, handler)
	_, err := provider.ListEvents(context.Background(), ListRequest{
		CalendarID: "room@example.com",
		TimeMin:    time.Now(),
		TimeMax:    time.Now().Add(time.Hour),
		MaxResults: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "not readable") {
		t.Fatalf("expected readability error, got %v", err)
	}
	if eventsCalled {
		t.Fatal("events.list must not run for a freeBusyReader calendar")
	}
}

func newTestGoogleProvider(t *testing.T, handler http.Handler) *GoogleProvider {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Result(), nil
	})}
	service, err := googlecalendar.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}
	service.BasePath = "https://calendar.test/calendar/v3/"
	return &GoogleProvider{service: service}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func writeGoogleJSON(t *testing.T, response http.ResponseWriter, value any) {
	t.Helper()
	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(value); err != nil {
		t.Fatal(err)
	}
}
