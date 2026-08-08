# Slice 1: Local development core

## Status

Planned.

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

- [ ] Add filesystem infrastructure with generic file operations.
- [ ] Add shared, concurrency-safe workspace discovery under filesystem.
- [ ] Preserve configured-directory order, duplicate basename precedence,
      canonical paths, and fresh discovery behaviour.
- [ ] Add Git infrastructure with typed technical outputs and cohesive external
      operations.
- [ ] Add GitHub infrastructure for repository listing, cloning, repository
      links, and optional pull request lookup.
- [ ] Add Linear infrastructure with `TicketForBranch(branch) (*Ticket, error)`.
- [ ] Add PTY infrastructure for start, read, write, resize, and close.
- [ ] Move infrastructure tests or add equivalent final-layer tests.

### Workspaces Model

- [ ] Create the agreed Model types, commands, errors, validators, configuration,
      and constructor.
- [ ] Implement `List`, `ListByIDs`, `Get`, `Materialise`, `ResolveLinks`, and
      `Configure`.
- [ ] Keep Materialise atomic with respect to a workspace ID or target path.
- [ ] Preserve best-effort optional provider links and log enrichment failures.
- [ ] Move workspace domain tests into the Model package.

### Repositories Model

- [ ] Create the agreed flat Model and its eleven public operations.
- [ ] Merge local repository identity, worktree, branch, ignored-file copy, and
      validation behaviour.
- [ ] Add bulk and targeted workspace-context queries.
- [ ] Retain `WorkspaceIDsByRemoteRepository` as a bulk optimisable query.
- [ ] Add per-repository mutation locking.
- [ ] Return detached Repository, WorkspaceContext, Worktree, and Branch values.
- [ ] Move repository and worktree domain tests into the Model package.

### RemoteRepositories Model

- [ ] Implement the focused `List` operation.
- [ ] Map and validate GitHub technical repository results.
- [ ] Remove cloning and local workspace mapping from this Model.
- [ ] Move remote repository tests into the Model package.

### Terminals Model

- [ ] Move terminal lifecycle and registry behaviour into the Model.
- [ ] Separate detached `Terminal` resources from live `TerminalSession` handles.
- [ ] Preserve terminal natural IDs, idempotent Put, buffering, selected-agent
      behaviour, control messages, and process cleanup.
- [ ] Implement all agreed lifecycle, input, activity, connection,
      configuration, and Close methods.
- [ ] Remove directory-based terminal closure from the public API.
- [ ] Move terminal domain tests into the Model package and PTY tests into
      infrastructure.

### Controllers

- [ ] Add aggregate-wide controller Model interfaces in
      `internal/controllers/model_interfaces.go`.
- [ ] Decode Model-owned command types directly where transport shapes match.
- [ ] Compose Workspace repository context, terminal activity, and links.
- [ ] Preserve targeted active-workspace loading through `ListByIDs` and
      `ListWorkspaceContextsByIDs`.
- [ ] Compose RemoteRepository local workspace IDs through the bulk Repository
      query.
- [ ] Move worktree resolution into the Repositories Model API.
- [ ] Preserve inspect, close terminals, remove, and revalidate ordering for
      worktree deletion.
- [ ] Update terminal WebSocket handling to use `TerminalSession`.
- [ ] Rename central service error mapping to Model error mapping while
      preserving every problem code and status.
- [ ] Update controller orchestration tests and reusable Model fakes.

### Composition and deletion

- [ ] Wire infrastructure and Models in `internal/app`.
- [ ] Share one filesystem workspace discovery instance across consuming Models.
- [ ] Keep the existing Settings bootstrap temporarily, mapping its runtime
      configuration into final Model configuration types.
- [ ] Update runtime settings application to configure the new Models until
      Slice 3 moves orchestration into the Settings controller.
- [ ] Delete `internal/services/workspaces`.
- [ ] Delete `internal/services/gitrepositories`.
- [ ] Delete `internal/services/worktrees`.
- [ ] Delete `internal/services/remoterepositories`.
- [ ] Delete `internal/services/terminals`.
- [ ] Delete `internal/services/workspacequeries`.
- [ ] Delete migrated files from `internal/repositories`.
- [ ] Confirm no compatibility adapters remain for migrated domains.

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

- Last completed: Not started.
- Next action: Confirm the overall migration design, then begin infrastructure
  and Model construction as one vertical slice.
- Current failures: None.
- Last validation: Not run for this slice.
- Important context: Do not adapt the old services to new infrastructure as a
  standalone phase.
