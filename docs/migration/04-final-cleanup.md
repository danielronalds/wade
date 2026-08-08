# Slice 4: Final cleanup

## Status

Complete.

## Goal

Remove residual architecture from the old service and repository pattern,
validate the external contract, and update permanent project documentation.

Read [`PLAN.md`](PLAN.md) and
[`API_ENDPOINT_PROPOSAL.md`](API_ENDPOINT_PROPOSAL.md) before implementing.

## Implementation checklist

- [x] Confirm `internal/services` no longer exists.
- [x] Confirm `internal/repositories` no longer exists.
- [x] Search all Go files for imports from either old path.
- [x] Remove temporary migration adapters and obsolete TODO comments.
- [x] Confirm every controller depends on aggregate-wide Model interfaces.
- [x] Confirm Models depend only on Model-owned infrastructure interfaces and
      technical infrastructure types.
- [x] Confirm infrastructure never imports Models or controllers.
- [x] Confirm exported functions and methods precede private functions and
      methods in every changed file.
- [x] Confirm exported Go APIs have concise contract comments.
- [x] Regenerate OpenAPI and inspect the diff for accidental contract changes.
- [x] Regenerate the frontend client and embedded assets through the standard
      tasks.
- [x] Confirm the frontend requires no behavioural change.
- [x] Verify `AGENTS.md` matches the implemented Controllers, Models,
      Infrastructure structure and remove its temporary legacy-migration note.
- [x] Confirm README architecture references are current.
- [x] Run the full test, lint, race, and lifecycle validation suite.

## Contract audit

Verify preservation of:

- Routes and operation IDs.
- Request and response JSON shapes.
- Problem codes and status codes.
- Workspace and repository identity rules.
- Active-workspace targeted loading.
- Worktree creation and close-before-remove behaviour.
- Terminal process, buffer, reconnect, and WebSocket behaviour.
- Review snapshot identity and revision semantics.
- Settings defaults, compatibility, and runtime reload behaviour.

## Validation

```sh
mise run test
mise run lint:openapi
mise run lint:fmt
mise run lint:vet
```

Also run focused race tests and the isolated server lifecycle smoke test described
in `AGENTS.md`. Confirm no stale test processes remain.

## Handoff

- Last completed: Removed residual migration artefacts, completed the dependency
  and contract audits, regenerated backend and frontend outputs, and aligned
  project documentation with the implemented architecture.
- Next action: Use `PLAN.md` and the completed slice records to build permanent
  architecture documentation.
- Current failures: None.
- Last validation: `mise run test`, `mise run lint:openapi`,
  `mise run lint:fmt`, and `mise run lint:vet` passed. Focused race tests passed
  for Workspaces, Repositories, Terminals, ReviewSnapshots, Settings, and HTTP
  controllers. The isolated managed and foreground lifecycle smoke test passed
  without stale processes or sockets.
- Important context: OpenAPI regeneration produced no contract diff and the
  frontend required no behavioural changes.
