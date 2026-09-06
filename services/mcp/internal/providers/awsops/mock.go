package awsops

import (
	"context"
	"time"
)

type MockProvider struct{}

func (MockProvider) Name() string { return "mock" }

func (MockProvider) LogGroupCount() int { return 3 }

func (MockProvider) PersonalToolsStatus(context.Context) (PersonalToolsStatus, error) {
	return PersonalToolsStatus{
		StackName:             "personal-tools-mock",
		StackStatus:           "UPDATE_COMPLETE",
		StackCreatedAt:        "2026-08-13T00:00:00Z",
		StackLastUpdatedAt:    "2026-08-14T00:00:00Z",
		TerminationProtection: true,
		Functions: []FunctionStatus{
			{Name: "personal-tools-mock-mcp", State: "Active", LastUpdateStatus: "Successful", MemoryMB: 256, TimeoutSeconds: 20},
			{Name: "personal-tools-mock-authorizer", State: "Active", LastUpdateStatus: "Successful", MemoryMB: 128, TimeoutSeconds: 5},
		},
	}, nil
}

func (MockProvider) HermesStatus(context.Context) (HermesStatus, error) {
	return HermesStatus{ManagedNodeID: "i-0123456789abcdef0", ConnectionState: "connected"}, nil
}

func (MockProvider) RecentErrors(_ context.Context, request RecentErrorsRequest) ([]LogError, error) {
	return []LogError{
		{
			LogGroup:  "/aws/lambda/personal-tools-mock-mcp",
			Timestamp: request.Since.Add(5 * time.Minute).Format(time.RFC3339),
			Message:   "mock provider error for protocol validation",
		},
	}, nil
}

func (MockProvider) CostSummary(_ context.Context, request CostSummaryRequest) (CostSummary, error) {
	return CostSummary{
		Metric:       "UnblendedCost",
		Granularity:  "MONTHLY",
		Start:        request.Start.Format(time.DateOnly),
		EndExclusive: request.End.Format(time.DateOnly),
		MonthlyCosts: []CostPeriod{
			{Start: request.MonthStart.Format(time.DateOnly), End: request.End.Format(time.DateOnly), Amount: "12.34", Unit: "USD", Estimated: true},
		},
		CurrentMonthTopServices: []ServiceCost{
			{Service: "Amazon Bedrock", Amount: "8.75", Unit: "USD"},
			{Service: "Amazon Elastic Compute Cloud - Compute", Amount: "3.59", Unit: "USD"},
		},
		CurrentMonthServiceCount: 2,
		ServicesTruncated:        false,
		BillingDataDelayed:       true,
	}, nil
}

func (MockProvider) Budgets(context.Context) ([]BudgetSummary, error) {
	return []BudgetSummary{
		{
			Name:              "personal-hermes-bedrock-monthly",
			BudgetType:        "COST",
			TimeUnit:          "MONTHLY",
			Limit:             Spend{Amount: "20", Unit: "USD"},
			ActualSpend:       Spend{Amount: "8.75", Unit: "USD"},
			ForecastedSpend:   &Spend{Amount: "17.50", Unit: "USD"},
			ActualPercent:     "43.75",
			ForecastedPercent: "87.50",
			CostFilters:       map[string][]string{"Service": {"Amazon Bedrock"}},
		},
		{
			Name:          "personal-aws-account-monthly",
			BudgetType:    "COST",
			TimeUnit:      "MONTHLY",
			Limit:         Spend{Amount: "50", Unit: "USD"},
			ActualSpend:   Spend{Amount: "12.34", Unit: "USD"},
			ActualPercent: "24.68",
		},
	}, nil
}
