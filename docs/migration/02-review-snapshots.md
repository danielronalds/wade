# Slice 2: Review snapshots

## Status

Complete.

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

- [x] Create `internal/models/reviewsnapshots` with the agreed `Create`, `Get`,
      `FileContents`, and `Delete` surface.
- [x] Define cohesive Model-owned filesystem, Git, and GitHub interfaces.
- [x] Extend infrastructure technical operations only where required.
- [x] Keep window construction, revision pinning, parsing, and file loading
      private to the Model.
- [x] Preserve in-memory snapshot lifetime and deletion behaviour.
- [x] Preserve snapshot-scoped file identity and pinned revision semantics.
- [x] Return defensive copies of every snapshot and nested mutable value.
- [x] Switch the ReviewSnapshots controller to the aggregate-wide Model
      interface.
- [x] Preserve all review scope names, responses, errors, and status codes.
- [x] Move service and registry tests into Model tests with fake infrastructure.
- [x] Delete `internal/services/review`.
- [x] Confirm no compatibility adapter remains.

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

- Last completed: Migrated ReviewSnapshots into its aggregate Model, switched
  controller and application wiring, added typed GitHub pull request
  infrastructure, and removed the legacy review Service.
- Next action: Review `03-settings-and-bootstrap.md` before beginning the
  Settings and bootstrap migration.
- Current failures: None.
- Last validation: `mise run test`, `mise run lint:openapi`,
  `mise run lint:fmt`, and `mise run lint:vet` passed. The focused
  ReviewSnapshots race test passed.
- Important context: Working-tree contents are captured by snapshots while the
  `current` scope continues to read current filesystem contents.
