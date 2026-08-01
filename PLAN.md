# API refactor implementation plan

## Agreed implementation decisions

- `workspaceId` is the directory basename.
- Duplicate workspace names retain the first discovered directory and silently skip later matches.
- `repositoryId` is the main Git worktree directory basename.
- Canonical paths and Git common directories remain the internal source of identity.
- `remoteRepositoryId` is GitHub's `nameWithOwner`.
- Frontend workspace routes use `/workspaces/:workspaceId`.
- Clone requests use the configured workspace-directory string directly.
- Agent names are used directly rather than introducing agent IDs.
- Terminals use workspace-scoped natural IDs such as `misc`, `server`, and `agent:pi`.
- `PUT` idempotently starts or returns a terminal.
- Restarting uses DELETE followed by PUT.
- Review snapshots remain in memory until deleted or the server restarts.
- The API will use a hard cutover with no legacy endpoint compatibility.

`API_ENDPOINT_PROPOSAL.md` is the authoritative API contract for this plan.

## Implementation steps

### 1. Introduce the core workspace domain

Use `git mv` to evolve `internal/services/projects` into `internal/services/workspaces`, following the existing service-package conventions:

```text
internal/services/workspaces/
  service.go
  types.go
  metadata.go
  validators.go
  errors.go
```

Define:

- `Workspace`
- `WorkspaceSummary`
- `Branch`
- Structured workspace links
- Workspace activity
- Typed not-found and invalid-ID errors

Update the consuming repository interface to work with workspace records rather than project-name strings.

Keep orchestration and domain behaviour in the service. Keep filesystem access in `internal/repositories`.

### 2. Refactor workspace discovery and resolution

Refactor `internal/repositories/projects.go` and `project_directories.go` into workspace-oriented files using `git mv`.

Implement:

- Discovery of direct child directories under configured workspace directories.
- Workspace IDs based on directory basenames.
- Deterministic first-match behaviour based on configured directory order.
- Silent skipping of duplicate workspace IDs.
- `Path(workspaceID)` resolution.
- `IDForDirectory(path)` for mapping active terminal directories back to workspaces.
- Canonical path comparison for symlinks and terminal activity.
- Reloading when workspace-directory settings change.

Update repository tests to describe workspace behaviour and explicitly cover duplicate skipping.

### 3. Add local repository identity

Extend the Git repository adapter in `internal/repositories/git.go` with operations for:

- Resolving the main worktree path.
- Resolving the canonical Git common directory.
- Detecting non-Git workspaces.
- Resolving the current branch, commit, and detached state.

Create a cohesive local repository service package, such as:

```text
internal/services/gitrepositories/
  service.go
  types.go
  errors.go
```

The service should:

- Resolve a workspace to its local repository.
- Derive `repositoryId` from the main worktree directory basename.
- Use the canonical Git common directory internally to verify repository identity.
- Return all discovered workspaces belonging to the same worktree group.
- Allow independent clones of the same remote to have different local repository IDs.
- Return `repositoryId: null` for non-Git workspaces.

Repository interfaces should remain in this consuming service package.

### 4. Reshape worktree and branch behaviour

Retain `internal/services/worktrees`, but change its public API to accept resolved repository context instead of a project path.

Implement:

```http
GET    /api/v1/repositories/{repositoryId}/worktrees
POST   /api/v1/repositories/{repositoryId}/worktrees
DELETE /api/v1/repositories/{repositoryId}/worktrees/{worktreeId}
GET    /api/v1/repositories/{repositoryId}/branches
```

Changes include:

- Use the worktree directory basename as both `worktreeId` and resulting `workspaceId`.
- Return `isMain` rather than `isBase`.
- Return structured branch representations.
- Resolve repository IDs to canonical main worktree paths before Git operations.
- Return the newly discoverable workspace when creating a worktree.
- Close terminals belonging to a worktree before removing it.
- Preserve ignored-file copying and warnings.
- Add tests for creating and removing worktrees through repository identity.

### 5. Convert remote projects to remote repositories

Use `git mv` to rename:

```text
internal/services/remoteprojects
```

to:

```text
internal/services/remoterepositories
```

Update the model and behaviour:

- Use `nameWithOwner` as `remoteRepositoryId`.
- Keep the endpoint at `GET /api/v1/remote-repositories`.
- Do not add provider filtering yet.
- Determine local status by comparing canonical Git remote identities rather than directory basenames.
- Return matching local workspace IDs.
- Clone through `POST /api/v1/workspaces`.
- Accept `remoteRepositoryId` and `workspaceDirectory`.
- Verify that `workspaceDirectory` exactly matches a configured directory.
- Return the created `Workspace` with `201 Created`.

### 6. Refactor terminal management around the Terminal model

Use `git mv` to rename `internal/services/terminalsessions` to `internal/services/terminals`.

Rename `ProjectSession` to `Terminal` and preserve the existing PTY, replay buffer, attachment, resize, and activation behaviour.

Define and validate natural terminal IDs:

```text
misc
server
scratchpad
agent:pi
agent:claude
```

Implement:

```http
GET    /api/v1/workspaces/{workspaceId}/terminals
DELETE /api/v1/workspaces/{workspaceId}/terminals
PUT    /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
GET    /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
DELETE /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
POST   /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input
GET    /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket
```

Behaviour:

- PUT returns `201` when starting a PTY and `200` when it already exists.
- Concurrent PUT requests resolve to one terminal.
- DELETE synchronously closes the PTY, detaches clients, and removes the terminal before returning.
- The socket endpoint attaches only and returns `404` when the terminal does not exist.
- Input targets an exact terminal and supports bracketed-paste mode.
- Agent terminals resolve the configured agent by case-insensitive name.
- Terminal process environments include `WADE_WORKSPACE_ID`, `WADE_TERMINAL_ID`, and `WADE_ADDR`.
- Restart remains DELETE, PUT, then WebSocket reconnect.

Delete the existing `internal/services/sessions` abstraction once active workspaces and terminal input use the new model.

### 7. Implement explicit review snapshots

Retain `internal/services/review`, but replace the controller-owned per-path cache with a service-owned snapshot registry.

Implement:

```http
POST   /api/v1/workspaces/{workspaceId}/review-snapshots
GET    /api/v1/review-snapshots/{snapshotId}
GET    /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents
DELETE /api/v1/review-snapshots/{snapshotId}
```

The review service should:

- Generate an ephemeral snapshot ID.
- Store the workspace path, repository root, Git revisions, and file metadata internally.
- Return no absolute repository paths in the public response.
- Validate that requested file IDs belong to the snapshot.
- Rename scopes from `git-diff` to `working-tree` and `all-files` to `current`.
- Hold snapshots until explicit deletion or server shutdown.
- Return not found after a restart rather than implementing expiry semantics.
- Protect the registry with the same explicit concurrency style used by terminals.

Move snapshot tests into the service package and retain the existing Git integration coverage.

### 8. Consolidate settings behaviour

Rename `projectDirectories` to `workspaceDirectories` across:

- Persisted settings
- Runtime configuration
- Backend service types
- API payloads
- Frontend settings types and forms

Preserve existing user settings by reading `projectDirectories` as a legacy key. On the next save, write `workspaceDirectories` and remove the old key.

Implement:

```http
GET  /api/v1/settings
PUT  /api/v1/settings
POST /api/v1/settings/reload
```

Move update orchestration into the config service so PUT:

1. Normalises and validates the complete request.
2. Resolves runtime-safe values.
3. Persists the settings.
4. Reloads workspace discovery and terminal configuration.
5. Returns the normalised settings.

The frontend should no longer call update followed by reload. The reload endpoint remains for out-of-band file edits.

### 9. Add the new controllers and perform the route cutover

Keep controllers in the single `internal/controllers` package with separate files per resource:

```text
workspaces.go
repositories.go
remote_repositories.go
worktrees.go
terminals.go
review_snapshots.go
settings.go
```

Controllers should only:

- Parse path, query, and JSON input.
- Invoke services.
- Translate typed service errors to HTTP responses.
- Write API responses.

Update `internal/server/routes.go` in one hard cutover:

- Register only the new `/api/v1` application endpoints.
- Retain `/api/openapi.json` and `/api/docs/`.
- Remove old project, session, terminal, worktree, review, and config routes.
- Replace `/ws` with the nested terminal socket route.

Keep stable OpenAPI operation IDs and annotations on controllers.

### 10. Standardise errors and response conventions

Replace the current `{ "message": "..." }` error with an `application/problem+json` response containing:

```text
type
title
status
detail
code
```

Add one central response helper in `internal/controllers/response.go`.

Use consistently:

- `400` for malformed JSON or syntax.
- `404` for unknown resource IDs.
- `409` for state conflicts such as removing the main worktree.
- `422` for valid requests that fail domain validation.
- `500` for unexpected failures.

Use `{ "items": [...] }` for collections, direct objects for individual resources, `201` plus `Location` for creation, and `204` for deletion.

Update the frontend HTTP helper to read `detail` while retaining plain-text handling for failed WebSocket upgrades.

### 11. Regenerate the API contract

Run the established generation flow after controller annotations are complete:

```sh
mise run gen:openapi
npm --prefix web run gen:api
```

Review generated schema names to ensure they expose domain names such as:

- `Workspace`
- `Repository`
- `RemoteRepository`
- `Worktree`
- `Terminal`
- `ReviewSnapshot`
- `Settings`

Do not add handwritten endpoint wrappers. Import generated Orval functions directly, following the existing frontend convention.

### 12. Rename frontend project concepts to workspace concepts

Use `git mv` for route and feature renames without decomposing components unnecessarily:

```text
web/src/views/project        -> web/src/views/workspace
web/src/features/projects    -> web/src/features/workspaces
```

Rename related stores, types, composables, props, events, and local variables.

Update Vue Router:

```text
/workspaces/:workspaceId
```

Rename the route to `workspace` and pass `workspaceId` to `WorkspaceView`.

Update user-facing language:

- Project picker to workspace picker.
- Active sessions to active workspaces.
- Project topbar to workspace topbar.
- Recent projects to recent workspaces.
- Remote projects to remote repositories.

### 13. Migrate frontend identity and stored state

Change frontend collections from strings to generated `Workspace` summaries.

Update:

- Fuzzy matching to use workspace names while actions use workspace IDs.
- Recent-workspace storage to store workspace IDs.
- Active-workspace queries to call `GET /workspaces?activity=active`.
- Project-detail state to workspace-detail state keyed by workspace ID.
- Selected-agent storage to use workspace IDs.
- Worktree navigation to use returned workspace IDs.
- Clone navigation to use the created workspace ID.

Because existing project names become workspace IDs, migrate existing local-storage arrays and selected-agent keys without needing path translation.

### 14. Migrate terminal frontend behaviour

Update the terminal composables to:

1. Construct the natural terminal ID.
2. PUT the terminal before connecting.
3. Attach through its nested socket URL.
4. Use DELETE followed by PUT for restart.
5. Send direct input through the terminal endpoint when HTTP input is needed.
6. List terminals when showing active workspace state.
7. Continue preserving replay, resize, Escape handling, WebGL setup, and focus behaviour.

Keep terminal implementation in the existing cross-route terminal feature area.

### 15. Migrate worktree, remote repository, review, and settings workflows

Update generated API usage in:

- Worktree command palettes
- Remote repository palette
- Review tab
- General command palette
- Settings store and settings view

Specific changes:

- Worktree palettes read `repositoryId` from the current workspace.
- Branch and worktree calls use repository routes.
- Worktree creation and removal navigate by returned `workspaceId`.
- Remote cloning sends `remoteRepositoryId` and `workspaceDirectory`.
- Review state stores `snapshotId` and uses it for every file-content request.
- Cancelling a review deletes its snapshot.
- Settings save uses one PUT and consumes the returned normalised settings.

Do not introduce new UI components solely for this structural migration.

### 16. Remove obsolete code and deepen the new domain modules

After the frontend uses the new contract:

- Remove old controllers and request/response types.
- Remove `internal/services/sessions`.
- Remove project-name path resolution helpers.
- Remove controller-owned review caching.
- Remove implicit terminal creation from WebSocket attachment.
- Remove duplicate handwritten frontend API types where generated types suffice.
- Remove obsolete local-storage composables after migration.

Keep the required Controllers, Services, Repositories architecture. Make the service packages deeper and cohesive rather than moving Git, filesystem, process, or GitHub IO into domain structs.

### 17. Add and update tests

Follow the existing Go test style with table-driven tests and local stubs.

Cover:

- Workspace discovery, resolution, and duplicate skipping.
- Main-worktree repository ID resolution.
- Independent clone identity.
- Remote repository matching and cloning.
- Repository-scoped branch and worktree operations.
- Natural terminal ID parsing and validation.
- Idempotent terminal PUT and concurrent creation.
- Terminal deletion before recreation.
- Exact terminal input targeting.
- WebSocket attachment to existing terminals.
- Snapshot creation, lookup, file loading, deletion, and restart semantics.
- Settings legacy-key migration and atomic application.
- Controller status codes and problem responses.
- Route registration for the complete `/api/v1` surface.

There is currently no frontend unit-test harness, so use the existing `vue-tsc` and build validation rather than introducing one solely for this refactor.

### 18. Validate the complete cutover

Run the project-standard checks:

```sh
mise run fmt
mise run lint:fmt
mise run lint:openapi
mise run test
```

Then perform the documented smoke test on a temporary port:

- Load the home, settings, and workspace routes.
- Open all terminal roles.
- Restart a terminal.
- Reconnect after navigating away.
- Create and remove a worktree.
- Clone a remote repository.
- Start, browse, and cancel a review snapshot.
- Save settings and reload out-of-band changes.
- Verify duplicate workspace names retain the first configured match.
- Confirm no old API routes or `/ws` remain available.
- Confirm no stale WADE processes remain on the test port.

No implementation changes should begin until this plan is approved.
