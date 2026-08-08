# Slice 2: Review snapshots

## Status

Planned. Depends on Slice 1.

## Goal

Migrate review snapshot behaviour into `internal/models/reviewsnapshots` using
the filesystem, Git, and GitHub infrastructure established in Slice 1.

Read [`PLAN.md`](PLAN.md) and the Review snapshots and API conventions sections
of [`API_ENDPOINT_PROPOSAL.md`](API_ENDPOINT_PROPOSAL.md) before implementing.

## Responsibility migration map

| Current location | Final responsibility |
| --- | --- |
| `internal/services/review/types.go` | ReviewSnapshots Model resource and value types |
| `internal/services/review/errors.go` | Typed ReviewSnapshots Model errors |
| `internal/services/review/registry.go` | Model snapshot registry and high-level operations |
| `internal/services/review/service.go` | Model domain workflow and private helpers |
| Review-specific methods in the old Git repository | Cohesive ReviewSnapshots Git interface implemented by `infrastructure/git` |
| Review file reads | ReviewSnapshots filesystem interface implemented by `infrastructure/filesystem` |
| Pull request lookup | ReviewSnapshots GitHub interface implemented by `infrastructure/github` |

## Implementation checklist

- [ ] Create `internal/models/reviewsnapshots` with the agreed `Create`, `Get`,
      `FileContents`, and `Delete` surface.
- [ ] Define cohesive Model-owned filesystem, Git, and GitHub interfaces.
- [ ] Extend infrastructure technical operations only where required.
- [ ] Keep window construction, revision pinning, parsing, and file loading
      private to the Model.
- [ ] Preserve in-memory snapshot lifetime and deletion behaviour.
- [ ] Preserve snapshot-scoped file identity and pinned revision semantics.
- [ ] Return defensive copies of every snapshot and nested mutable value.
- [ ] Switch the ReviewSnapshots controller to the aggregate-wide Model
      interface.
- [ ] Preserve all review scope names, responses, errors, and status codes.
- [ ] Move service and registry tests into Model tests with fake infrastructure.
- [ ] Delete `internal/services/review`.
- [ ] Confirm no compatibility adapter remains.

## Acceptance criteria

- Review API and OpenAPI output are unchanged.
- Snapshot results remain stable after workspace state changes.
- Current-content requests retain their existing semantics.
- Snapshot registry access is race-safe.
- No ReviewSnapshots code imports old services or repositories.

## Validation

```sh
mise run test
mise run lint:openapi
mise run lint:fmt
mise run lint:vet
```

Run focused race tests for the snapshot registry.

## Handoff

- Last completed: Not started.
- Next action: Wait for Slice 1 acceptance criteria.
- Current failures: None.
- Last validation: Not run for this slice.
- Important context: Keep infrastructure technical and leave review workflow in
  the Model.
