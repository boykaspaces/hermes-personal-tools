package config

import "testing"

func TestFromEnvironmentDefaults(t *testing.T) {
	config, err := FromEnvironment(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if config.Calendar.Mode != ProviderMock || config.Gmail.Mode != ProviderDisabled || config.AWSOps.Mode != ProviderDisabled {
		t.Fatalf("unexpected defaults: %#v", config)
	}
}

func TestFromEnvironmentAWSOperations(t *testing.T) {
	values := map[string]string{
		"AWSOPS_PROVIDER_MODE":            "aws",
		"AWSOPS_STACK_NAME":               "personal-tools",
		"AWSOPS_MCP_FUNCTION_NAME":        "personal-tools-mcp",
		"AWSOPS_AUTHORIZER_FUNCTION_NAME": "personal-tools-authorizer",
		"AWSOPS_HERMES_MANAGED_NODE_ID":   "i-0123456789abcdef0",
		"AWSOPS_LOG_GROUP_NAMES":          " /aws/lambda/mcp, /aws/lambda/authorizer, /aws/lambda/mcp ",
		"AWSOPS_ACCOUNT_ID":               "123456789012",
		"AWSOPS_BUDGET_NAMES":             " personal-hermes-bedrock-monthly, personal-aws-account-monthly ",
	}
	config, err := FromEnvironment(func(name string) string { return values[name] })
	if err != nil {
		t.Fatal(err)
	}
	if !config.UsesAWS() || !config.UsesLiveProviders() {
		t.Fatalf("AWS provider was not activated: %#v", config)
	}
	if len(config.AWSOperations.LogGroupNames) != 2 || len(config.AWSOperations.BudgetNames) != 2 {
		t.Fatalf("AWS target lists were not normalized: %#v", config.AWSOperations)
	}
}

func TestFromEnvironmentIndependentModes(t *testing.T) {
	values := map[string]string{
		"CALENDAR_PROVIDER_MODE":       "disabled",
		"GMAIL_PROVIDER_MODE":          "google",
		"GOOGLE_CREDENTIAL_SECRET_ARN": "arn:example",
	}
	config, err := FromEnvironment(func(name string) string { return values[name] })
	if err != nil {
		t.Fatal(err)
	}
	if config.Calendar.Mode != ProviderDisabled || config.Gmail.Mode != ProviderGoogle || !config.UsesGoogle() {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestFromEnvironmentRejectsInvalidConfiguration(t *testing.T) {
	tests := []map[string]string{
		{"CALENDAR_PROVIDER_MODE": "github"},
		{"GMAIL_PROVIDER_MODE": "google"},
		{"AWSOPS_PROVIDER_MODE": "aws"},
		{
			"AWSOPS_PROVIDER_MODE":            "aws",
			"AWSOPS_STACK_NAME":               "personal-tools",
			"AWSOPS_MCP_FUNCTION_NAME":        "mcp",
			"AWSOPS_AUTHORIZER_FUNCTION_NAME": "authorizer",
			"AWSOPS_HERMES_MANAGED_NODE_ID":   "not-an-instance",
			"AWSOPS_LOG_GROUP_NAMES":          "/aws/lambda/mcp",
			"AWSOPS_ACCOUNT_ID":               "123456789012",
			"AWSOPS_BUDGET_NAMES":             "personal-aws-account-monthly",
		},
	}
	for _, values := range tests {
		if _, err := FromEnvironment(func(name string) string { return values[name] }); err == nil {
			t.Fatalf("expected error for %#v", values)
		}
	}
}
