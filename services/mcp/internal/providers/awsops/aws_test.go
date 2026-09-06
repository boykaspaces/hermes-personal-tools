package awsops

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	budgetsservice "github.com/aws/aws-sdk-go-v2/service/budgets"
	budgetstypes "github.com/aws/aws-sdk-go-v2/service/budgets/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	costtypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	lambdaservice "github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type fakeCloudFormation struct {
	stackName string
}

func (f *fakeCloudFormation) DescribeStacks(_ context.Context, input *cloudformation.DescribeStacksInput, _ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	f.stackName = aws.ToString(input.StackName)
	created := time.Date(2026, 8, 13, 1, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	return &cloudformation.DescribeStacksOutput{Stacks: []cloudformationtypes.Stack{{
		StackName:                   aws.String(f.stackName),
		StackStatus:                 cloudformationtypes.StackStatusUpdateComplete,
		CreationTime:                &created,
		LastUpdatedTime:             &updated,
		EnableTerminationProtection: aws.Bool(true),
	}}}, nil
}

type fakeLambda struct {
	functionNames []string
}

func (f *fakeLambda) GetFunctionConfiguration(_ context.Context, input *lambdaservice.GetFunctionConfigurationInput, _ ...func(*lambdaservice.Options)) (*lambdaservice.GetFunctionConfigurationOutput, error) {
	name := aws.ToString(input.FunctionName)
	f.functionNames = append(f.functionNames, name)
	return &lambdaservice.GetFunctionConfigurationOutput{
		FunctionName:     aws.String(name),
		State:            lambdatypes.StateActive,
		LastUpdateStatus: lambdatypes.LastUpdateStatusSuccessful,
		LastModified:     aws.String("2026-08-14T01:00:00Z"),
		MemorySize:       aws.Int32(256),
		Timeout:          aws.Int32(20),
	}, nil
}

type fakeSSM struct {
	target string
}

func (f *fakeSSM) GetConnectionStatus(_ context.Context, input *ssm.GetConnectionStatusInput, _ ...func(*ssm.Options)) (*ssm.GetConnectionStatusOutput, error) {
	f.target = aws.ToString(input.Target)
	return &ssm.GetConnectionStatusOutput{Target: input.Target, Status: ssmtypes.ConnectionStatusConnected}, nil
}

type fakeLogs struct {
	groups []string
}

type fakeCostExplorer struct {
	inputs []*costexplorer.GetCostAndUsageInput
}

func (f *fakeCostExplorer) GetCostAndUsage(_ context.Context, input *costexplorer.GetCostAndUsageInput, _ ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	f.inputs = append(f.inputs, input)
	if len(input.GroupBy) == 0 {
		return &costexplorer.GetCostAndUsageOutput{ResultsByTime: []costtypes.ResultByTime{{
			TimePeriod: input.TimePeriod,
			Estimated:  true,
			Total: map[string]costtypes.MetricValue{
				costMetric: {Amount: aws.String("12.50"), Unit: aws.String("USD")},
			},
		}}}, nil
	}
	return &costexplorer.GetCostAndUsageOutput{ResultsByTime: []costtypes.ResultByTime{{
		Groups: []costtypes.Group{
			{Keys: []string{"Amazon EC2"}, Metrics: map[string]costtypes.MetricValue{costMetric: {Amount: aws.String("2.50"), Unit: aws.String("USD")}}},
			{Keys: []string{"Amazon Bedrock"}, Metrics: map[string]costtypes.MetricValue{costMetric: {Amount: aws.String("10.00"), Unit: aws.String("USD")}}},
		},
	}}}, nil
}

type fakeBudgets struct {
	accountIDs []string
	names      []string
}

func (f *fakeBudgets) DescribeBudget(_ context.Context, input *budgetsservice.DescribeBudgetInput, _ ...func(*budgetsservice.Options)) (*budgetsservice.DescribeBudgetOutput, error) {
	f.accountIDs = append(f.accountIDs, aws.ToString(input.AccountId))
	f.names = append(f.names, aws.ToString(input.BudgetName))
	updated := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	return &budgetsservice.DescribeBudgetOutput{Budget: &budgetstypes.Budget{
		BudgetName: aws.String(aws.ToString(input.BudgetName)),
		BudgetType: budgetstypes.BudgetTypeCost,
		TimeUnit:   budgetstypes.TimeUnitMonthly,
		BudgetLimit: &budgetstypes.Spend{
			Amount: aws.String("20"),
			Unit:   aws.String("USD"),
		},
		CalculatedSpend: &budgetstypes.CalculatedSpend{
			ActualSpend:     &budgetstypes.Spend{Amount: aws.String("5"), Unit: aws.String("USD")},
			ForecastedSpend: &budgetstypes.Spend{Amount: aws.String("10"), Unit: aws.String("USD")},
		},
		FilterExpression: &budgetstypes.Expression{Or: []budgetstypes.Expression{
			{Dimensions: &budgetstypes.ExpressionDimensionValues{
				Key:          budgetstypes.DimensionService,
				Values:       []string{"Amazon Bedrock"},
				MatchOptions: []budgetstypes.MatchOption{budgetstypes.MatchOptionEquals},
			}},
			{Dimensions: &budgetstypes.ExpressionDimensionValues{
				Key:          budgetstypes.DimensionBillingEntity,
				Values:       []string{"AWS Marketplace"},
				MatchOptions: []budgetstypes.MatchOption{budgetstypes.MatchOptionEquals},
			}},
		}},
		LastUpdatedTime: &updated,
	}}, nil
}

func (f *fakeLogs) FilterLogEvents(_ context.Context, input *cloudwatchlogs.FilterLogEventsInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error) {
	group := aws.ToString(input.LogGroupName)
	f.groups = append(f.groups, group)
	timestamp := *input.StartTime + int64(len(f.groups))*1000
	return &cloudwatchlogs.FilterLogEventsOutput{Events: []cloudwatchlogstypes.FilteredLogEvent{{
		Timestamp: &timestamp,
		Message:   aws.String(" error\x00 from " + group),
	}}}, nil
}

func TestAWSProviderUsesOnlyConfiguredTargets(t *testing.T) {
	cloudFormation := &fakeCloudFormation{}
	lambda := &fakeLambda{}
	logs := &fakeLogs{}
	ssmClient := &fakeSSM{}
	costExplorer := &fakeCostExplorer{}
	budgetsClient := &fakeBudgets{}
	provider := newAWSProvider(Targets{
		StackName:              "personal-tools",
		MCPFunctionName:        "personal-tools-mcp",
		AuthorizerFunctionName: "personal-tools-authorizer",
		HermesManagedNodeID:    "i-0123456789abcdef0",
		LogGroupNames:          []string{"/aws/lambda/mcp", "/aws/lambda/authorizer"},
		AccountID:              "123456789012",
		BudgetNames:            []string{"personal-hermes-bedrock-monthly", "personal-aws-account-monthly"},
	}, cloudFormation, lambda, logs, ssmClient, costExplorer, budgetsClient)

	status, err := provider.PersonalToolsStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cloudFormation.stackName != "personal-tools" || len(lambda.functionNames) != 2 || status.StackStatus != "UPDATE_COMPLETE" {
		t.Fatalf("unexpected status query: stack=%q functions=%#v status=%#v", cloudFormation.stackName, lambda.functionNames, status)
	}

	hermes, err := provider.HermesStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ssmClient.target != "i-0123456789abcdef0" || hermes.ConnectionState != "connected" {
		t.Fatalf("unexpected Hermes query: target=%q status=%#v", ssmClient.target, hermes)
	}

	errorsFound, err := provider.RecentErrors(context.Background(), RecentErrorsRequest{
		Since:      time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC),
		MaxResults: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.groups) != 2 || len(errorsFound) != 1 || errorsFound[0].LogGroup != "/aws/lambda/authorizer" {
		t.Fatalf("unexpected log query: groups=%#v errors=%#v", logs.groups, errorsFound)
	}
	if errorsFound[0].Message != "error from /aws/lambda/authorizer" {
		t.Fatalf("log message was not sanitized: %q", errorsFound[0].Message)
	}

	costs, err := provider.CostSummary(context.Background(), CostSummaryRequest{
		Start:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
		MonthStart:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		MaxServices: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(costExplorer.inputs) != 2 || len(costs.MonthlyCosts) != 1 || len(costs.CurrentMonthTopServices) != 1 || costs.CurrentMonthTopServices[0].Service != "Amazon Bedrock" || !costs.ServicesTruncated {
		t.Fatalf("unexpected cost query: inputs=%d costs=%#v", len(costExplorer.inputs), costs)
	}
	if len(costExplorer.inputs[1].GroupBy) != 1 || aws.ToString(costExplorer.inputs[1].GroupBy[0].Key) != "SERVICE" {
		t.Fatalf("service query was not fixed to SERVICE: %#v", costExplorer.inputs[1])
	}

	configuredBudgets, err := provider.Budgets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(configuredBudgets) != 2 || len(budgetsClient.names) != 2 || budgetsClient.accountIDs[0] != "123456789012" || configuredBudgets[0].ActualPercent != "25.00" || configuredBudgets[0].ForecastedPercent != "50.00" {
		t.Fatalf("unexpected budget query: requests=%#v budgets=%#v", budgetsClient.names, configuredBudgets)
	}
	filter := configuredBudgets[0].FilterExpression
	if filter != `(SERVICE EQUALS ["Amazon Bedrock"]) OR (BILLING_ENTITY EQUALS ["AWS Marketplace"])` {
		t.Fatalf("unexpected budget filter expression: %q", filter)
	}
}
