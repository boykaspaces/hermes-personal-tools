package calendar

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/googleauth"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	maxSummaryRunes          = 500
	maxEventDescriptionRunes = 2000
	maxEventLocationRunes    = 500
	maxContactReferenceRunes = 500
	maxCustomTypeNameRunes   = 200
	maxCalendarCount         = 250
)

type GoogleProvider struct {
	service *googlecalendar.Service
}

func NewGoogleProvider(ctx context.Context, secretJSON string) (*GoogleProvider, error) {
	httpClient, err := googleauth.NewHTTPClient(ctx, secretJSON, googlecalendar.CalendarReadonlyScope)
	if err != nil {
		return nil, err
	}
	service, err := googlecalendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("initialize Google Calendar client: %w", err)
	}
	return &GoogleProvider{service: service}, nil
}

func (p *GoogleProvider) Name() string { return "google_calendar" }

func (p *GoogleProvider) ListCalendars(ctx context.Context) (CalendarList, error) {
	response, err := p.service.CalendarList.List().
		Context(ctx).
		MinAccessRole("reader").
		ShowDeleted(false).
		ShowHidden(true).
		MaxResults(maxCalendarCount).
		Fields("nextPageToken,items(id,summary,summaryOverride,timeZone,accessRole,primary,selected)").
		Do()
	if err != nil {
		return CalendarList{}, fmt.Errorf("list Google calendars: %w", err)
	}

	calendars := make([]Calendar, 0, len(response.Items))
	for _, item := range response.Items {
		if item == nil || !canReadEventContent(item.AccessRole) {
			continue
		}
		calendars = append(calendars, Calendar{
			ID:         boundedText(item.Id, maxCalendarIDRunes),
			Summary:    calendarSummary(item),
			TimeZone:   boundedText(item.TimeZone, 100),
			AccessRole: boundedText(item.AccessRole, 32),
			Primary:    item.Primary,
			Selected:   item.Selected,
		})
	}
	return CalendarList{
		Calendars: calendars,
		Truncated: response.NextPageToken != "",
	}, nil
}

func (p *GoogleProvider) ListEvents(ctx context.Context, request ListRequest) ([]Event, error) {
	calendarEntry, err := p.service.CalendarList.Get(request.CalendarID).
		Context(ctx).
		Fields("id,summary,summaryOverride,accessRole").
		Do()
	if err != nil {
		return nil, fmt.Errorf("verify Google calendar readability: %w", err)
	}
	if !canReadEventContent(calendarEntry.AccessRole) {
		return nil, fmt.Errorf("calendar is not readable at event-content level (access role %q)", calendarEntry.AccessRole)
	}

	response, err := p.service.Events.List(request.CalendarID).
		Context(ctx).
		TimeMin(request.TimeMin.Format(timeFormatRFC3339)).
		TimeMax(request.TimeMax.Format(timeFormatRFC3339)).
		MaxResults(int64(request.MaxResults)).
		SingleEvents(true).
		OrderBy("startTime").
		ShowDeleted(false).
		Fields("items(id,summary,description,location,eventType,birthdayProperties,start,end,status)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list Google Calendar events: %w", err)
	}

	events := make([]Event, 0, len(response.Items))
	for _, item := range response.Items {
		if item == nil {
			continue
		}
		events = append(events, Event{
			ID:                 boundedText(item.Id, maxSummaryRunes),
			CalendarID:         boundedText(calendarEntry.Id, maxCalendarIDRunes),
			CalendarSummary:    calendarSummary(calendarEntry),
			Summary:            boundedText(item.Summary, maxSummaryRunes),
			Description:        boundedText(item.Description, maxEventDescriptionRunes),
			Location:           boundedText(item.Location, maxEventLocationRunes),
			EventType:          boundedText(item.EventType, 64),
			BirthdayProperties: birthdayProperties(item.BirthdayProperties),
			Start:              eventDateTime(item.Start),
			End:                eventDateTime(item.End),
			Status:             boundedText(item.Status, 32),
		})
	}
	return events, nil
}

func birthdayProperties(properties *googlecalendar.EventBirthdayProperties) *BirthdayProperties {
	if properties == nil {
		return nil
	}
	return &BirthdayProperties{
		Contact:        boundedText(properties.Contact, maxContactReferenceRunes),
		Type:           boundedText(properties.Type, 64),
		CustomTypeName: boundedText(properties.CustomTypeName, maxCustomTypeNameRunes),
	}
}

func calendarSummary(entry *googlecalendar.CalendarListEntry) string {
	if entry == nil {
		return ""
	}
	summary := entry.SummaryOverride
	if summary == "" {
		summary = entry.Summary
	}
	return boundedText(summary, maxSummaryRunes)
}

func canReadEventContent(accessRole string) bool {
	switch accessRole {
	case "reader", "writerWithoutPrivateAccess", "writer", "owner":
		return true
	default:
		return false
	}
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

func eventDateTime(value *googlecalendar.EventDateTime) string {
	if value == nil {
		return ""
	}
	if value.DateTime != "" {
		return value.DateTime
	}
	return value.Date
}

func boundedText(value string, maxRunes int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes])
}
