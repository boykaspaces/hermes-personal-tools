package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type ProviderMode string

const (
	ProviderDisabled ProviderMode = "disabled"
	ProviderMock     ProviderMode = "mock"
	ProviderGoogle   ProviderMode = "google"
	ProviderAWS      ProviderMode = "aws"
)

type Config struct {
	Calendar        Feature
	Gmail           Feature
	AWSOps          Feature
	GoogleWorkspace GoogleWorkspace
	AWSOperations   AWSOperations
}

type Feature struct {
	Mode ProviderMode
}

type GoogleWorkspace struct {
	CredentialSecretARN string
}

// AWSOperations contains deployment-controlled targets. MCP callers never
// supply resource names, IDs or ARNs, which keeps every AWS read within the
// Personal Tools and Hermes resources selected by CloudFormation.
type AWSOperations struct {
	StackName              string
	MCPFunctionName        string
	AuthorizerFunctionName string
	HermesManagedNodeID    string
	LogGroupNames          []string
	AccountID              string
	BudgetNames            []string
}

type LookupEnv func(string) string

func FromEnvironment(lookup LookupEnv) (Config, error) {
	calendarMode, err := providerMode(lookup("CALENDAR_PROVIDER_MODE"), ProviderMock, "CALENDAR_PROVIDER_MODE", ProviderGoogle)
	if err != nil {
		return Config{}, err
	}
	gmailMode, err := providerMode(lookup("GMAIL_PROVIDER_MODE"), ProviderDisabled, "GMAIL_PROVIDER_MODE", ProviderGoogle)
	if err != nil {
		return Config{}, err
	}
	awsOpsMode, err := providerMode(lookup("AWSOPS_PROVIDER_MODE"), ProviderDisabled, "AWSOPS_PROVIDER_MODE", ProviderAWS)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Calendar: Feature{Mode: calendarMode},
		Gmail:    Feature{Mode: gmailMode},
		AWSOps:   Feature{Mode: awsOpsMode},
		GoogleWorkspace: GoogleWorkspace{
			CredentialSecretARN: strings.TrimSpace(lookup("GOOGLE_CREDENTIAL_SECRET_ARN")),
		},
		AWSOperations: AWSOperations{
			StackName:              strings.TrimSpace(lookup("AWSOPS_STACK_NAME")),
			MCPFunctionName:        strings.TrimSpace(lookup("AWSOPS_MCP_FUNCTION_NAME")),
			AuthorizerFunctionName: strings.TrimSpace(lookup("AWSOPS_AUTHORIZER_FUNCTION_NAME")),
			HermesManagedNodeID:    strings.TrimSpace(lookup("AWSOPS_HERMES_MANAGED_NODE_ID")),
			LogGroupNames:          commaSeparatedValues(lookup("AWSOPS_LOG_GROUP_NAMES")),
			AccountID:              strings.TrimSpace(lookup("AWSOPS_ACCOUNT_ID")),
			BudgetNames:            commaSeparatedValues(lookup("AWSOPS_BUDGET_NAMES")),
		},
	}
	if config.UsesGoogle() && config.GoogleWorkspace.CredentialSecretARN == "" {
		return Config{}, errors.New("GOOGLE_CREDENTIAL_SECRET_ARN is required when a provider uses google mode")
	}
	if config.UsesAWS() {
		if err := validateAWSOperations(config.AWSOperations); err != nil {
			return Config{}, err
		}
	}
	return config, nil
}

func (c Config) UsesGoogle() bool {
	return c.Calendar.Mode == ProviderGoogle || c.Gmail.Mode == ProviderGoogle
}

func (c Config) UsesAWS() bool {
	return c.AWSOps.Mode == ProviderAWS
}

func (c Config) UsesLiveProviders() bool {
	return c.UsesGoogle() || c.UsesAWS()
}

var managedNodeIDPattern = regexp.MustCompile(`^(i-[0-9a-f]{8,17}|mi-[0-9a-f]{17})$`)
var awsAccountIDPattern = regexp.MustCompile(`^[0-9]{12}$`)

func validateAWSOperations(targets AWSOperations) error {
	required := map[string]string{
		"AWSOPS_STACK_NAME":               targets.StackName,
		"AWSOPS_MCP_FUNCTION_NAME":        targets.MCPFunctionName,
		"AWSOPS_AUTHORIZER_FUNCTION_NAME": targets.AuthorizerFunctionName,
		"AWSOPS_HERMES_MANAGED_NODE_ID":   targets.HermesManagedNodeID,
		"AWSOPS_ACCOUNT_ID":               targets.AccountID,
	}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required when AWSOPS_PROVIDER_MODE=aws", name)
		}
	}
	if !managedNodeIDPattern.MatchString(targets.HermesManagedNodeID) {
		return errors.New("AWSOPS_HERMES_MANAGED_NODE_ID must be an EC2 or SSM managed-node ID")
	}
	if len(targets.LogGroupNames) == 0 {
		return errors.New("AWSOPS_LOG_GROUP_NAMES is required when AWSOPS_PROVIDER_MODE=aws")
	}
	if len(targets.LogGroupNames) > 10 {
		return errors.New("AWSOPS_LOG_GROUP_NAMES cannot contain more than 10 log groups")
	}
	if !awsAccountIDPattern.MatchString(targets.AccountID) {
		return errors.New("AWSOPS_ACCOUNT_ID must be a 12-digit AWS account ID")
	}
	if len(targets.BudgetNames) == 0 {
		return errors.New("AWSOPS_BUDGET_NAMES is required when AWSOPS_PROVIDER_MODE=aws")
	}
	if len(targets.BudgetNames) > 10 {
		return errors.New("AWSOPS_BUDGET_NAMES cannot contain more than 10 budget names")
	}
	for _, name := range targets.BudgetNames {
		if len(name) > 100 || strings.ContainsAny(name, `:\`) || strings.Contains(name, "/action/") {
			return fmt.Errorf("AWSOPS_BUDGET_NAMES contains invalid budget name %q", name)
		}
	}
	return nil
}

func commaSeparatedValues(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	values := make([]string, 0, strings.Count(value, ",")+1)
	seen := make(map[string]struct{})
	for _, raw := range strings.Split(value, ",") {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if _, duplicate := seen[item]; duplicate {
			continue
		}
		seen[item] = struct{}{}
		values = append(values, item)
	}
	return values
}

// providerMode validates one feature's supported provider set. Disabled and
// mock are common lifecycle modes; real providers are supplied per feature so
// adding a future GitHub provider does not widen Calendar or Gmail's values.
func providerMode(value string, defaultMode ProviderMode, environmentName string, realProviders ...ProviderMode) (ProviderMode, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultMode, nil
	}
	mode := ProviderMode(strings.ToLower(value))
	switch mode {
	case ProviderDisabled, ProviderMock:
		return mode, nil
	}
	for _, provider := range realProviders {
		if mode == provider {
			return mode, nil
		}
	}
	return "", fmt.Errorf("%s has unsupported provider mode %q", environmentName, value)
}
