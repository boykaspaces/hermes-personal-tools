package calendar

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	calendarprovider "github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/providers/calendar"
)

const maxConcurrentCalendarQueries = 5

type calendarQueryResult struct {
	events []calendarprovider.Event
	err    error
}

func (m *Module) queryEvents(ctx context.Context, request calendarprovider.ListRequest) ([]calendarprovider.Event, int, error) {
	if !request.AllReadableCalendars {
		events, err := m.provider.ListEvents(ctx, request)
		return events, 1, err
	}
	return m.queryAllReadableCalendars(ctx, request)
}

func (m *Module) queryAllReadableCalendars(ctx context.Context, request calendarprovider.ListRequest) ([]calendarprovider.Event, int, error) {
	calendarList, err := m.provider.ListCalendars(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("discover readable calendars: %w", err)
	}
	if calendarList.Truncated {
		return nil, 0, errors.New("readable calendar list is truncated; refusing to return an incomplete aggregate")
	}
	if len(calendarList.Calendars) == 0 {
		return []calendarprovider.Event{}, 0, nil
	}

	aggregateCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerCount := min(maxConcurrentCalendarQueries, len(calendarList.Calendars))
	jobs := make(chan calendarprovider.Calendar)
	results := make(chan calendarQueryResult, workerCount)
	var workers sync.WaitGroup
	workers.Add(workerCount)

	for range workerCount {
		go func() {
			defer workers.Done()
			for calendarEntry := range jobs {
				calendarRequest := request
				calendarRequest.CalendarID = calendarEntry.ID
				calendarRequest.AllReadableCalendars = false
				events, queryErr := m.provider.ListEvents(aggregateCtx, calendarRequest)
				select {
				case results <- calendarQueryResult{events: events, err: queryErr}:
				case <-aggregateCtx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, calendarEntry := range calendarList.Calendars {
			select {
			case jobs <- calendarEntry:
			case <-aggregateCtx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	allEvents := make([]calendarprovider.Event, 0)
	var firstError error
	for result := range results {
		if result.err != nil {
			if firstError == nil {
				firstError = result.err
				cancel()
			}
			continue
		}
		allEvents = append(allEvents, result.events...)
	}
	if firstError != nil {
		return nil, 0, fmt.Errorf("query readable calendar: %w", firstError)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	sort.Slice(allEvents, func(left, right int) bool {
		leftTime, leftOK := eventStartTime(allEvents[left].Start)
		rightTime, rightOK := eventStartTime(allEvents[right].Start)
		if leftOK && rightOK && !leftTime.Equal(rightTime) {
			return leftTime.Before(rightTime)
		}
		if leftOK != rightOK {
			return leftOK
		}
		if allEvents[left].Start != allEvents[right].Start {
			return allEvents[left].Start < allEvents[right].Start
		}
		if allEvents[left].CalendarID != allEvents[right].CalendarID {
			return allEvents[left].CalendarID < allEvents[right].CalendarID
		}
		return allEvents[left].ID < allEvents[right].ID
	})
	if len(allEvents) > request.MaxResults {
		allEvents = allEvents[:request.MaxResults]
	}
	return allEvents, len(calendarList.Calendars), nil
}

func eventStartTime(value string) (time.Time, bool) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, true
	}
	if parsed, err := time.Parse(time.DateOnly, value); err == nil {
		return parsed, true
	}
	return time.Time{}, false
}
