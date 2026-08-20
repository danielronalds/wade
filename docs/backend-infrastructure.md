# Infrastructure Modules

Packages under `internal/infrastructure` provide concrete access to the operating system and external tools. They own mechanical IO, command execution, timeouts and external-format parsing, while Models retain domain validation, policy and multi-step workflows.

Each Model defines the infrastructure interfaces it consumes. Infrastructure packages return technical values and never depend on Models or controllers.

WADE has six infrastructure modules:

- [`environment`](#environment)
- [`filesystem`](#filesystem)
- [`git`](#git)
- [`github`](#github)
- [`linear`](#linear)
- [`pty`](#pty)

## Environment

The `environment` module reads process-level state needed during configuration and startup. It provides the current user's home directory, environment variables, the inherited shell and executable lookup through `PATH`.

The module is stateless and contains no configuration policy. The `settings` Model decides how environment values interact with persisted settings and defaults.

## Filesystem

The `filesystem` module provides general file and directory operations, settings-file access and workspace discovery. It creates directories, checks paths, reads and copies files, persists settings, and resolves workspace IDs to local directories.

Workspace discovery scans configured directories fresh, preserves directory precedence, skips duplicate IDs and canonical paths, and uses canonical paths for comparisons. Its configured directory list can be reloaded safely and is shared by the Models that need workspace locations.

The module owns mechanical filesystem behaviour rather than domain policy. Models decide which paths are valid, when files should be copied and how filesystem failures map to domain errors.

## Git

The `git` module runs bounded Git commands and parses their output into technical values. It supports repository and worktree identity, remotes, branches, worktree mutations, ignored paths, review diffs, revision resolution and content loading.

The client is stateless and receives the working directory and request context for each operation. Command failures and timeouts are returned to the consuming Model with the relevant technical output.

Models remain responsible for workflows such as branch selection, target-path construction, worktree removability and review-window construction. This keeps Git command syntax out of the domain layer without moving domain policy into infrastructure.

## GitHub

The `github` module integrates with GitHub through the `gh` CLI. It lists visible repositories, clones repositories, and resolves pull request URLs and metadata into provider-owned technical types.

Commands run with bounded contexts and provider output is parsed before being returned to Models. Consuming Models decide whether a provider failure is fatal, such as cloning failure, or best-effort, such as optional link enrichment.

## Linear

The `linear` module identifies Linear ticket keys embedded in branch names and builds issue references for a workspace slug supplied to each call. A branch without a ticket key returns no result rather than an error.

The client is stateless and contains only provider-specific parsing and URL construction. The `workspaces` Model owns enablement and workspace-selection policy, maps the technical ticket value into workspace link resources and decides how resolution failures affect enrichment.

## PTY

The `pty` module starts interactive shells and configured commands through operating-system pseudo-terminals. It sets terminal and WADE environment variables and exposes low-level process operations for reading, writing, resizing and closing.

It also resolves the configured shell or platform fallback, but does not own terminal resources, registries, buffering or WebSocket clients. Those lifecycle and concurrency concerns remain in the `terminals` Model.
