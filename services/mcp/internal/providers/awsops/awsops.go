package awsops

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultLookbackMinutes = 60
	defaultMaxResults      = 10
	defaultLookbackMonths  = 3
	defaultMaxServices     = 10
	maxLookbackMinutes     = 24 * 60
	maxResults             = 20
	maxLookbackMonths      = 12
	maxServices            = 20
	maxLogMessageRunes     = 1000
)

type Targets struct {
	StackName              string
	MCPFunctionName        string
	AuthorizerFunctionName string
	HermesManagedNodeID    string
	LogGroupNames          []string
	AccountID              string
	BudgetNames            []string
}

type FunctionStatus struct {
	Name             string `json:"name"`
	State            string `json:"state"`
	LastUpdateStatus string `json:"last_update_status"`
	LastModified     string `json:"last_modified,omitempty"`
	MemoryMB         int32  `json:"memory_mb"`
	TimeoutSeconds   int32  `json:"timeout_seconds"`
}

type PersonalToolsStatus struct {
	StackName             string           `json:"stack_name"`
	StackStatus           string           `json:"stack_status"`
	StackCreatedAt        string           `json:"stack_created_at,omitempty"`
	StackLastUpdatedAt    string           `json:"stack_last_updated_at,omitempty"`
	TerminationProtection bool             `json:"termination_protection"`
	Functions             []FunctionStatus `json:"functions"`
}

type PersonalToolsStatusOutput struct {
	Status   PersonalToolsStatus `json:"status"`
	ReadOnly bool                `json:"read_only"`
	Provider string              `json:"provider"`
}

type HermesStatus struct {
	ManagedNodeID   string `json:"managed_node_id"`
	ConnectionState string `json:"connection_state"`
}

type HermesStatusOutput struct {
	Status   HermesStatus `json:"status"`
	ReadOnly bool         `json:"read_only"`
	Provider string       `json:"provider"`
}

type RecentErrorsInput struct {
	LookbackMinutes int `json:"lookback_minutes,omitempty" jsonschema:"lookback window from 1 to 1440 minutes; defaults to 60"`
	MaxResults      int `json:"max_results,omitempty" jsonschema:"global maximum number of error events from 1 to 20; defaults to 10"`
}

type RecentErrorsRequest struct {
	Since      time.Time
	MaxResults int
}

type LogError struct {
	LogGroup  string `json:"log_group"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

type RecentErrorsOutput struct {
	Since                    string     `json:"since"`
	Errors                   []LogError `json:"errors"`
	LogGroupsQueried         int        `json:"log_groups_queried"`
	ReadOnly                 bool       `json:"read_only"`
	Provider                 string     `json:"provider"`
	UntrustedOperationalData bool       `json:"untrusted_operational_data"`
}

type CostSummaryInput struct {
	LookbackMonths int `json:"lookback_months,omitempty" jsonschema:"number of calendar months from 1 to 12; defaults to 3 and includes the current partial month"`
	MaxServices    int `json:"max_services,omitempty" jsonschema:"maximum current-month services from 1 to 20; defaults to 10"`
}

type CostSummaryRequest struct {
	Start       time.Time
	End         time.Time
	MonthStart  time.Time
	MaxServices int
}

type CostPeriod struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Amount    string `json:"amount"`
	Unit      string `json:"unit"`
	Estimated bool   `json:"estimated"`
}

type ServiceCost struct {
	Service string `json:"service"`
	Amount  string `json:"amount"`
	Unit    string `json:"unit"`
}

type CostSummary struct {
	Metric                   string        `json:"metric"`
	Granularity              string        `json:"granularity"`
	Start                    string        `json:"start"`
	EndExclusive             string        `json:"end_exclusive"`
	MonthlyCosts             []CostPeriod  `json:"monthly_costs"`
	CurrentMonthTopServices  []ServiceCost `json:"current_month_top_services"`
	CurrentMonthServiceCount int           `json:"current_month_service_count"`
	ServicesTruncated        bool          `json:"services_truncated"`
	BillingDataDelayed       bool          `json:"billing_data_delayed"`
}

type CostSummaryOutput struct {
	Summary  CostSummary `json:"summary"`
	ReadOnly bool        `json:"read_only"`
	Provider string      `json:"provider"`
}

type Spend struct {
	Amount string `json:"amount"`
	Unit   string `json:"unit"`
}

type BudgetSummary struct {
	Name              string              `json:"name"`
	BudgetType        string              `json:"budget_type"`
	TimeUnit          string              `json:"time_unit"`
	Limit             Spend               `json:"limit"`
	ActualSpend       Spend               `json:"actual_spend"`
	ForecastedSpend   *Spend              `json:"forecasted_spend,omitempty"`
	ActualPercent     string              `json:"actual_percent,omitempty"`
	ForecastedPercent string              `json:"forecasted_percent,omitempty"`
	CostFilters       map[string][]string `json:"cost_filters,omitempty"`
	FilterExpression  string              `json:"filter_expression,omitempty"`
	LastUpdatedAt     string              `json:"last_updated_at,omitempty"`
}

type BudgetsOutput struct {
	Budgets  []BudgetSummary `json:"budgets"`
	ReadOnly bool            `json:"read_only"`
	Provider string          `json:"provider"`
}

type Provider interface {
	PersonalToolsStatus(context.Context) (PersonalToolsStatus, error)
	HermesStatus(context.Context) (HermesStatus, error)
	RecentErrors(context.Context, RecentErrorsRequest) ([]LogError, error)
	CostSummary(context.Context, CostSummaryRequest) (CostSummary, error)
	Budgets(context.Context) ([]BudgetSummary, error)
	LogGroupCount() int
	Name() string
}

func ValidateRecentErrorsInput(input RecentErrorsInput, now time.Time) (RecentErrorsRequest, error) {
	lookback := input.LookbackMinutes
	if lookback == 0 {
		lookback = defaultLookbackMinutes
	}
	if lookback < 1 || lookback > maxLookbackMinutes {
		return RecentErrorsRequest{}, errors.New("lookback_minutes must be between 1 and 1440")
	}
	limit := input.MaxResults
	if limit == 0 {
		limit = defaultMaxResults
	}
	if limit < 1 || limit > maxResults {
		return RecentErrorsRequest{}, errors.New("max_results must be between 1 and 20")
	}
	return RecentErrorsRequest{
		Since:      now.UTC().Add(-time.Duration(lookback) * time.Minute),
		MaxResults: limit,
	}, nil
}

func ValidateCostSummaryInput(input CostSummaryInput, now time.Time) (CostSummaryRequest, error) {
	months := input.LookbackMonths
	if months == 0 {
		months = defaultLookbackMonths
	}
	if months < 1 || months > maxLookbackMonths {
		return CostSummaryRequest{}, errors.New("lookback_months must be between 1 and 12")
	}
	serviceLimit := input.MaxServices
	if serviceLimit == 0 {
		serviceLimit = defaultMaxServices
	}
	if serviceLimit < 1 || serviceLimit > maxServices {
		return CostSummaryRequest{}, errors.New("max_services must be between 1 and 20")
	}

	now = now.UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return CostSummaryRequest{
		Start:       monthStart.AddDate(0, -(months - 1), 0),
		End:         time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		MonthStart:  monthStart,
		MaxServices: serviceLimit,
	}, nil
}

func sanitizeLogMessage(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= maxLogMessageRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxLogMessageRunes]) + "…"
}
