package awsops

import (
	"strings"
	"testing"
	"time"
)

func TestValidateRecentErrorsInput(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	request, err := ValidateRecentErrorsInput(RecentErrorsInput{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if request.MaxResults != 10 || !request.Since.Equal(now.Add(-time.Hour)) {
		t.Fatalf("unexpected defaults: %#v", request)
	}

	invalid := []RecentErrorsInput{
		{LookbackMinutes: -1},
		{LookbackMinutes: 1441},
		{MaxResults: -1},
		{MaxResults: 21},
	}
	for _, input := range invalid {
		if _, err := ValidateRecentErrorsInput(input, now); err == nil {
			t.Fatalf("expected error for %#v", input)
		}
	}
}

func TestValidateCostSummaryInput(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.FixedZone("SGT", 8*60*60))
	request, err := ValidateCostSummaryInput(CostSummaryInput{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if request.MaxServices != 10 || request.Start.Format(time.DateOnly) != "2026-06-01" || request.MonthStart.Format(time.DateOnly) != "2026-08-01" || request.End.Format(time.DateOnly) != "2026-08-23" {
		t.Fatalf("unexpected defaults: %#v", request)
	}

	invalid := []CostSummaryInput{
		{LookbackMonths: -1},
		{LookbackMonths: 13},
		{MaxServices: -1},
		{MaxServices: 21},
	}
	for _, input := range invalid {
		if _, err := ValidateCostSummaryInput(input, now); err == nil {
			t.Fatalf("expected error for %#v", input)
		}
	}
}

func TestPercentage(t *testing.T) {
	if got := percentage("8.75", "20"); got != "43.75" {
		t.Fatalf("percentage = %q", got)
	}
	if got := percentage("1", "0"); got != "" {
		t.Fatalf("zero-limit percentage = %q", got)
	}
}

func TestSanitizeLogMessage(t *testing.T) {
	message := sanitizeLogMessage("  error\x00\x1b[31m\nline  ")
	if message != "error[31m\nline" {
		t.Fatalf("unexpected sanitized message %q", message)
	}
	long := sanitizeLogMessage(strings.Repeat("a", maxLogMessageRunes+10))
	if len([]rune(long)) != maxLogMessageRunes+1 {
		t.Fatalf("long message was not bounded: %d runes", len([]rune(long)))
	}
}
