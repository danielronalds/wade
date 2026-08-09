# Aggregate Models

Each package under `internal/models` represents a domain aggregate and exposes one application-scoped, concurrency-safe `Model`. Models are constructed through dependency injection and do not use package-global mutable state. The Model is the authoritative boundary for its domain behaviour, validation, state and high-level workflows, returning detached value snapshots rather than exposing mutable internal state.

Operations that perform IO accept a request context, but Models never retain request contexts beyond an operation's lifetime.

WADE has six aggregate Models:

- [`workspaces`](#workspaces)
- [`repositories`](#repositories)
- [`remoterepositories`](#remoterepositories)
- [`terminals`](#terminals)
- [`reviewsnapshots`](#reviewsnapshots)
- [`settings`](#settings)

Related concepts remain within their owning aggregate, such as worktrees and branches within `repositories`, rather than becoming separate Models or introducing Model-to-Model dependencies.

## Workspaces

The `workspaces` Model owns workspace discovery, identity, lookup, materialisation and provider links. It lists all or selected workspaces, loads individual workspaces and clones remote repositories into configured destinations.

It works with `filesystem` workspace discovery, along with `github` and `linear` clients. Workspace state is read fresh, while conflicting materialisations are serialised by workspace identity and target path.

Controllers add repository context and terminal activity to workspace responses. Provider-link enrichment is best-effort, so optional GitHub or Linear failures do not prevent workspace loading.

## Repositories

The `repositories` Model owns local repository identity, workspace Git contexts, worktrees, branches and remote mappings. It loads repositories, maps remotes, and lists or mutates worktrees and branches.

Repository data is loaded through the `filesystem` and `git` infrastructure packages. It is read fresh, worktree mutations are serialised per repository, and all returned resources are detached values.

Worktrees and branches remain part of the `repositories` aggregate. Controllers close a worktree's terminals before removal, but the Model remains authoritative and independently revalidates removability.

## RemoteRepositories

The `remoterepositories` Model provides a read-only view of repositories available from GitHub. It validates required provider fields, maps results into domain resources and returns them in a stable order.

It depends only on `github` infrastructure and loads provider state on each operation rather than retaining a repository cache.

Workspace materialisation belongs to `workspaces`. Controllers add local workspace IDs through one bulk `repositories` query.

## Terminals

The `terminals` Model owns PTY-backed terminal resources, processes, buffers, clients and lifecycle. It creates terminals idempotently, handles input and connections, reports workspace activity, applies configuration and releases runtime resources.

It uses `filesystem` workspace discovery and `pty` infrastructure. Its application-scoped process registry is concurrency-safe, so concurrent creation of the same terminal resolves to one PTY.

API operations return detached `Terminal` values. Controllers use explicit live `TerminalSession` handles and remain responsible for WebSocket transport and protocol orchestration.

## ReviewSnapshots

The `reviewsnapshots` Model owns point-in-time review snapshots, revision pinning, file identity and scoped content loading. It creates, retrieves and deletes snapshots and loads their file contents.

It uses the `filesystem`, `git` and `github` infrastructure packages, including the shared workspace-discovery capability. Snapshots remain in a concurrency-safe in-memory registry until deletion or server shutdown, and callers receive defensive copies.

Snapshot scopes use captured file identity and pinned revisions. The `current` scope intentionally reads the workspace's current filesystem contents instead.

## Settings

The `settings` Model owns persisted settings and their resolved runtime configuration. It ensures the settings file exists, loads startup configuration, persists updates and reloads out-of-band changes.

It uses the `filesystem` settings-file capability and `environment` infrastructure. Persistence mutations are serialised while defaults, validation, legacy keys, unknown keys, normalisation and environment precedence are preserved.

`settings` remains independent of other Models. The HTTP controller coordinates runtime reconfiguration, while the CLI and server startup share the same `settings` Model.
