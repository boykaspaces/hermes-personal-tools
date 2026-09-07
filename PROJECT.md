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
- `tasks/` and `.hermes/` own only this public repository's maintenance state.

## Context entry points

| Artifact | Purpose | Read when |
|---|---|---|
| [`.hermes/context-kit.json`](./.hermes/context-kit.json) | Adopted Context Kit contract | Changing or validating project context |
| [`.hermes/context-index.md`](./.hermes/context-index.md) | Current-first repository context | Starting or resuming maintenance |
| [`.hermes/state.md`](./.hermes/state.md) | Current repository summary | Asking what work is active |
| [`tasks/current.md`](./tasks/current.md) | Primary active Task pointer | Continuing current component work |
| [`docs/decisions/README.md`](./docs/decisions/README.md) | Repository decision index | Work depends on a durable local decision |

## Boundaries

- No production identifiers, deployment records, credentials, or environment
  overrides are stored in this public repository. Consuming deployment
  Tasks/Checkpoints remain private; public maintenance context stays generic.
- Provider permissions and targets are deployment-selected or server-defined;
  untrusted callers cannot request arbitrary targets, permissions, or expiry.
