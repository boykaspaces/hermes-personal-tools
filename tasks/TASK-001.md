# TASK-001: Adopt Cross-Repository Task Routing

Status: Completed
Type: Component
Priority: High
Parent System Task: personal-hermes-agent:TASK-011

## Goal

Add public, index-first repository-maintenance context and route system-scoped
Personal Tools work through the shared multi-repository protocol without
introducing credentials or private deployment state.

## Completed

- Added current-first context, Task, Checkpoint, and decision indexes.
- Added AI routing for local Component Tasks, parent System Tasks, and
  inaccessible integration repositories.
- Extended repository documentation and validation for the new context paths.

## Remaining

None.

## Blockers

None.

## Relevant Files

- `AGENTS.md`
- `PROJECT.md`
- `.hermes/`
- `tasks/`
- `docs/FILE_MAP.md`
- `scripts/validate.sh`

## Next Step

None. Future component work creates a new local Task and links a parent System
Task only when system scope exists.

## Result

The public Personal Tools repository now has validated index-first maintenance
context and routes cross-repository work without depending on private ops
access.
