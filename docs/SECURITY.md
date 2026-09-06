# Security Model

## Trust zones

| Zone | May hold | Must not hold |
|---|---|---|
| Provider issuer Lambda | Purpose-specific long-lived provider secret | Arbitrary caller-selected targets or permissions |
| Hermes host provisioner | EC2 IAM authority and short-lived lease | Provider private key |
| Coding container | Read-only short-lived lease required by its adapter | EC2 role, provider private key, container-engine socket |
| Model/tool context | Redacted metadata and command results | Tokens, private keys, Secret values |

## Credential rules

- The lease endpoint uses `AWS_IAM`, not the MCP bearer Authorizer.
- Profiles are server-defined and map to fixed issuer configuration.
- The GitHub issuer requests one numeric repository ID and a fixed permission
  map. Responses are not cached by intermediaries or logged.
- The provisioner writes atomically to tmpfs with mode `0600` and refreshes
  before expiry.
- Credential adapters implement lookup only; `store` and `erase` do not persist
  provider tokens.
- The coding container can technically read its mounted short-lived token. This
  is an accepted residual risk bounded by expiry, one-repository scope,
  protected refs, isolated IAM, and restricted egress.

## Deployment rules

- Populate Secrets Manager values out of band; never use CloudFormation
  parameters, repository files, chat, or build logs for secret values.
- Attach deployment and credential-operator policies only to reviewed operator
  identities and remove temporary deployment permissions after use.
- Keep GitHub protected-ref enforcement active before issuing a write-capable
  lease.
- Treat tests and examples as non-production values.
