package awsops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	budgetsservice "github.com/aws/aws-sdk-go-v2/service/budgets"
	budgetstypes "github.com/aws/aws-sdk-go-v2/service/budgets/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	costtypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	lambdaservice "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const (
	operationalErrorFilter = `?ERROR ?Error ?error ?Exception ?"Task timed out"`
	billingAPIRegion       = "us-east-1"
	costMetric             = "UnblendedCost"
	maxCostPages           = 10
)

type cloudFormationClient interface {
	DescribeStacks(context.Context, *cloudformation.DescribeStacksInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

type lambdaClient interface {
	GetFunctionConfiguration(context.Context, *lambdaservice.GetFunctionConfigurationInput, ...func(*lambdaservice.Options)) (*lambdaservice.GetFunctionConfigurationOutput, error)
}

type logsClient interface {
	FilterLogEvents(context.Context, *cloudwatchlogs.FilterLogEventsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error)
}

type ssmClient interface {
	GetConnectionStatus(context.Context, *ssm.GetConnectionStatusInput, ...func(*ssm.Options)) (*ssm.GetConnectionStatusOutput, error)
}

type costExplorerClient interface {
	GetCostAndUsage(context.Context, *costexplorer.GetCostAndUsageInput, ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}

type budgetsClient interface {
	DescribeBudget(context.Context, *budgetsservice.DescribeBudgetInput, ...func(*budgetsservice.Options)) (*budgetsservice.DescribeBudgetOutput, error)
}

type AWSProvider struct {
	targets        Targets
	cloudFormation cloudFormationClient
	lambda         lambdaClient
	logs           logsClient
	ssm            ssmClient
	costExplorer   costExplorerClient
	budgets        budgetsClient
}

func NewAWSProvider(ctx context.Context, targets Targets) (*AWSProvider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS SDK configuration: %w", err)
	}
	billingCfg := cfg.Copy()
	billingCfg.Region = billingAPIRegion
	return newAWSProvider(
		targets,
		cloudformation.NewFromConfig(cfg),
		lambdaservice.NewFromConfig(cfg),
		cloudwatchlogs.NewFromConfig(cfg),
		ssm.NewFromConfig(cfg),
		costexplorer.NewFromConfig(billingCfg),
		budgetsservice.NewFromConfig(billingCfg),
	), nil
}

func newAWSProvider(targets Targets, cloudFormation cloudFormationClient, lambda lambdaClient, logs logsClient, ssmClient ssmClient, costExplorer costExplorerClient, budgets budgetsClient) *AWSProvider {
	return &AWSProvider{
		targets:        targets,
		cloudFormation: cloudFormation,
		lambda:         lambda,
		logs:           logs,
		ssm:            ssmClient,
		costExplorer:   costExplorer,
		budgets:        budgets,
	}
}

func (*AWSProvider) Name() string { return "aws" }

func (p *AWSProvider) LogGroupCount() int { return len(p.targets.LogGroupNames) }

func (p *AWSProvider) PersonalToolsStatus(ctx context.Context) (PersonalToolsStatus, error) {
	stackOutput, err := p.cloudFormation.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(p.targets.StackName),
	})
	if err != nil {
		return PersonalToolsStatus{}, fmt.Errorf("describe configured stack: %w", err)
	}
	if len(stackOutput.Stacks) != 1 {
		return PersonalToolsStatus{}, fmt.Errorf("describe configured stack returned %d stacks", len(stackOutput.Stacks))
	}
	stack := stackOutput.Stacks[0]

	functionNames := []string{p.targets.MCPFunctionName, p.targets.AuthorizerFunctionName}
	functions := make([]FunctionStatus, 0, len(functionNames))
	for _, functionName := range functionNames {
		output, err := p.lambda.GetFunctionConfiguration(ctx, &lambdaservice.GetFunctionConfigurationInput{
			FunctionName: aws.String(functionName),
		})
		if err != nil {
			return PersonalToolsStatus{}, fmt.Errorf("get configured Lambda %q: %w", functionName, err)
		}
		functions = append(functions, FunctionStatus{
			Name:             aws.ToString(output.FunctionName),
			State:            string(output.State),
			LastUpdateStatus: string(output.LastUpdateStatus),
			LastModified:     aws.ToString(output.LastModified),
			MemoryMB:         aws.ToInt32(output.MemorySize),
			TimeoutSeconds:   aws.ToInt32(output.Timeout),
		})
	}

	return PersonalToolsStatus{
		StackName:             aws.ToString(stack.StackName),
		StackStatus:           string(stack.StackStatus),
		StackCreatedAt:        formatTime(stack.CreationTime),
		StackLastUpdatedAt:    formatTime(stack.LastUpdatedTime),
		TerminationProtection: aws.ToBool(stack.EnableTerminationProtection),
		Functions:             functions,
	}, nil
}

func (p *AWSProvider) HermesStatus(ctx context.Context) (HermesStatus, error) {
	output, err := p.ssm.GetConnectionStatus(ctx, &ssm.GetConnectionStatusInput{
		Target: aws.String(p.targets.HermesManagedNodeID),
	})
	if err != nil {
		return HermesStatus{}, fmt.Errorf("get configured Hermes managed-node connection: %w", err)
	}
	return HermesStatus{
		ManagedNodeID:   aws.ToString(output.Target),
		ConnectionState: string(output.Status),
	}, nil
}

func (p *AWSProvider) RecentErrors(ctx context.Context, request RecentErrorsRequest) ([]LogError, error) {
	if request.MaxResults < 1 {
		return nil, errors.New("max results must be positive")
	}
	type timestampedError struct {
		value     LogError
		timestamp time.Time
	}
	all := make([]timestampedError, 0, request.MaxResults*len(p.targets.LogGroupNames))
	startMilliseconds := request.Since.UnixMilli()
	limit := int32(request.MaxResults)
	startFromHead := false
	for _, logGroupName := range p.targets.LogGroupNames {
		output, err := p.logs.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
			FilterPattern: aws.String(operationalErrorFilter),
			Limit:         &limit,
			LogGroupName:  aws.String(logGroupName),
			StartFromHead: &startFromHead,
			StartTime:     &startMilliseconds,
		})
		if err != nil {
			return nil, fmt.Errorf("filter configured log group %q: %w", logGroupName, err)
		}
		for _, event := range output.Events {
			if event.Timestamp == nil {
				continue
			}
			eventTime := time.UnixMilli(*event.Timestamp).UTC()
			all = append(all, timestampedError{
				timestamp: eventTime,
				value: LogError{
					LogGroup:  logGroupName,
					Timestamp: eventTime.Format(time.RFC3339Nano),
					Message:   sanitizeLogMessage(aws.ToString(event.Message)),
				},
			})
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].timestamp.After(all[j].timestamp)
	})
	if len(all) > request.MaxResults {
		all = all[:request.MaxResults]
	}
	errorsFound := make([]LogError, len(all))
	for index := range all {
		errorsFound[index] = all[index].value
	}
	return errorsFound, nil
}

func (p *AWSProvider) CostSummary(ctx context.Context, request CostSummaryRequest) (CostSummary, error) {
	if request.MaxServices < 1 {
		return CostSummary{}, errors.New("max services must be positive")
	}
	if request.End.Before(request.Start) {
		return CostSummary{}, errors.New("cost query end must not precede start")
	}

	summary := CostSummary{
		Metric:                  costMetric,
		Granularity:             string(costtypes.GranularityMonthly),
		Start:                   request.Start.Format(time.DateOnly),
		EndExclusive:            request.End.Format(time.DateOnly),
		MonthlyCosts:            []CostPeriod{},
		CurrentMonthTopServices: []ServiceCost{},
		BillingDataDelayed:      true,
	}
	if !request.Start.Before(request.End) {
		return summary, nil
	}

	totals, err := p.costExplorer.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		Granularity: costtypes.GranularityMonthly,
		Metrics:     []string{costMetric},
		TimePeriod: &costtypes.DateInterval{
			Start: aws.String(request.Start.Format(time.DateOnly)),
			End:   aws.String(request.End.Format(time.DateOnly)),
		},
	})
	if err != nil {
		return CostSummary{}, fmt.Errorf("get monthly account cost: %w", err)
	}
	for _, result := range totals.ResultsByTime {
		metric, ok := result.Total[costMetric]
		if !ok || result.TimePeriod == nil {
			continue
		}
		summary.MonthlyCosts = append(summary.MonthlyCosts, CostPeriod{
			Start:     aws.ToString(result.TimePeriod.Start),
			End:       aws.ToString(result.TimePeriod.End),
			Amount:    aws.ToString(metric.Amount),
			Unit:      aws.ToString(metric.Unit),
			Estimated: result.Estimated,
		})
	}

	if !request.MonthStart.Before(request.End) {
		return summary, nil
	}
	services, err := p.currentMonthServices(ctx, request.MonthStart, request.End)
	if err != nil {
		return CostSummary{}, err
	}
	summary.CurrentMonthServiceCount = len(services)
	if len(services) > request.MaxServices {
		summary.ServicesTruncated = true
		services = services[:request.MaxServices]
	}
	summary.CurrentMonthTopServices = services
	return summary, nil
}

func (p *AWSProvider) currentMonthServices(ctx context.Context, start time.Time, end time.Time) ([]ServiceCost, error) {
	services := make([]ServiceCost, 0)
	var nextPageToken *string
	for page := 0; page < maxCostPages; page++ {
		output, err := p.costExplorer.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			Granularity: costtypes.GranularityMonthly,
			Metrics:     []string{costMetric},
			TimePeriod: &costtypes.DateInterval{
				Start: aws.String(start.Format(time.DateOnly)),
				End:   aws.String(end.Format(time.DateOnly)),
			},
			GroupBy: []costtypes.GroupDefinition{{
				Type: costtypes.GroupDefinitionTypeDimension,
				Key:  aws.String("SERVICE"),
			}},
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("get current-month service costs: %w", err)
		}
		for _, period := range output.ResultsByTime {
			for _, group := range period.Groups {
				metric, ok := group.Metrics[costMetric]
				if !ok || len(group.Keys) != 1 {
					continue
				}
				services = append(services, ServiceCost{
					Service: group.Keys[0],
					Amount:  aws.ToString(metric.Amount),
					Unit:    aws.ToString(metric.Unit),
				})
			}
		}
		nextPageToken = output.NextPageToken
		if nextPageToken == nil || aws.ToString(nextPageToken) == "" {
			sortServiceCosts(services)
			return services, nil
		}
	}
	return nil, fmt.Errorf("current-month service cost query exceeded %d pages", maxCostPages)
}

func (p *AWSProvider) Budgets(ctx context.Context) ([]BudgetSummary, error) {
	result := make([]BudgetSummary, 0, len(p.targets.BudgetNames))
	for _, name := range p.targets.BudgetNames {
		output, err := p.budgets.DescribeBudget(ctx, &budgetsservice.DescribeBudgetInput{
			AccountId:  aws.String(p.targets.AccountID),
			BudgetName: aws.String(name),
		})
		if err != nil {
			return nil, fmt.Errorf("describe configured budget %q: %w", name, err)
		}
		if output.Budget == nil {
			return nil, fmt.Errorf("describe configured budget %q returned no budget", name)
		}
		result = append(result, mapBudget(*output.Budget))
	}
	return result, nil
}

func mapBudget(value budgetstypes.Budget) BudgetSummary {
	result := BudgetSummary{
		Name:             aws.ToString(value.BudgetName),
		BudgetType:       string(value.BudgetType),
		TimeUnit:         string(value.TimeUnit),
		CostFilters:      value.CostFilters, //nolint:staticcheck
		FilterExpression: formatBudgetFilterExpression(value.FilterExpression),
		LastUpdatedAt:    formatTime(value.LastUpdatedTime),
	}
	if value.BudgetLimit != nil {
		result.Limit = mapSpend(value.BudgetLimit)
	}
	if value.CalculatedSpend != nil {
		result.ActualSpend = mapSpend(value.CalculatedSpend.ActualSpend)
		result.ActualPercent = percentage(result.ActualSpend.Amount, result.Limit.Amount)
		if value.CalculatedSpend.ForecastedSpend != nil {
			forecast := mapSpend(value.CalculatedSpend.ForecastedSpend)
			result.ForecastedSpend = &forecast
			result.ForecastedPercent = percentage(forecast.Amount, result.Limit.Amount)
		}
	}
	return result
}

func formatBudgetFilterExpression(value *budgetstypes.Expression) string {
	if value == nil {
		return ""
	}
	parts := make([]string, 0)
	for _, child := range value.And {
		if formatted := formatBudgetFilterExpression(&child); formatted != "" {
			parts = append(parts, "("+formatted+")")
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " AND ")
	}
	for _, child := range value.Or {
		if formatted := formatBudgetFilterExpression(&child); formatted != "" {
			parts = append(parts, "("+formatted+")")
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " OR ")
	}
	if value.Not != nil {
		return "NOT (" + formatBudgetFilterExpression(value.Not) + ")"
	}
	if value.Dimensions != nil {
		return formatBudgetFilterValues(string(value.Dimensions.Key), value.Dimensions.Values, value.Dimensions.MatchOptions)
	}
	if value.Tags != nil {
		return formatBudgetFilterValues("TAG:"+aws.ToString(value.Tags.Key), value.Tags.Values, value.Tags.MatchOptions)
	}
	if value.CostCategories != nil {
		return formatBudgetFilterValues("COST_CATEGORY:"+aws.ToString(value.CostCategories.Key), value.CostCategories.Values, value.CostCategories.MatchOptions)
	}
	return ""
}

func formatBudgetFilterValues(key string, values []string, matchOptions []budgetstypes.MatchOption) string {
	options := make([]string, len(matchOptions))
	for index, value := range matchOptions {
		options[index] = string(value)
	}
	if len(options) == 0 {
		options = []string{"EQUALS"}
	}
	encodedValues, _ := json.Marshal(values)
	return key + " " + strings.Join(options, "+") + " " + string(encodedValues)
}

func mapSpend(value *budgetstypes.Spend) Spend {
	if value == nil {
		return Spend{}
	}
	return Spend{Amount: aws.ToString(value.Amount), Unit: aws.ToString(value.Unit)}
}

func percentage(amount string, limit string) string {
	amountValue, ok := new(big.Rat).SetString(amount)
	if !ok {
		return ""
	}
	limitValue, ok := new(big.Rat).SetString(limit)
	if !ok || limitValue.Sign() == 0 {
		return ""
	}
	numerator := new(big.Rat).Mul(amountValue, big.NewRat(100, 1))
	return new(big.Rat).Quo(numerator, limitValue).FloatString(2)
}

func sortServiceCosts(costs []ServiceCost) {
	sort.SliceStable(costs, func(i, j int) bool {
		left, leftOK := new(big.Rat).SetString(costs[i].Amount)
		right, rightOK := new(big.Rat).SetString(costs[j].Amount)
		if leftOK && rightOK {
			comparison := left.Cmp(right)
			if comparison != 0 {
				return comparison > 0
			}
		}
		return costs[i].Service < costs[j].Service
	})
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
