# Internal Model refactor plan

## Status

This is the authoritative architecture and progress index for the internal Model
migration. Update it as decisions are made and after every implementation
session. No implementation should begin until the design has been reviewed and
confirmed.

`API_ENDPOINT_PROPOSAL.md` remains the authoritative external API contract. The
external API refactor is complete. This plan reorganises the internal Go code so
that it reflects the established API domain.

| Slice | Status | Execution document |
| --- | --- | --- |
| Local development core | Complete | [`01-local-development-core.md`](01-local-development-core.md) |
| Review snapshots | Complete | [`02-review-snapshots.md`](02-review-snapshots.md) |
| Settings and bootstrap | Planned | [`03-settings-and-bootstrap.md`](03-settings-and-bootstrap.md) |
| Final cleanup | Planned | [`04-final-cleanup.md`](04-final-cleanup.md) |

Current phase: Slice 2 complete; Settings and bootstrap planned.

## Session handoff protocol

At the end of every implementation session:

1. Update the active slice's checklist.
2. Record its last completed task, next action, current failures, and last
   validation result.
3. Update the status table above.
4. Record any new cross-cutting architectural decision in this document rather
   than duplicating it in a slice document.

A new implementation session should read `AGENTS.md`, this document, the active
slice document, the relevant sections of `API_ENDPOINT_PROPOSAL.md`, and then the
relevant source files.

## Purpose

Replace the current Controllers, Services, Repositories structure with a direct
MVC-style design:

```text
controllers -> models -> infrastructure
```

Aggregate Models will combine the domain behaviour currently split between
services and domain repositories. They will expose high-level operations to thin
controllers. Infrastructure packages will perform mechanical external IO.

This is an internal architectural refactor. It must not intentionally change API
or runtime behaviour.

## Agreed aggregate Models

Use one package per aggregate Model:

```text
internal/models/
  remoterepositories/
  repositories/
  reviewsnapshots/
  settings/
  terminals/
  workspaces/
```

The Repositories Model owns repositories, worktrees, branches, and workspace Git
contexts. Worktree and Branch do not become independent aggregate Models.

Each package exposes a long-lived, concurrency-safe `Model` constructed through
normal dependency injection. Serialisable domain entities are detached value
snapshots rather than pointers into mutable Model state.

Keep resource, command, and value types with their owning aggregate. Do not add a
generic `models/common` or shared-types package. Where similar concepts have
different established API shapes, controllers map between aggregate-owned types
rather than introducing a broad shared type with many optional fields. A small
amount of deliberate duplication is preferred over weakening aggregate
boundaries.

## Agreed infrastructure

Use these top-level infrastructure packages:

```text
internal/infrastructure/
  environment/
  filesystem/
    filesystem.go
    settings_file.go
    workspace_discovery.go
  git/
  github/
  linear/
  pty/
```

Responsibilities:

- `environment` reads process environment values, the home directory, and the
  inherited shell.
- `filesystem` performs generic filesystem operations, settings-file IO, and
  configured workspace discovery and resolution.
- `git` executes Git operations and parses output into technical Git types.
- `github` executes GitHub provider operations, including remote listing,
  cloning, repository links, and pull request lookup.
- `linear` identifies a ticket associated with a branch and returns provider
  ticket data.
- `pty` starts and manages low-level PTY processes.

Workspace discovery belongs in `infrastructure/filesystem`, not in a separate
repository or pseudo-Model package. One application-scoped discovery instance is
shared by Models that need workspace locations.

Settings-file infrastructure owns mechanical path, read, existence, and write
behaviour. Preserve the current straightforward direct-write behaviour rather
than adding temporary-file and atomic-rename machinery during this refactor. The
Settings Model owns defaults, JSON interpretation, legacy-key compatibility,
unknown-key preservation, validation, and runtime resolution.

## Infrastructure interface standard

Each Model owns cohesive infrastructure interfaces shaped around that Model's
needs. Prefer substantial interfaces over many one-method interfaces, but do not
create one universal interface containing unrelated capabilities.

Infrastructure parses external formats into infrastructure-owned technical
result types. Models map those technical types into domain and API entities.
Infrastructure must not import Model packages.

Thick infrastructure operations stop at cohesive external-system operations.
Infrastructure owns command syntax, execution, timeouts, and output parsing.
Models own multi-step workflows and policy, including remote preference, branch
selection, target paths, idempotency, ignored-file copying, removability, and
domain error selection.

The same concrete infrastructure client may satisfy interfaces owned by several
Models.

## Controller boundary

Controllers are thin orchestration and transport layers. They:

- Decode and validate HTTP syntax.
- Call high-level aggregate Model methods.
- Coordinate workflows spanning multiple Models.
- Compose API resources from detached Model snapshots.
- Map typed Model errors to `application/problem+json`.
- Write HTTP and WebSocket responses.

When an HTTP request directly represents a Model command, controllers decode
into the Model-owned command type, such as `workspaces.MaterialiseRequest`,
`repositories.CreateWorktreeRequest`, `terminals.Input`, or `settings.Settings`.
Controller-owned request DTOs remain only when the transport shape genuinely
differs from the Model command.

Controllers do not:

- Execute Git, GitHub, Linear, filesystem, or PTY operations directly.
- Receive or pass internal filesystem paths between Models.
- Reimplement domain validation.
- Own aggregate state or aggregate mutation locks.

Controllers depend on cohesive, consumer-defined Model interfaces. Define one
aggregate-wide interface per Model for the complete `internal/controllers`
package in `model_interfaces.go`, rather than creating a separate narrow
interface for each controller method or file. Interfaces include the high-level
surface consumed anywhere in the controller package, but no operations unused by
the controller layer. Controller tests share reusable fakes with default
implementations.

### Validation boundary

Controllers validate transport concerns such as malformed JSON, malformed HTTP
parameters, and route-specific filters.

Models validate workspace and repository IDs, branch references, worktree
removability, configured destinations, terminal identifiers and input, settings
invariants, review scopes, and other domain rules.

### Error boundary

Typed domain errors live in their aggregate Model package and contain no HTTP
status or problem-response details. Controllers centrally map those errors to
the existing problem codes, status codes, and messages.

Infrastructure failures are wrapped with operation context by Models.

## Cross-aggregate response composition

Controllers compose fields that come from multiple aggregates.

### Workspace responses

The Workspaces controller combines:

- Base workspace identity from Workspaces.
- Repository, worktree, and branch context from Repositories.
- Active terminal counts from Terminals.
- Provider links resolved through Workspaces using GitHub and Linear
  infrastructure.

The existing active-workspace optimisation must remain. The controller first
gets active workspace IDs from Terminals, then asks Workspaces and Repositories
for only those IDs. It must not list every workspace and filter afterwards.

### Remote repository responses

The RemoteRepositories controller combines:

- Provider repositories from RemoteRepositories.
- `localWorkspaceIds` from one bulk Repositories query keyed by remote repository
  ID.

### Workspace materialisation

`POST /api/v1/workspaces` is owned by the Workspaces Model because it creates a
Workspace. Workspaces validates the destination, enforces identity and path
uniqueness, clones through infrastructure, and returns a detached Workspace.
The controller enriches the created resource with repository context, links, and
terminal activity before returning it.

RemoteRepositories owns provider listing and description, not local workspace
materialisation.

## Provider link behaviour

Provider-specific mechanics belong to infrastructure:

```go
func (client Client) TicketForBranch(branch string) (*Ticket, error)
```

For Linear:

- `nil, nil` means the branch has no associated ticket.
- A non-nil ticket contains at least its key and URL.
- A non-nil error means ticket resolution failed.

GitHub pull request lookup follows the same optional-result approach.

Workspace link enrichment is best-effort:

- Clone and remote-listing failures remain fatal to their operations.
- Pull request or ticket enrichment failures do not fail workspace loading.
- Provider enrichment errors are logged.
- Failed optional links remain `null`.

## Model lifetime and state

Construct one long-lived instance of each aggregate Model per application. Pass
instances explicitly through dependency injection. Do not use package globals.

Long-lived Models must:

- Be safe for concurrent use.
- Never store request contexts.
- Close owned runtime resources during application shutdown.
- Return defensive copies of internal slices, maps, buffers, and optional values.

Read workspace, repository, branch, worktree, remote provider, and settings state
fresh on each operation. Do not introduce an entity cache or identity map.
Caching may be added later behind Model APIs only when profiling justifies it.

Intentionally retained state includes terminal processes, terminal buffers and
clients, active-agent selection, review snapshots, and runtime configuration.

## Concurrency

Models own concurrency control for their aggregate invariants:

- Workspaces serialises conflicting materialisation by workspace identity or
  target path.
- Repositories serialises worktree mutations by repository ID.
- Settings serialises persisted settings mutation.
- Terminals protects its process registry and preserves idempotent terminal
  creation.
- ReviewSnapshots protects its in-memory snapshot registry.

Prefer per-aggregate locking for long-running operations so unrelated aggregates
can proceed concurrently.

The Settings controller additionally serialises the complete update and reload
orchestration across persistence and runtime Model reconfiguration. This avoids
concurrent requests leaving persisted settings and runtime settings out of sync.
The Settings controller must be held and called by pointer so its mutex is not
copied.

## Worktree removal orchestration

Preserve close-before-remove behaviour:

1. The controller loads the target Worktree from Repositories.
2. If the detached snapshot says it is removable, the controller closes its
   workspace terminals.
3. The controller calls `RemoveWorktree`.
4. Repositories independently revalidates removability while holding its
   repository mutation lock.

The controller uses `IsRemovable` only to coordinate terminal closure. The Model
remains authoritative for the domain invariant and typed error.

## Terminal resource and session separation

Do not expose a live PTY-backed terminal object as an API entity.

- `Terminal` is a detached serialisable resource returned by `Get`, `List`, and
  `Put`.
- `TerminalSession` is an explicit live handle returned for WebSocket streaming.
- The Terminals Model owns process, registry, client, buffer, and locking details.
- The controller owns WebSocket upgrade and protocol orchestration through the
  session handle.

The agreed Model surface includes `List`, `Get`, idempotent `Put`, `Delete`,
`DeleteAll`, `Input`, `Connect`, `ActiveTerminalCount`, `ActiveWorkspaceIDs`,
`Configure`, and application-level `Close` operations. IO-capable resource
operations accept a request context. `CloseTerminalsForDirectory` is removed
because worktree controllers already have the target workspace ID.

`TerminalSession` exposes only `Output`, `Write`, `ApplyControlMessage`, and
`Close` for WebSocket streaming.

## Agreed ReviewSnapshots Model surface

The Model exposes `Create`, `Get`, `FileContents`, and `Delete`. Snapshot creation
and file-content loading accept request contexts because they perform IO. In-memory
get and delete operations do not require a context. Existing window-building and
raw file-loading helpers become private implementation details.

## Settings and composition roots

The Settings Model owns persisted settings and resolved startup configuration.
It is constructed before the server listener so startup configuration can be
loaded through the same Model used by the Settings API.

`cmd/wade/main.go` becomes the top-level composition root for infrastructure and
the Settings Model shared by CLI commands. `internal/app` remains the composition
root for the HTTP application and constructs the remaining Models and
controllers using the resolved runtime configuration and existing Settings
Model.

The `wade config` CLI controller consumes an injected Settings Model interface.
The Settings Model ensures the file exists and returns its location. Launching
`$EDITOR` remains a CLI controller concern.

The Settings HTTP controller coordinates runtime configuration application to
Workspaces, Repositories, and Terminals. Model `Configure` operations are
concurrency-safe and cannot fail after Settings has validated and resolved the
configuration.

The agreed Settings Model surface is `EnsureFile`, `Get`,
`LoadRuntimeConfiguration`, `Update`, and `Reload`. `Update` and `Reload` return
an `UpdateResult` containing the detached serialisable Settings and a neutral
resolved `RuntimeConfiguration`. Settings must not import other Model packages;
composition roots and the Settings controller map runtime values into each
aggregate configuration.

## Agreed Workspaces Model surface

```go
func New(
    files FileSystem,
    discovery WorkspaceDiscovery,
    github GitHub,
    linear Linear,
    configuration Configuration,
) *Model

func (model *Model) List(ctx context.Context) ([]WorkspaceSummary, error)
func (model *Model) ListByIDs(ctx context.Context, workspaceIDs []string) ([]WorkspaceSummary, error)
func (model *Model) Get(ctx context.Context, workspaceID string) (Workspace, error)
func (model *Model) Materialise(ctx context.Context, request MaterialiseRequest) (Workspace, error)
func (model *Model) ResolveLinks(ctx context.Context, linkContext LinkContext) (WorkspaceLinks, error)
func (model *Model) Configure(configuration Configuration)
```

Controllers enrich detached Workspace values before serialisation.
`ResolveLinks` may return partial links and an error so the controller can log
provider failures while preserving best-effort workspace loading.

## Agreed RemoteRepositories Model surface

```go
func New(github GitHub) *Model
func (model *Model) List(ctx context.Context) ([]RemoteRepository, error)
```

The focused read-only Model maps GitHub technical results into validated, sorted
RemoteRepository resources. Materialisation belongs to Workspaces and local
workspace ID composition belongs to the controller using Repositories.

## Repositories Model API design

The Repositories aggregate needs operations for:

- Repository lookup.
- Bulk and targeted workspace Git contexts.
- Mapping remote repository IDs to local workspace IDs.
- Worktree listing, lookup, creation, and removal.
- Branch listing.
- Runtime worktree configuration.

Aggregate-owned scoped façades were considered:

```go
repositoryModel.WorkspaceContexts().Get(ctx, workspaceID)
repositoryModel.Worktrees().Create(ctx, repositoryID, request)
repositoryModel.Branches().List(ctx, repositoryID, kind)
```

The agreed design is one flat, deep `repositories.Model`. Scoped façades would
exist only to simulate namespaces, add accessor calls and shallow public types,
complicate construction, and risk recreating service boundaries inside the
aggregate.

A cohesive Model with roughly ten to thirteen descriptive methods is not
unusually large in Go. Concern-based source files provide implementation
organisation, while consumer-defined controller interfaces ensure each
controller sees only the relevant subset.

The agreed flat Model surface is `Get`, `ListWorkspaceContexts`,
`ListWorkspaceContextsByIDs`, `GetWorkspaceContext`,
`WorkspaceIDsByRemoteRepository`, `ListWorktrees`, `GetWorktree`,
`CreateWorktree`, `RemoveWorktree`, `ListBranches`, and `Configure`.

`GetWorkspaceContext` returns `nil, nil` for a valid non-Git workspace and typed
errors for invalid or unknown workspace IDs. Bulk context queries return detached
values, with the targeted query keyed by workspace ID.

The bulk `WorkspaceIDsByRemoteRepository` query keeps remote-repository
composition efficient and allows the Model to optimise local repository scanning
behind its API. The active-workspace optimisation separately relies on targeted
workspace-context loading by workspace IDs.

## Package file organisation

Organise files by cohesive concern within each aggregate package. Do not force
all public methods into one large file.

Example:

```text
internal/models/repositories/
  model.go
  interfaces.go
  repositories.go
  workspace_contexts.go
  worktrees.go
  branches.go
  types.go
  errors.go
  validators.go
```

Within every file, exported functions and methods appear above private functions
and methods. Each Model package keeps its cohesive infrastructure interfaces in
`interfaces.go`, and its Model type, constructor, configuration, lifecycle, and
primary implementation in `model.go`. Use an import alias only when the declared
package name would otherwise collide. Do not create empty or unnecessary
convention files.

## External compatibility requirements

Preserve:

- Every `/api/v1` route.
- Request and response JSON shapes.
- OpenAPI operation IDs and domain schema names.
- Problem codes and HTTP status mappings.
- Workspace, repository, worktree, branch, and terminal identifiers.
- Terminal PTY, buffering, reconnect, input, and WebSocket behaviour.
- Review snapshot semantics.
- Settings-file defaults, legacy keys, and unknown-key preservation.
- Active-workspace performance characteristics.

Regenerate OpenAPI only to reflect internal Go type movement. Generated output
must not contain an intentional contract change. The frontend should require no
behavioural changes.

## Migration strategy

Migrate by complete vertical domain slices. Do not perform a preliminary phase
that merely moves repository implementations into infrastructure and rewires the
old services. Extract each infrastructure capability as part of introducing the
Model that consumes it, switch controllers and composition in the same slice,
and remove the replaced service and repository code immediately.

Keep the project compiling and tests passing at the end of each complete slice:

1. Migrate the connected local-development core: Workspaces, Repositories,
   RemoteRepositories, and Terminals. Introduce filesystem workspace discovery,
   Git, GitHub, Linear, filesystem, and PTY infrastructure as those Models are
   built. Move response composition and active filtering into controllers, move
   materialisation into Workspaces, separate Terminal resources from sessions,
   switch application wiring, and remove the replaced workspace, repository,
   worktree, remote-repository, terminal, and workspace-query services and
   repositories.
2. Migrate ReviewSnapshots using the established filesystem, Git, and GitHub
   infrastructure. Switch its controller and remove the old review service.
3. Migrate Settings and bootstrap configuration. Add environment and settings
   file infrastructure, switch HTTP and CLI controllers, move runtime
   reconfiguration into the Settings controller, and remove the config service
   and remaining settings repository code.
4. Remove any residual `internal/services` and `internal/repositories`
   directories, update imports and error mapping, regenerate OpenAPI, and confirm
   there are no compatibility adapters.
5. Verify `AGENTS.md` still matches the implemented architecture and remove its
   temporary legacy-migration note.

Use `git mv` for genuine file moves and renames during implementation. Split
mixed-responsibility files directly into their final Model and infrastructure
locations rather than creating temporary architectural layers.

## Testing strategy

### Controller tests

- Inject fake aggregate Model interfaces.
- Verify orchestration, response composition, typed error mapping, and call
  ordering.
- Preserve explicit tests for active-workspace targeted loading.
- Test remote repository local-workspace composition.
- Test close-before-remove worktree behaviour.
- Test serialised Settings update and reload orchestration.

### Model tests

- Inject fake infrastructure interfaces.
- Verify domain validation and typed errors.
- Verify aggregate mutation concurrency and idempotency.
- Verify configuration changes.
- Verify detached snapshots and defensive copying.
- Verify materialisation, repository/worktree behaviour, terminal lifecycle,
  review snapshots, and settings persistence through high-level methods.

### Infrastructure tests

- Use temporary directories and stub command runners.
- Verify workspace discovery and duplicate precedence.
- Verify settings-file reads and writes.
- Verify Git, GitHub, and Linear parsing and optional-result semantics.
- Verify PTY adaptation independently from terminal domain behaviour.

Keep route and OpenAPI drift tests as external contract protection. Final
validation must run:

```sh
mise run test
mise run lint:openapi
mise run lint:fmt
mise run lint:vet
```

Run focused race tests for stateful and mutation-heavy Model packages during
implementation.

## Final architecture documentation

`AGENTS.md` has been updated before implementation so migration sessions receive
the target Controllers, Models, Infrastructure rules rather than the legacy
service and repository rules. Keep it aligned with architectural decisions made
during implementation. After migration, remove its temporary note describing
`internal/services` and `internal/repositories` as legacy code.
