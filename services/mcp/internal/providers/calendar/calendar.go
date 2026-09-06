package calendar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxCalendarIDRunes = 1024

type Calendar struct {
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	TimeZone   string `json:"time_zone,omitempty"`
	AccessRole string `json:"access_role"`
	Primary    bool   `json:"primary"`
	Selected   bool   `json:"selected"`
}

type CalendarList struct {
	Calendars []Calendar
	Truncated bool
}

type ListCalendarsOutput struct {
	Calendars             []Calendar `json:"calendars"`
	Truncated             bool       `json:"truncated"`
	ReadOnly              bool       `json:"read_only"`
	Provider              string     `json:"provider"`
	UntrustedExternalData bool       `json:"untrusted_external_data"`
}

type ListEventsInput struct {
	CalendarID           string `json:"calendar_id,omitempty" jsonschema:"calendar identifier returned by calendar_list_calendars; defaults to primary when all_readable_calendars is false"`
	AllReadableCalendars bool   `json:"all_readable_calendars,omitempty" jsonschema:"query and merge events from every event-readable calendar; cannot be combined with calendar_id"`
	TimeMin              string `json:"time_min,omitempty" jsonschema:"inclusive RFC3339 lower bound"`
	TimeMax              string `json:"time_max,omitempty" jsonschema:"exclusive RFC3339 upper bound"`
	MaxResults           int    `json:"max_results,omitempty" jsonschema:"global maximum number of events from 1 to 50"`
}

type ListRequest struct {
	CalendarID           string
	AllReadableCalendars bool
	TimeMin              time.Time
	TimeMax              time.Time
	MaxResults           int
}

type Event struct {
	ID                 string              `json:"id"`
	CalendarID         string              `json:"calendar_id"`
	CalendarSummary    string              `json:"calendar_summary"`
	Summary            string              `json:"summary"`
	Description        string              `json:"description,omitempty"`
	Location           string              `json:"location,omitempty"`
	EventType          string              `json:"event_type,omitempty"`
	BirthdayProperties *BirthdayProperties `json:"birthday_properties,omitempty"`
	Start              string              `json:"start"`
	End                string              `json:"end"`
	Status             string              `json:"status"`
}

type BirthdayProperties struct {
	Contact        string `json:"contact,omitempty"`
	Type           string `json:"type,omitempty"`
	CustomTypeName string `json:"custom_type_name,omitempty"`
}

type ListEventsOutput struct {
	CalendarID            string  `json:"calendar_id,omitempty"`
	AllReadableCalendars  bool    `json:"all_readable_calendars"`
	CalendarsQueried      int     `json:"calendars_queried"`
	Events                []Event `json:"events"`
	ReadOnly              bool    `json:"read_only"`
	Provider              string  `json:"provider"`
	UntrustedExternalData bool    `json:"untrusted_external_data"`
}

type Provider interface {
	ListCalendars(context.Context) (CalendarList, error)
	ListEvents(context.Context, ListRequest) ([]Event, error)
	Name() string
}

func ValidateListInput(input ListEventsInput, now time.Time) (ListRequest, error) {
	calendarID := input.CalendarID
	if input.AllReadableCalendars && calendarID != "" {
		return ListRequest{}, errors.New("calendar_id cannot be combined with all_readable_calendars=true")
	}
	if !input.AllReadableCalendars && calendarID == "" {
		calendarID = "primary"
	}
	if calendarID != "" && strings.TrimSpace(calendarID) != calendarID {
		return ListRequest{}, errors.New("calendar_id must not have surrounding whitespace")
	}
	if utf8.RuneCountInString(calendarID) > maxCalendarIDRunes {
		return ListRequest{}, fmt.Errorf("calendar_id cannot exceed %d characters", maxCalendarIDRunes)
	}
	if strings.IndexFunc(calendarID, unicode.IsControl) >= 0 {
		return ListRequest{}, errors.New("calendar_id cannot contain control characters")
	}

	maxResults := input.MaxResults
	if maxResults == 0 {
		maxResults = 10
	}
	if maxResults < 1 || maxResults > 50 {
		return ListRequest{}, errors.New("max_results must be between 1 and 50")
	}

	timeMin := now.UTC()
	var err error
	if input.TimeMin != "" {
		timeMin, err = time.Parse(time.RFC3339, input.TimeMin)
		if err != nil {
			return ListRequest{}, fmt.Errorf("time_min must be RFC3339: %w", err)
		}
	}
	timeMax := timeMin.Add(7 * 24 * time.Hour)
	if input.TimeMax != "" {
		timeMax, err = time.Parse(time.RFC3339, input.TimeMax)
		if err != nil {
			return ListRequest{}, fmt.Errorf("time_max must be RFC3339: %w", err)
		}
	}
	if !timeMax.After(timeMin) {
		return ListRequest{}, errors.New("time_max must be later than time_min")
	}
	if timeMax.Sub(timeMin) > 31*24*time.Hour {
		return ListRequest{}, errors.New("requested time range cannot exceed 31 days")
	}

	return ListRequest{
		CalendarID:           calendarID,
		AllReadableCalendars: input.AllReadableCalendars,
		TimeMin:              timeMin.UTC(),
		TimeMax:              timeMax.UTC(),
		MaxResults:           maxResults,
	}, nil
}
