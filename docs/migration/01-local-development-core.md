# Slice 1: Local development core

## Status

Complete.

## Goal

Migrate Workspaces, Repositories, RemoteRepositories, and Terminals to their
final aggregate Models. Introduce the infrastructure those Models consume,
switch controllers and application composition, and remove the replaced service
and repository code in the same slice.

Read [`PLAN.md`](PLAN.md) and the relevant sections of
[`API_ENDPOINT_PROPOSAL.md`](API_ENDPOINT_PROPOSAL.md) before implementing. The
plan is authoritative for all cross-cutting architecture decisions, while the
proposal is authoritative for the external API contract.

## Final packages introduced

```text
internal/models/
  workspaces/
  repositories/
  remoterepositories/
  terminals/

internal/infrastructure/
  filesystem/
  git/
  github/
  linear/
  pty/
```

## Responsibility migration map

| Current location | Final responsibility |
| --- | --- |
| `internal/repositories/files.go` | Generic operations in `infrastructure/filesystem` |
| `internal/repositories/workspaces.go` | Workspace discovery in `infrastructure/filesystem/workspace_discovery.go` |
| `internal/repositories/workspace_directories.go` | Workspace directory resolution in filesystem infrastructure |
| `internal/repositories/git.go` | Typed Git client and technical result types in `infrastructure/git` |
| `internal/repositories/github.go` | Typed GitHub client and technical result types in `infrastructure/github` |
| `internal/services/workspaces` | `models/workspaces` |
| Linear parsing in workspace metadata | `infrastructure/linear`, mapped by Workspaces |
| `internal/services/gitrepositories` | Repository and workspace Git-context behaviour in `models/repositories` |
| `internal/services/worktrees` | Worktree and branch behaviour in `models/repositories` |
| `internal/services/remoterepositories` | Provider mapping in `models/remoterepositories`; materialisation moves to Workspaces |
| `internal/services/terminals/process.go` and `shell.go` | Low-level process behaviour in `infrastructure/pty` |
| Remaining `internal/services/terminals` behaviour | `models/terminals` |
| `internal/services/workspacequeries` | Removed; orchestration moves to controllers |

Mixed files should be split directly into final responsibilities. Do not create
temporary repository wrappers around new infrastructure.

## Implementation checklist

### Infrastructure

- [x] Add filesystem infrastructure with generic file operations.
- [x] Add shared, concurrency-safe workspace discovery under filesystem.
- [x] Preserve configured-directory order, duplicate basename precedence,
      canonical paths, and fresh discovery behaviour.
- [x] Add Git infrastructure with typed technical outputs and cohesive external
      operations.
- [x] Add GitHub infrastructure for repository listing, cloning, repository
      links, and optional pull request lookup.
- [x] Add Linear infrastructure with `TicketForBranch(branch) (*Ticket, error)`.
- [x] Add PTY infrastructure for start, read, write, resize, and close.
- [x] Move infrastructure tests or add equivalent final-layer tests.

### Workspaces Model

- [x] Create the agreed Model types, commands, errors, validators, configuration,
      and constructor.
- [x] Implement `List`, `ListByIDs`, `Get`, `Materialise`, `ResolveLinks`, and
      `Configure`.
- [x] Keep Materialise atomic with respect to a workspace ID or target path.
- [x] Preserve best-effort optional provider links and log enrichment failures.
- [x] Move workspace domain tests into the Model package.

### Repositories Model

- [x] Create the agreed flat Model and its eleven public operations.
- [x] Merge local repository identity, worktree, branch, ignored-file copy, and
      validation behaviour.
- [x] Add bulk and targeted workspace-context queries.
- [x] Retain `WorkspaceIDsByRemoteRepository` as a bulk optimisable query.
- [x] Add per-repository mutation locking.
- [x] Return detached Repository, WorkspaceContext, Worktree, and Branch values.
- [x] Move repository and worktree domain tests into the Model package.

### RemoteRepositories Model

- [x] Implement the focused `List` operation.
- [x] Map and validate GitHub technical repository results.
- [x] Remove cloning and local workspace mapping from this Model.
- [x] Move remote repository tests into the Model package.

### Terminals Model

- [x] Move terminal lifecycle and registry behaviour into the Model.
- [x] Separate detached `Terminal` resources from live `TerminalSession` handles.
- [x] Preserve terminal natural IDs, idempotent Put, buffering, selected-agent
      behaviour, control messages, and process cleanup.
- [x] Implement all agreed lifecycle, input, activity, connection,
      configuration, and Close methods.
- [x] Remove directory-based terminal closure from the public API.
- [x] Move terminal domain tests into the Model package and PTY tests into
      infrastructure.

### Controllers

- [x] Add aggregate-wide controller Model interfaces in
      `internal/controllers/model_interfaces.go`.
- [x] Decode Model-owned command types directly where transport shapes match.
- [x] Compose Workspace repository context, terminal activity, and links.
- [x] Preserve targeted active-workspace loading through `ListByIDs` and
      `ListWorkspaceContextsByIDs`.
- [x] Compose RemoteRepository local workspace IDs through the bulk Repository
      query.
- [x] Move worktree resolution into the Repositories Model API.
- [x] Preserve inspect, close terminals, remove, and revalidate ordering for
      worktree deletion.
- [x] Update terminal WebSocket handling to use `TerminalSession`.
- [x] Rename central service error mapping to Model error mapping while
      preserving every problem code and status.
- [x] Update controller orchestration tests and reusable Model fakes.

### Composition and deletion

- [x] Wire infrastructure and Models in `internal/app`.
- [x] Share one filesystem workspace discovery instance across consuming Models.
- [x] Keep the existing Settings bootstrap temporarily, mapping its runtime
      configuration into final Model configuration types.
- [x] Update runtime settings application to configure the new Models until
      Slice 3 moves orchestration into the Settings controller.
- [x] Delete `internal/services/workspaces`.
- [x] Delete `internal/services/gitrepositories`.
- [x] Delete `internal/services/worktrees`.
- [x] Delete `internal/services/remoterepositories`.
- [x] Delete `internal/services/terminals`.
- [x] Delete `internal/services/workspacequeries`.
- [x] Delete migrated files from `internal/repositories`.
- [x] Confirm no compatibility adapters remain for migrated domains.

## Acceptance criteria

- All affected API routes, schemas, operation IDs, errors, and status codes are
  unchanged.
- Active workspace loading inspects only active workspace IDs.
- Remote repository listing performs one bulk local mapping operation.
- Concurrent conflicting workspace and worktree mutations are serialised.
- Terminal resources are detached values and WebSockets use explicit sessions.
- No migrated controller imports `internal/services` or `internal/repositories`.
- Focused tests and the full project test suite pass.

## Validation

```sh
mise run test
mise run lint:openapi
mise run lint:fmt
mise run lint:vet
```

Run focused race tests for Workspaces, Repositories, and Terminals.

## Handoff

- Last completed: Migrated Workspaces, Repositories, RemoteRepositories, and
  Terminals into final Models; introduced filesystem, Git, GitHub, Linear, and
  PTY infrastructure; rewired controllers and application composition; removed
  the replaced services, workspace query layer, and migrated repository files.
- Next action: Review `02-review-snapshots.md` before beginning the ReviewSnapshots
  migration.
- Current failures: None.
- Last validation: `mise run test`, `mise run lint:openapi`, `mise run lint:fmt`,
  and `mise run lint:vet` passed. Focused race tests passed for Workspaces,
  Repositories, and Terminals.
- Important context: Review remains a legacy Service until Slice 2. Settings and
  bootstrap remain on the legacy configuration path until Slice 3. No
  compatibility adapters remain for the migrated Slice 1 domains.
