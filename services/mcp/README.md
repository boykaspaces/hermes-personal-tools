# MCP Service

This service owns the MCP protocol, trusted caller propagation, tool input
validation, provider execution and redacted auditing.

It never validates the original Hermes client token. API Gateway removes that
header before invoking this Lambda and supplies the trusted authorizer context.

Calendar, Gmail and AWS operations have independent provider modes.
Disabled features are not registered and are invisible to MCP clients. Enabled
tools are intentionally limited to:

- `calendar_list_calendars`: discovers calendars with event-content read access.
- `calendar_list_events`: reads `primary` or an exact calendar ID returned by
  `calendar_list_calendars`; with `all_readable_calendars=true`, it queries all
  readable calendars, merges and sorts events, and applies `max_results`
  globally. Event output includes bounded `description`, `location`,
  `event_type` and `birthday_properties` in addition to the core schedule
  fields. Google access is rechecked before every per-calendar query.
- `gmail_create_draft`: creates an unsent draft for human review.
- `awsops_get_personal_tools_status`: describes the configured CloudFormation
  Stack and the MCP/Authorizer Lambda runtime status.
- `awsops_get_hermes_status`: checks the configured Hermes managed node's SSM
  connection state.
- `awsops_get_recent_errors`: reads at most 20 bounded messages from the three
  configured operational log groups over a maximum 24-hour window. Messages
  are marked as untrusted operational data.
- `awsops_get_cost_summary`: reads account-wide monthly unblended cost for up
  to 12 calendar months and returns a bounded current-month service ranking.
- `awsops_get_budgets`: reads limits, actual spend, forecast spend,
  utilization, and legacy cost filters or current filter expressions for the
  deployment-configured Hermes AI and account budgets.

There is no email-send tool. Authentication confirms that the caller is the
trusted Hermes client; the MCP service does not maintain per-client scopes.

Feature definitions live under `internal/features/<feature>`, while
`internal/catalog` is the single production allowlist. The generic resolver
handles all definitions without knowing specific tools or providers. Protocol
handling, tool definitions and provider adapters live in separate packages so
adding a future integration does not expand the MCP server core.

AWS operations resource targets come only from deployment environment variables.
No AWS operations tool accepts an account ID, budget name, ARN, Stack name,
function name, log group or instance ID, and there is no arbitrary AWS CLI,
SSM command or shell tool. Cost Explorer and Budgets use the global billing
endpoint in `us-east-1`; cost output is explicitly marked as delayed billing data.
