# Credential Lease Service

This Go Lambda issues short-lived credentials from server-defined profiles. It
is intentionally separate from MCP: lease values are returned only through an
API Gateway `AWS_IAM` route to the EC2 host provisioner and never become an MCP
tool result.

## Current profile

`github-self-management` is the first provider profile. The deployment fixes:

- GitHub App ID and installation ID;
- one immutable repository ID and its expected owner/name;
- `contents: write` and `pull_requests: write` permissions;
- `github.com` HTTPS as the only credential target; and
- GitHub's one-hour installation-token lifetime.

The caller sends only:

```json
{"profile":"github-self-management"}
```

It cannot select a Secret, repository, upstream URL, permissions, or expiry.
The response uses the provider-neutral lease schema under `internal/lease`.
Sensitive responses carry `Cache-Control: no-store`; audit events contain only
caller, profile, provider, lease ID, and expiry.

One HTTP handler serves every registered profile. `CREDENTIAL_PROVIDERS` names
only provider adapters compiled into the service; the provider composition
package loads their namespaced configuration and builds an exact profile
registry. Adding another provider changes that composition package, not the
Lambda entry point or route handler.

## Secret format

`GITHUB_PRIVATE_KEY_SECRET_ARN` may contain the raw RSA PEM or this JSON shape:

```json
{"private_key_pem":"<GitHub App private key PEM supplied by secret manager>"}
```

Populate the Secret out of band. Never put the key in CloudFormation
parameters, logs, shell arguments, chat, or this repository.

## Packages

| Path | Responsibility |
|---|---|
| `cmd/lambda` | Generic AWS startup and Lambda transport wiring |
| `internal/app` | One strict profile-routed HTTP handler and redacted audit |
| `internal/providers/githubapp` | GitHub provider configuration, App JWT signing, and installation-token issuance |
| `internal/lease` | Provider-neutral request and lease schema |
| `internal/provider` | Provider-neutral issuer interface and duplicate-safe profile registry |
| `internal/providers` | Compiled provider allowlist and service composition |
| `internal/secrets` | Narrow Secrets Manager reader |
| `internal/transport` | API Gateway v2 and IAM-caller adapter |

Run from the `personal-tools` directory with `make test-credentials`, or use
the repository-wide `make check` and `make build` targets.
