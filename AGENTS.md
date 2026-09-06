# Repository AI Instructions

## Entry and retrieval

- Start with `README.md`; use `.hermes/context-index.md` for repository
  maintenance state, then follow the narrowest component route.
- Prefer current source, CloudFormation, tests, and the nearest README.
- Use `docs/FILE_MAP.md` only when the owning path is not already known.

## Project and system context

- Use `project-context-management` from the reviewed Hermes Context Kit for
  this repository's Task, State, Checkpoint, ADR, and index mutations.
- Also use `multi-repo-system-management` when a Task has a parent System Task,
  changes another repository, advances a component revision, or requires an
  integration Handoff.
- If the integration repository is inaccessible, produce a validated Handoff;
  do not claim its Task, lock, deployment, or acceptance state changed.

## Boundaries

- Preserve service and client trust boundaries described in `docs/SECURITY.md`.
- Never add credentials, production identifiers, deployment records, or local
  environment files. Do not add consuming deployment context.
- Provider-specific configuration belongs with its provider adapter; generic
  registries and handlers must not acquire provider-specific fields.
- Update affected indexes when a path, responsibility, or extension route
  changes.

## Validation

- Run `make check` and `make build` for source changes.
- Run `deploy/aws/validate-template.sh` for AWS template changes.
- Run `scripts/validate.sh` before reporting a repository-wide change complete.
