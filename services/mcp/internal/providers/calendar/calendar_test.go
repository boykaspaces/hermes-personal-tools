package calendar

import (
	"testing"
	"time"
)

func TestValidateListInputDefaults(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	got, err := ValidateListInput(ListEventsInput{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.CalendarID != "primary" || got.MaxResults != 10 {
		t.Fatalf("unexpected defaults: %#v", got)
	}
	if got.TimeMax.Sub(got.TimeMin) != 7*24*time.Hour {
		t.Fatalf("default range = %s", got.TimeMax.Sub(got.TimeMin))
	}
}

func TestValidateListInputRejectsWideRange(t *testing.T) {
	_, err := ValidateListInput(ListEventsInput{
		TimeMin: "2026-01-01T00:00:00Z",
		TimeMax: "2026-03-01T00:00:00Z",
	}, time.Now())
	if err == nil {
		t.Fatal("expected a range validation error")
	}
}

func TestValidateListInputAllowsDiscoveredCalendarID(t *testing.T) {
	got, err := ValidateListInput(ListEventsInput{
		CalendarID: "contacts#birthdays@group.v.calendar.google.com",
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got.CalendarID != "contacts#birthdays@group.v.calendar.google.com" {
		t.Fatalf("calendar id = %q", got.CalendarID)
	}
}

func TestValidateListInputAllowsAllReadableCalendars(t *testing.T) {
	got, err := ValidateListInput(ListEventsInput{
		AllReadableCalendars: true,
		MaxResults:           50,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !got.AllReadableCalendars || got.CalendarID != "" || got.MaxResults != 50 {
		t.Fatalf("unexpected aggregate request: %#v", got)
	}
}

func TestValidateListInputRejectsAllReadableCalendarsWithCalendarID(t *testing.T) {
	_, err := ValidateListInput(ListEventsInput{
		CalendarID:           "primary",
		AllReadableCalendars: true,
	}, time.Now())
	if err == nil {
		t.Fatal("expected mutually exclusive calendar selection error")
	}
}

func TestValidateListInputRejectsUnsafeCalendarID(t *testing.T) {
	for _, calendarID := range []string{" primary", "primary\n"} {
		_, err := ValidateListInput(ListEventsInput{CalendarID: calendarID}, time.Now())
		if err == nil {
			t.Fatalf("calendar id %q must be rejected", calendarID)
		}
	}
}
