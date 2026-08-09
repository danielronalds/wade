# Backend Domain and API Conventions

WADE's backend API is organised around explicit domain resources rather than controller operations. This document records the identity and relationship rules behind those resources. The generated OpenAPI specification remains authoritative for routes, operation IDs and request and response schemas.

The core domain resources are:

- [`Workspace`](#workspace)
- [`Repository`](#repository)
- [`RemoteRepository`](#remoterepository)
- [`Worktree`](#worktree)
- [`Branch`](#branch)
- [`Terminal`](#terminal)
- [`ReviewSnapshot`](#reviewsnapshot)
- [`Settings`](#settings)

## Workspace

A `Workspace` is one local directory that can be opened in WADE. It may be a non-Git directory, a repository's main worktree or a linked Git worktree. Terminals and review snapshots belong to a workspace because they operate within that exact directory.

A workspace ID is the directory's basename. Configured workspace directories are searched in order; when multiple directories contain the same basename, the first workspace is retained and later duplicates are skipped. Canonical paths are the internal source of path identity and comparison, but are not exposed as resource identifiers.

Git relationships are optional. A non-Git workspace has no repository, remote repository, worktree or branch values. A workspace is considered active when it has at least one running terminal.

## Repository

A `Repository` represents one local Git repository shared by its main and linked worktrees. Its ID is the basename of the main worktree directory, while the canonical Git common directory is the internal source of repository identity.

All linked worktrees share their local repository ID. Independent clones remain separate local repositories even when they point to the same remote repository. Branches and worktrees belong to this local repository aggregate.

## RemoteRepository

A `RemoteRepository` represents a repository available from an external provider. Its ID is GitHub's `nameWithOwner` value, such as `example/wade`.

Local matching uses canonical Git remote URLs rather than directory or repository basenames. Several independent local repositories and workspaces may therefore relate to the same remote repository.

## Worktree

A `Worktree` represents a Git worktree belonging to a local repository. Its directory is also a WADE workspace, so its worktree ID and workspace ID are both the directory basename.

Every Git workspace has a worktree relationship, including the main worktree. Main worktrees are not removable through WADE, while linked worktrees may be removable when repository invariants allow it.

## Branch

A `Branch` is represented by its full Git reference and structured metadata rather than a display name alone. Repository branch resources can identify their remote, whether a local branch exists and which workspace currently has the branch checked out.

Workspace branch values also capture detached state and the current commit. Full references keep local and remote branch identities unambiguous across worktree operations.

## Terminal

A `Terminal` represents one running PTY process scoped to a workspace. Terminal IDs are natural identifiers that describe their role, such as `misc`, `server`, `scratchpad` or `agent:pi`.

There is no separate project-session resource. A workspace's active state is derived from its terminals, and each live WebSocket connection targets an existing terminal by its workspace and terminal IDs.

## ReviewSnapshot

A `ReviewSnapshot` is a point-in-time description of files and comparisons in one workspace. Snapshot IDs are application-wide, while file IDs only need to be unique within their snapshot.

Snapshots retain pinned Git revisions and captured file identity until they are deleted or the server restarts. The `current` file-content scope is the deliberate exception and reads current filesystem contents.

## Settings

`Settings` is the persisted editable configuration for workspace directories, agents, worktree behaviour and interface preferences. Workspace materialisation requests identify one of the configured workspace directory strings, which the backend validates before cloning.

Configured values may resolve into runtime paths and commands, but the persisted representation remains suitable for editing and round-tripping. Runtime configuration is derived from settings rather than exposed as a separate API resource.

## API Conventions

Resource identifiers never use array indexes or absolute filesystem paths. Collection responses use an `{ "items": [...] }` object, while single-resource operations return the resource directly.

Creation returns the created resource with `201 Created` and a `Location` header. Successful deletion returns `204 No Content`. Optional relationships are represented explicitly as `null` rather than omitted or inferred.

Errors use `application/problem+json` with stable machine-readable problem codes. Controllers map typed domain errors to the appropriate status and problem response, while unexpected failures use an internal error response.

The generated OpenAPI specification and frontend client define the concrete HTTP contract. This document records the domain semantics that are not apparent from schemas alone.
