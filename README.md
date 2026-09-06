# Personal Tools Services

The Personal Tools deployment contains three independent Go services plus one
host/client module. They have separate responsibilities, dependencies,
binaries, and runtime trust boundaries.

## Repository map

| Path | Status | Owns | Read when |
|---|---|---|---|
| [`services/`](./docs/FILE_MAP.md#services) | Active source | Authorizer, MCP, and Credential Lease Lambda modules | Changing server behavior or providers |
| [`clients/`](./docs/FILE_MAP.md#clients) | Active source | Host provisioner and credential adapters | Changing credential delivery or Git integration |
| [`deploy/aws/`](./deploy/aws/README.md) | Current deployment | Parameterized CloudFormation and operator policy templates | Deploying to AWS |
| [`docs/`](./docs/README.md) | Current index | Architecture, security, GitHub credentials, and file ownership | Understanding or integrating the system |
| [`Makefile`](./Makefile) | Active build entry | Tests, vet, lint, race builds, and release archives | Validating or packaging changes |
| [`PROJECT.md`](./PROJECT.md) | Stable identity | Project goal and source-of-truth boundaries | Starting repository work |

Start from the narrowest component README or file-map entry. Production
identifiers, deployment records, and credentials belong in the consuming
operator repository, never here.

## Layout

```text
services/
├── authorizer/
│   ├── cmd/lambda/              Token Authorizer Lambda entry point
│   ├── internal/auth/           Header parsing and token verification
│   ├── internal/secrets/        Client-token Secret cache
│   ├── go.mod
│   └── go.sum
├── mcp/
│   ├── cmd/lambda/              MCP Lambda entry point
│   ├── cmd/server/              Local MCP development server
│   ├── internal/app/            HTTP route assembly
│   ├── internal/catalog/        Explicit production feature allowlist
│   ├── internal/config/         Typed per-feature provider configuration
│   ├── internal/features/       Generic feature resolver and feature definitions
│   ├── internal/protocol/       MCP SDK and transport-independent server core
│   ├── internal/tools/          Calendar, Gmail and AWS operations MCP modules
│   ├── internal/providers/      Google Workspace and fixed-target AWS adapters
│   ├── internal/identity/       Trusted caller context (no application scopes)
│   ├── internal/secrets/        AWS Secrets Manager string source
│   ├── internal/audit/          Redacted tool-call audit events
│   ├── internal/transport/      API Gateway v2 adapter
│   ├── go.mod
│   └── go.sum
└── credentials/
    ├── cmd/lambda/              Credential Lease Lambda entry point
    ├── internal/app/            Single profile-routed HTTP API and audit
    ├── internal/provider/       Generic issuer contract and profile registry
    ├── internal/providers/
    │   ├── registry.go          Compiled provider allowlist and composition
    │   └── githubapp/           GitHub provider configuration and issuer
    ├── internal/lease/          Provider-neutral lease schema
    ├── internal/secrets/        Purpose-specific Secret reader
    ├── internal/transport/      API Gateway AWS_IAM caller adapter
    ├── go.mod
    └── go.sum

clients/
└── credential-agent/
    ├── cmd/hermes-credential-provisioner/  Host IAM lease client
    ├── cmd/git-credential-hermes/          Git credential adapter
    └── internal/                            Lease, provisioner, and adapter core
```

The Authorizer module does not depend on the MCP SDK or Google client. The MCP
module does not contain the Hermes client-token Secret reader or Authorizer
handler. The Credentials module cannot register MCP tools and only its Lambda
role can read the GitHub App private key. The client module has no provider
private key and runs the provisioner outside the coding container.

## Build and test

```bash
make install-lint  # one-time, installs pinned golangci-lint into bin/
make check
make build
```

`make check` runs unit tests, `go vet`, `golangci-lint` and the Race Detector
for all four Go modules. The lint configuration is `.golangci.yml`, uses
golangci-lint `v2.12.2`, and intentionally starts with the official `standard`
set plus `gofmt`.

Artifacts:

```text
dist/personal-tools-authorizer.zip
dist/personal-tools-mcp.zip
dist/personal-tools-credentials.zip
dist/personal-tools-credential-agent.zip
```

The three Lambda archives contain static Linux arm64 `bootstrap` executables.
The credential-agent archive contains static Linux amd64 host and helper
binaries for the current `t3` EC2 instance.

## Local MCP verification

```bash
cd services/mcp
GATEWAY_BEARER_TOKEN=local-development-token \
CALENDAR_PROVIDER_MODE=mock \
GMAIL_PROVIDER_MODE=mock \
AWSOPS_PROVIDER_MODE=mock \
go run ./cmd/server
```

Each feature independently supports `disabled`, `mock` and `google` in the
Lambda. `disabled` omits the module entirely, so its tools do not appear in
`tools/list`. The local server intentionally supports only `disabled` and
`mock`.

The local server simulates the trusted caller that API Gateway normally
supplies after Authorizer success. Requests use:

```text
X-Hermes-Gateway-Token: Bearer local-development-token
Content-Type: application/json
Accept: application/json, text/event-stream
```

## Production boundary

- API Gateway sends `X-Hermes-Gateway-Token` only to the Authorizer service.
- The Authorizer returns only `caller_id`, never the token or application scopes.
- API Gateway removes the dedicated header before invoking the MCP service.
- The MCP transport also drops the dedicated header and `Authorization`.
- The credential route uses API Gateway `AWS_IAM` and accepts only a
  deployment-defined profile ID. It is not exposed through MCP or its bearer
  Authorizer.
- The GitHub issuer requests an installation token for one numeric repository
  ID and a fixed permission map. Its audit log excludes credentials and HTTP
  response bodies.
- The host provisioner receives renewable authority only through its EC2
  instance role. The coding container receives a read-only, expiring lease and
  cannot call the lease endpoint itself.
- The MCP registry exposes read-only Calendar discovery/event queries and Gmail
  draft creation; it does not expose email sending or destructive operations.
- The AWS operations module exposes fixed-target Stack/Lambda status, Hermes
  SSM connection status, bounded recent-error reads, bounded account cost
  summaries and two deployment-selected budgets. MCP callers cannot supply an
  account ID, budget name, billing filter, Stack name, function name,
  managed-node ID or log group.
- Operational log messages are bounded, stripped of unsafe control characters
  and explicitly marked as untrusted data rather than instructions.
- Calendar reads execute automatically. Gmail output remains an unsent draft
  that a human reviews and sends in Gmail.
- Non-primary Calendar IDs must come from the authenticated account's Google
  CalendarList. The Google adapter rejects `freeBusyReader` and rechecks event-
  content access immediately before every events query.
- `calendar_list_events` defaults to `primary`; explicit
  `all_readable_calendars=true` performs a bounded-concurrency aggregate across
  the complete readable CalendarList, then globally sorts and truncates events.
- Event details include bounded description, location, Google event type and
  birthday properties. Attendees, conference credentials and attachments remain
  excluded from the model-facing result.
- Provider implementations do not expose tools by themselves. A feature must
  also be explicitly listed in `internal/catalog`, which is the production
  allowlist.
- The Authorizer role cannot read Google credentials.
- The MCP role cannot read the Hermes client-token Secret. Its AWS operations
  permissions are limited to the current Stack, two functions, three log groups
  the configured Hermes managed node, account Cost Explorer reads and
  `ViewBudget` on two fixed budget ARNs.

Infrastructure lives in [`deploy/aws/`](./deploy/aws/README.md) and uses native
CloudFormation. GitHub App and protected-branch integration is documented in
[`docs/GITHUB_CREDENTIALS.md`](./docs/GITHUB_CREDENTIALS.md).

## Extension model

There are three different extension cases:

1. Add a tool to an existing feature: implement the handler in
   `internal/tools/<feature>` and register it from that module's `Register`.
2. Add another provider for an existing feature: implement the provider
   interface in `internal/providers/<feature>`, then add one lazy factory to
   `internal/features/<feature>/definition.go`.
3. Add a new feature: add its typed config, provider interface/adapters, tool
   module and feature definition, then add exactly one entry to
   `internal/catalog/catalog.go`. CloudFormation must separately declare its
   mode, Secret and least-privilege IAM access.

Credential providers are a separate extension surface. Add a server-defined
profile and purpose-specific issuer/Secret role without allowing callers to
select raw targets or permissions. Add protocol adapters to the generic client
lease core; never put credential leasing behind an MCP tool.

The generic feature resolver only understands a feature name, selected mode
and a map of lazy provider factories. It has no Calendar, Gmail, Google or AWS
business logic. Disabled factories are never called, and a provider package
cannot expose a tool unless the production catalog includes its feature.
