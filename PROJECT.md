# Hermes Personal Tools

Project ID: hermes-personal-tools
Name: Hermes Personal Tools
Status: Active

## Goal

Provide small, auditable personal-service modules and a reusable short-lived
credential delivery path for Hermes without exposing long-lived provider
credentials to the agent or coding container.

## Sources of truth

- Go source and tests own service behavior.
- `deploy/aws/cloudformation.yaml` owns AWS resource and IAM role behavior.
- The nearest component README owns current operating and extension guidance.
- `docs/SECURITY.md` owns the repository-level trust-boundary summary.
- `docs/FILE_MAP.md` owns navigation only.

## Boundaries

- No production identifiers, deployment records, credentials, or environment
  overrides are stored in this public repository.
- Provider permissions and targets are deployment-selected or server-defined;
  untrusted callers cannot request arbitrary targets, permissions, or expiry.
