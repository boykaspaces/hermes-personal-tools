# File Map

Status: Current index

## Services

| Path | Owns | Read when |
|---|---|---|
| `services/authorizer/` | Dedicated bearer-token Authorizer Lambda and Secret cache | Changing gateway client authentication |
| `services/mcp/` | MCP protocol, feature catalog, tools, providers, audit, and API adapter | Changing personal tools or provider behavior |
| `services/credentials/` | IAM-authenticated lease API, profile registry, issuers, and GitHub App adapter | Changing credential issuance or adding a provider |

## Clients

| Path | Owns | Read when |
|---|---|---|
| `clients/credential-agent/cmd/hermes-credential-provisioner/` | Host refresh process and CLI | Changing lease acquisition or lifecycle |
| `clients/credential-agent/cmd/git-credential-hermes/` | Git Credential Helper entry point | Changing Git integration |
| `clients/credential-agent/internal/provisioner/` | SigV4 lease request, validation, redacted logging, and atomic write | Changing host delivery behavior |
| `clients/credential-agent/internal/lease/` | Provider-neutral lease loading, expiry, and target matching | Changing the client lease contract |
| `clients/credential-agent/internal/gitcredential/` | Git helper protocol and non-persistence behavior | Adding or changing Git credential semantics |

## Deployment and validation

| Path | Owns | Read when |
|---|---|---|
| `deploy/aws/cloudformation.yaml` | API, Lambdas, roles, Secrets, logs, parameters, conditions, and outputs | Deploying or changing AWS resources |
| `deploy/aws/artifacts-cloudformation.yaml` | Private versioned artifact bucket | Creating artifact storage |
| `deploy/aws/policies/*.json.tmpl` | Parameterized operator policy examples | Granting reviewed deployment or secret-write access |
| `deploy/aws/README.md` | Current build, package, and deployment flow | Operating the AWS deployment |
| `deploy/aws/validate-template.sh` | CloudFormation and policy-template checks | Validating deployment changes |
| `Makefile` | Go tests, vet, lint, race, and release builds | Validating source changes |
| `scripts/validate.sh` | Repository-wide public-boundary and documentation checks | Before release or publication |

Package-level source and tests own implementation truth. Open only the package
that owns the behavior being changed.
