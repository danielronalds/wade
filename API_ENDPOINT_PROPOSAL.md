# API endpoint proposal

## Purpose

This proposal reorganises the WADE API around explicit domain resources rather
than controller operations. It is intentionally a breaking redesign and uses an
`/api/v1` prefix.

The primary resource is a **Workspace**: an openable local working directory in
WADE. The term `Workspace` should be used consistently by the frontend,
backend, API, routes, and user-facing product language. A directory may be a
non-Git directory, a Git repository's main worktree, or a linked Git worktree.

The central identity rule is:

> A workspace ID is the workspace directory's basename.

Configured workspace directories are searched in order. If more than one
workspace has the same basename, the first discovered workspace is retained and
later duplicates are silently skipped. Canonical paths remain the internal
source of identity and path comparison.

## Domain models

### Workspace

An openable local working directory and its optional Git relationships.

```text
Workspace
  id
  name
  repositoryId: string | null
  remoteRepositoryId: string | null
  worktree: WorktreeReference | null
    id
    isMain
    isRemovable
  branch: Branch | null
    ref
    name
    isDetached
    commit
  links
    repository
    pullRequest
    issue
  activity
    activeTerminalCount
```

A non-Git workspace has `null` repository, remote repository, worktree, and
branch values. Every Git working directory, including the main worktree, has a
worktree reference.

Workspace links should use structured values rather than provider-specific
top-level fields. An issue reference can contain `provider`, `key`, and `url`,
rather than only a `linearTicketUrl` field.

Terminals and review snapshots belong to a workspace because they operate
within one exact directory.

### Repository

A local Git repository shared by its main and linked worktrees.

```text
Repository
  id
  remoteRepositoryId: string | null
  mainWorkspaceId
  workspaceIds
```

A repository ID is the basename of its main Git worktree directory. Linked Git
worktrees therefore share a `repositoryId`. Independent clones have different
`repositoryId` values when their main worktree directory names differ because
their Git state and worktree sets are independent. If those clones point to the
same external repository, they share a `remoteRepositoryId`.

Canonical Git common directories remain the internal source of repository
identity. Branches and worktrees belong to this local repository model.

### RemoteRepository

A repository available from an external provider and potentially materialised
as a workspace.

```text
RemoteRepository
  id
  name
  webUrl
  cloneUrl
  localWorkspaceIds
```

The remote repository ID is GitHub's `nameWithOwner`, such as `example/wade`.
Local matching should compare canonical Git remotes rather than repository
basenames.

### Worktree

A Git worktree belonging to the repository associated with a workspace.

```text
Worktree
  id
  repositoryId
  workspaceId
  name
  branch
  isMain
  isRemovable
```

A worktree's local directory is also a `Workspace`. Its `worktreeId` and
`workspaceId` are both the worktree directory basename. Worktree responses
should therefore reference the resulting `workspaceId`.

### Branch

A structured Git branch reference.

```text
Branch
  ref
  name
  remote
  hasLocalBranch
  checkedOutWorkspaceId
```

### Terminal

One running PTY process.

```text
Terminal
  id
  workspaceId
  role: agent | misc | server | scratchpad
  agent: string | null
  status
  socketUrl
```

There is no separate project-session resource. A workspace is active when it
has one or more terminals.

### ReviewSnapshot

A point-in-time description of files and comparisons in a workspace.

```text
ReviewSnapshot
  id
  workspaceId
  branch
  pullRequest
  files
  createdAt
```

File IDs only need to be unique within their snapshot.

### Settings

The persisted editable configuration includes:

- Configured workspace directory strings
- Configured agents identified by their unique names
- Worktree preferences
- Theme preferences

Clone requests use a configured workspace directory string directly. The
backend verifies that the requested directory is configured before cloning.

## Endpoints

### Workspaces

```http
GET  /api/v1/workspaces
GET  /api/v1/workspaces/{workspaceId}
POST /api/v1/workspaces
```

The frontend opens a workspace at `/workspaces/{workspaceId}`.

#### List workspaces

```http
GET /api/v1/workspaces
```

Returns workspace summaries rather than project name strings.

Supported filters can include:

```text
?activity=active
?repositoryId={repositoryId}
?remoteRepositoryId={remoteRepositoryId}
```

The active-workspace picker uses:

```http
GET /api/v1/workspaces?activity=active
```

Example response:

```json
{
  "items": [
    {
      "id": "wade",
      "name": "wade",
      "repositoryId": "wade",
      "remoteRepositoryId": "example/wade",
      "worktree": {
        "id": "wade",
        "isMain": true,
        "isRemovable": false
      },
      "branch": {
        "ref": "refs/heads/main",
        "name": "main",
        "isDetached": false,
        "commit": "abc123"
      },
      "links": {
        "repository": "https://github.com/example/wade",
        "pullRequest": null,
        "issue": null
      },
      "activity": {
        "activeTerminalCount": 2
      }
    }
  ]
}
```

#### Get a workspace

```http
GET /api/v1/workspaces/{workspaceId}
```

Returns the complete workspace representation.

#### Materialise a remote repository

```http
POST /api/v1/workspaces
```

Example request:

```json
{
  "remoteRepositoryId": "example/wade",
  "workspaceDirectory": "~/Personal"
}
```

Returns `201 Created` with the resulting `Workspace` and a `Location` header.
This replaces the dedicated remote-project clone command.

### Remote repositories

```http
GET /api/v1/remote-repositories
```

Example response:

```json
{
  "items": [
    {
      "id": "example/wade",
      "name": "wade",
      "webUrl": "https://github.com/example/wade",
      "cloneUrl": "git@github.com:example/wade.git",
      "localWorkspaceIds": ["wade"]
    }
  ]
}
```

A remote repository is not itself a WADE workspace or local repository.

### Repositories

```http
GET /api/v1/repositories/{repositoryId}
```

Returns the local Git repository and the workspaces that share it.

Example response:

```json
{
  "id": "wade",
  "remoteRepositoryId": "example/wade",
  "mainWorkspaceId": "wade",
  "workspaceIds": ["wade", "wade-feature"]
}
```

Two independent clones of the same remote repository return different local
repository IDs but the same remote repository ID.

### Worktrees

```http
GET    /api/v1/repositories/{repositoryId}/worktrees
POST   /api/v1/repositories/{repositoryId}/worktrees
DELETE /api/v1/repositories/{repositoryId}/worktrees/{worktreeId}
```

Worktrees are owned by a local repository. Each worktree references the
workspace representing its working directory.

#### List worktrees

```http
GET /api/v1/repositories/{repositoryId}/worktrees
```

Example response:

```json
{
  "items": [
    {
      "id": "wade",
      "repositoryId": "wade",
      "workspaceId": "wade",
      "name": "wade",
      "branch": {
        "ref": "refs/heads/main",
        "name": "main",
        "remote": null
      },
      "isMain": true,
      "isRemovable": false
    }
  ]
}
```

#### Create a worktree

```http
POST /api/v1/repositories/{repositoryId}/worktrees
```

Example request:

```json
{
  "branchRef": "refs/remotes/origin/feature/example"
}
```

Returns `201 Created` with the new `Worktree`. The response includes the
`workspaceId` of the newly openable workspace.

#### Remove a worktree

```http
DELETE /api/v1/repositories/{repositoryId}/worktrees/{worktreeId}
```

Returns `204 No Content`. Removal uses one unambiguous worktree ID and does not
accept a request body, filesystem path, branch, or display name.

### Branches

```http
GET /api/v1/repositories/{repositoryId}/branches
```

Supported filters can include:

```text
?kind=remote
?kind=local
```

Example response:

```json
{
  "items": [
    {
      "ref": "refs/remotes/origin/feature/example",
      "name": "feature/example",
      "remote": "origin",
      "hasLocalBranch": false,
      "checkedOutWorkspaceId": null
    }
  ]
}
```

### Terminals

```http
GET    /api/v1/workspaces/{workspaceId}/terminals
DELETE /api/v1/workspaces/{workspaceId}/terminals

PUT    /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
GET    /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
DELETE /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
POST   /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input
GET    /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket
```

Terminal IDs are natural identifiers scoped to a workspace:

```text
misc
server
scratchpad
agent:pi
agent:claude
```

Agent terminal IDs use the configured agent name case-insensitively. Terminal
representations return the configured display name.

#### List a workspace's terminals

```http
GET /api/v1/workspaces/{workspaceId}/terminals
```

Example response:

```json
{
  "items": [
    {
      "id": "agent:pi",
      "workspaceId": "wade",
      "role": "agent",
      "agent": "Pi",
      "status": "running",
      "socketUrl": "/api/v1/workspaces/wade/terminals/agent:pi/socket"
    }
  ]
}
```

#### Start or reconnect to a terminal

```http
PUT /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
```

The request has no body because the terminal ID identifies its role and
configured agent. The operation is idempotent:

- If the terminal does not exist, start it and return `201 Created`.
- If the terminal is already running, return it with `200 OK`.
- Concurrent PUT requests resolve to the same PTY.

#### Close all terminals for a workspace

```http
DELETE /api/v1/workspaces/{workspaceId}/terminals
```

Returns `204 No Content`. This replaces deletion of a project session.

#### Get a terminal

```http
GET /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
```

Returns the exact terminal resource.

#### Close a terminal

```http
DELETE /api/v1/workspaces/{workspaceId}/terminals/{terminalId}
```

Returns `204 No Content`. The terminal is removed from the registry before the
response is returned.

Reloading a terminal is represented by closing and recreating it:

1. Delete the existing terminal.
2. PUT a replacement terminal using the same natural ID.
3. Connect to the replacement's `socketUrl`.

#### Send terminal input

```http
POST /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input
```

Example request:

```json
{
  "text": "Review the current changes",
  "mode": "bracketed-paste"
}
```

Returns `204 No Content`. The request targets an exact terminal rather than
searching for an implicitly selected agent terminal.

An agent process receives its workspace and terminal IDs through the
environment so scripts can target the correct terminal directly.

#### Connect to a terminal

```http
GET /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket
```

This upgrades to a WebSocket connection. The terminal process must already
exist, so the endpoint returns `404 Not Found` rather than creating a terminal
implicitly.

Binary messages contain terminal input and output. Text messages contain JSON
control messages such as resize and activation events.

### Review snapshots

```http
POST   /api/v1/workspaces/{workspaceId}/review-snapshots
GET    /api/v1/review-snapshots/{snapshotId}
GET    /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents
DELETE /api/v1/review-snapshots/{snapshotId}
```

#### Create a review snapshot

```http
POST /api/v1/workspaces/{workspaceId}/review-snapshots
```

Creating a snapshot captures the workspace's current Git and filesystem state.
It returns `201 Created`, the complete snapshot, and a `Location` header.

Example response:

```json
{
  "id": "review_snapshot_01",
  "workspaceId": "wade",
  "branch": {
    "ref": "refs/heads/feature/example",
    "name": "feature/example",
    "remote": null
  },
  "pullRequest": {
    "number": 123,
    "url": "https://github.com/example/wade/pull/123",
    "baseRef": "refs/heads/main",
    "headRef": "refs/heads/feature/example"
  },
  "files": [],
  "createdAt": "2026-08-01T06:30:00Z"
}
```

#### Get a review snapshot

```http
GET /api/v1/review-snapshots/{snapshotId}
```

Returns the same point-in-time representation originally created.

#### Get review file contents

```http
GET /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents?scope={scope}
```

Supported scopes:

```text
pull-request
working-tree
last-commit
current
```

`working-tree` replaces the less precise `git-diff` name. `current` replaces
`all-files` when requesting the current file content.

Example response:

```json
{
  "originalContent": "...",
  "modifiedContent": "..."
}
```

The snapshot ID guarantees that contents use the same file identity and Git
revisions as the snapshot's file list.

#### Delete a review snapshot

```http
DELETE /api/v1/review-snapshots/{snapshotId}
```

Returns `204 No Content`. Snapshots are held in memory until they are deleted
or the server restarts.

### Settings

```http
GET  /api/v1/settings
PUT  /api/v1/settings
POST /api/v1/settings/reload
```

#### Get settings

```http
GET /api/v1/settings
```

Example response:

```json
{
  "workspaceDirectories": ["~/Personal", "~/Work"],
  "shell": "",
  "agents": [
    {
      "name": "Pi",
      "command": "pi -c",
      "default": true
    }
  ],
  "copyIgnoredFilesOnWorktreeCreation": false,
  "openWorktreesInNewTabs": false,
  "worktreeCopyExcludes": [],
  "themeAccentColor": "white"
}
```

The settings loader reads the legacy `projectDirectories` key. Saving settings
writes `workspaceDirectories` and removes the legacy key.

#### Update settings

```http
PUT /api/v1/settings
```

The request contains the complete settings representation. The server
validates, persists, and applies runtime-safe settings changes atomically.

Returns `200 OK` with the normalised settings that were saved and applied. The
frontend does not need to make a separate reload request after an update.

#### Reload settings

```http
POST /api/v1/settings/reload
```

Reloads out-of-band edits from the settings file and applies runtime-safe
changes. Returns `200 OK` with the loaded settings.

## Current endpoint mapping

| Current endpoint | Proposed endpoint |
| --- | --- |
| `GET /api/projects` | `GET /api/v1/workspaces` |
| `GET /api/project?project={name}` | `GET /api/v1/workspaces/{workspaceId}` |
| `GET /api/remote-projects` | `GET /api/v1/remote-repositories` |
| `POST /api/remote-projects/clone` | `POST /api/v1/workspaces` |
| `GET /api/worktrees?project={name}` | `GET /api/v1/repositories/{repositoryId}/worktrees` |
| `POST /api/worktrees` | `POST /api/v1/repositories/{repositoryId}/worktrees` |
| `DELETE /api/worktrees` | `DELETE /api/v1/repositories/{repositoryId}/worktrees/{worktreeId}` |
| `GET /api/worktrees/remote-branches` | `GET /api/v1/repositories/{repositoryId}/branches?kind=remote` |
| `GET /api/sessions` | `GET /api/v1/workspaces?activity=active` |
| `DELETE /api/sessions/{sessionName}` | `DELETE /api/v1/workspaces/{workspaceId}/terminals` |
| `POST /api/sessions/{projectName}/agent` | `POST /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input` |
| `POST /api/terminal/reload` | DELETE then PUT `/api/v1/workspaces/{workspaceId}/terminals/{terminalId}` |
| `GET /ws` | `GET /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket` |
| `GET /api/review` | `POST /api/v1/workspaces/{workspaceId}/review-snapshots` |
| `POST /api/review/file` | `GET /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents` |
| `GET /api/config` | `GET /api/v1/settings` |
| `POST /api/config` | `PUT /api/v1/settings` |
| `POST /api/config/reload` | `POST /api/v1/settings/reload` |

## API conventions

### Identifiers

- Workspace IDs are workspace directory basenames.
- Repository IDs are main Git worktree directory basenames.
- Worktree IDs are their workspace directory basenames.
- Remote repository IDs are GitHub `nameWithOwner` values.
- Terminal IDs are natural identifiers scoped to a workspace.
- Do not use array indexes or absolute filesystem paths as resource identifiers.
- Do not return absolute `path` or `repoRoot` values unless a client genuinely
  needs them.

### Responses

- Return `{ "items": [...] }` for collections.
- Return the resource directly for single-resource requests.
- Return the created resource and a `Location` header with `201 Created`.
- Return `204 No Content` for successful deletion.
- Use the same structured `Branch`, `PullRequest`, and issue-reference schemas
  wherever those concepts appear.

### Errors

Use `application/problem+json` with a stable machine-readable `code` extension.

Example:

```json
{
  "type": "https://wade.local/problems/worktree-not-removable",
  "title": "Worktree cannot be removed",
  "status": 409,
  "detail": "The main worktree cannot be removed.",
  "code": "worktree_not_removable"
}
```

Recommended status usage:

- `400 Bad Request` for malformed syntax or JSON.
- `404 Not Found` for unknown resource IDs.
- `409 Conflict` for uniqueness conflicts and resources that cannot be changed
  in their current state.
- `422 Unprocessable Content` for structurally valid requests that violate
  domain validation rules.
- `500 Internal Server Error` for unexpected failures.

### OpenAPI

- Give schemas domain names such as `Workspace`, `Worktree`, `Terminal`, and
  `ReviewSnapshot`.
- Do not expose implementation names such as `handlers.projectResponse`.
- Give every operation a stable operation ID.
- Generate and use frontend API models directly instead of maintaining
  duplicate handwritten representations of backend resources.

## Terminology decision

Use `Workspace` consistently throughout WADE rather than translating between a
frontend `Project` and a backend `Workspace`.

- A `Workspace` is one local directory opened by WADE.
- A `Repository` is one local Git repository and its worktree set.
- A `RemoteRepository` is an external source repository, such as a GitHub
  repository.
- A `Worktree` connects a Git working directory to both its local repository
  and its WADE workspace.

This vocabulary should be reflected in route names, stores, generated API
models, component props, command-palette language, and user-facing labels.
