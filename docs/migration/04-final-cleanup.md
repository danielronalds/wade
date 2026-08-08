# Slice 4: Final cleanup

## Status

Planned. Depends on Slices 1 through 3.

## Goal

Remove residual architecture from the old service and repository pattern,
validate the external contract, and update permanent project documentation.

Read [`PLAN.md`](PLAN.md) and
[`API_ENDPOINT_PROPOSAL.md`](API_ENDPOINT_PROPOSAL.md) before implementing.

## Implementation checklist

- [ ] Confirm `internal/services` no longer exists.
- [ ] Confirm `internal/repositories` no longer exists.
- [ ] Search all Go files for imports from either old path.
- [ ] Remove temporary migration adapters and obsolete TODO comments.
- [ ] Confirm every controller depends on aggregate-wide Model interfaces.
- [ ] Confirm Models depend only on Model-owned infrastructure interfaces and
      technical infrastructure types.
- [ ] Confirm infrastructure never imports Models or controllers.
- [ ] Confirm exported functions and methods precede private functions and
      methods in every changed file.
- [ ] Confirm exported Go APIs have concise contract comments.
- [ ] Regenerate OpenAPI and inspect the diff for accidental contract changes.
- [ ] Regenerate the frontend client and embedded assets through the standard
      tasks.
- [ ] Confirm the frontend requires no behavioural change.
- [ ] Verify `AGENTS.md` matches the implemented Controllers, Models,
      Infrastructure structure and remove its temporary legacy-migration note.
- [ ] Update README architecture references if any are stale.
- [ ] Run the full test, lint, race, and lifecycle validation suite.

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

- Last completed: Not started.
- Next action: Wait for Slice 3 acceptance criteria.
- Current failures: None.
- Last validation: Not run for this slice.
- Important context: This slice must not introduce new architecture or API
  behaviour. It validates and documents the completed migration.
