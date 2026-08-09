# HTTP Controllers

HTTP and WebSocket controllers live in `internal/controllers` and form WADE's transport and orchestration layer. They translate external requests into Model operations without owning domain behaviour.

The HTTP controller package defines one cohesive, consumer-owned interface for each aggregate Model in `model_interfaces.go`. Controllers depend on these interfaces, never call infrastructure directly, and coordinate workflows across Models when an API resource spans aggregate boundaries.

Controllers handle transport concerns such as malformed JSON, query parameters, HTTP responses and WebSocket upgrades. Models retain domain validation and expose typed errors, which controllers centrally map to stable `application/problem+json` responses.

WADE's HTTP controller set includes:

- [`Workspaces`](#workspaces)
- [`Repositories`](#repositories)
- [`RemoteRepositories`](#remoterepositories)
- [`Worktrees`](#worktrees)
- [`Terminals`](#terminals)
- [`ReviewSnapshots`](#reviewsnapshots)
- [`Settings`](#settings)
- [`Docs`](#docs)
- [`Page`](#page)

## Workspaces

The `Workspaces` controller lists, loads and materialises workspaces. Materialisation requests are decoded directly into the aggregate-owned command type before the created workspace is returned with its resource location.

Workspace API resources combine workspace identity, Git context, terminal activity and provider links from several aggregates.

Active workspace requests first identify workspaces with running terminals, then perform targeted workspace and repository loading. This avoids scanning every configured workspace before filtering the response.

### Depends on

- [`workspaces`](backend-models.md#workspaces)
- [`repositories`](backend-models.md#repositories)
- [`terminals`](backend-models.md#terminals)

## Repositories

The `Repositories` controller loads a local repository by ID and returns the detached resource directly. Repository identity validation and lookup behaviour remain in the domain layer.

### Depends on

- [`repositories`](backend-models.md#repositories)

## RemoteRepositories

The `RemoteRepositories` controller lists provider repositories and adds their local workspace IDs. It performs one bulk local mapping query for all returned remote repository IDs rather than querying each repository separately.

### Depends on

- [`remoterepositories`](backend-models.md#remoterepositories)
- [`repositories`](backend-models.md#repositories)

## Worktrees

The `Worktrees` controller lists, creates and removes worktrees and lists local or remote branches. It handles request decoding, branch filters, resource locations and HTTP response semantics.

Removal first loads the target worktree. If its detached value is removable, the controller closes the associated workspace terminals before requesting removal. The domain layer then revalidates removability while holding its mutation lock, so controller orchestration does not weaken the invariant.

### Depends on

- [`repositories`](backend-models.md#repositories)
- [`terminals`](backend-models.md#terminals)

## Terminals

The `Terminals` controller exposes terminal listing, idempotent creation, lookup, deletion and input operations. REST operations return detached `Terminal` values without exposing process ownership.

For live connections, the controller upgrades the request to a WebSocket and obtains a `TerminalSession`. It streams terminal output as binary messages, sends binary client input to the session, and applies text control messages such as resize events. Origin checks and connection lifetime remain transport concerns, while PTY processes, buffering, replay and client registration remain in the domain layer.

### Depends on

- [`terminals`](backend-models.md#terminals)

## ReviewSnapshots

The `ReviewSnapshots` controller creates, retrieves and deletes snapshots. It also loads file comparisons for a snapshot, file ID and requested review scope.

The controller owns snapshot resource locations and HTTP response semantics. Snapshot identity, scope validation, pinned revisions and file-content behaviour remain in the domain layer.

### Depends on

- [`reviewsnapshots`](backend-models.md#reviewsnapshots)

## Settings

The `Settings` controller loads, updates and reloads settings. After an update or reload, it maps and applies the resolved runtime configuration across the running application.

A controller-level mutex covers persistence and all runtime configuration calls so concurrent requests cannot apply settings out of order. The controller is stored and called by pointer to prevent this mutex from being copied.

### Depends on

- [`settings`](backend-models.md#settings)
- [`workspaces`](backend-models.md#workspaces)
- [`repositories`](backend-models.md#repositories)
- [`terminals`](backend-models.md#terminals)

## Docs

The `Docs` controller serves the generated OpenAPI JSON and interactive documentation UI. The OpenAPI annotations defining the HTTP contract remain on the controller methods that implement each API operation.

### Depends on

- No Models

## Page

The `Page` controller serves the embedded frontend assets and application shell. It recognises frontend routes, serves the service worker without browser caching and returns not found responses for unknown page paths.

### Depends on

- No Models

