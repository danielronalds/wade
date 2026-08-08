# Slice 3: Settings and bootstrap

## Status

Complete.

## Goal

Migrate persisted settings, runtime configuration resolution, HTTP runtime
reconfiguration, CLI configuration, and server bootstrap to the Settings Model.
Remove the remaining config service and settings repository code.

Read [`PLAN.md`](PLAN.md) and the Settings and API conventions sections of
[`API_ENDPOINT_PROPOSAL.md`](API_ENDPOINT_PROPOSAL.md) before implementing.

## Responsibility migration map

| Current location | Final responsibility |
| --- | --- |
| `internal/repositories/settings.go` file IO | `infrastructure/filesystem/settings_file.go` |
| Settings defaults, parsing, compatibility, preservation, and validation | `models/settings` |
| `internal/repositories/settings_repository.go` | Removed |
| `internal/services/config` persisted settings behaviour | `models/settings` |
| `internal/services/config` runtime resolution | `models/settings.RuntimeConfiguration` |
| `internal/app/config_reload.go` orchestration | Settings HTTP controller |
| `cmd/wade/internal/controllers/config` settings access | Injected Settings Model interface |
| Server startup config loading | Shared Settings Model constructed from `cmd/wade/main.go` |

## Implementation checklist

### Infrastructure and Model

- [x] Add environment infrastructure for home directory, environment variables,
      and inherited shell values.
- [x] Add settings file location, existence, read, and direct-write behaviour to
      filesystem infrastructure.
- [x] Preserve straightforward direct writes without temporary-file rename
      machinery.
- [x] Create the Settings Model with `EnsureFile`, `Get`,
      `LoadRuntimeConfiguration`, `Update`, and `Reload`.
- [x] Define detached `Settings`, `Agent`, `RuntimeConfiguration`, and
      `UpdateResult` types.
- [x] Preserve defaults, legacy `projectDirectories`, unknown keys, validation,
      normalisation, and environment precedence.
- [x] Keep Settings independent from all other Model packages.
- [x] Serialise Settings persistence mutations inside the Model.

### HTTP orchestration

- [x] Decode and serialise the Model-owned Settings resource directly.
- [x] Inject Workspaces, Repositories, and Terminals Model interfaces into the
      Settings controller.
- [x] Hold one controller orchestration mutex across Update or Reload and all
      runtime Configure calls.
- [x] Store and call the Settings controller by pointer so its mutex is never
      copied.
- [x] Map neutral runtime values into each aggregate's Configuration type.
- [x] Remove `runtimeConfigApplier` after controller orchestration is active.

### CLI and startup

- [x] Construct environment, filesystem settings access, and one Settings Model
      in `cmd/wade/main.go`.
- [x] Pass the same Settings Model through the CLI router.
- [x] Make `wade config` consume an injected Settings Model interface.
- [x] Keep editor selection, process IO, and CLI presentation in the CLI
      controller.
- [x] Load server runtime configuration through the Settings Model before
      opening the listener.
- [x] Pass the same Settings Model and resolved configuration into
      `internal/app`.

### Tests and deletion

- [x] Move settings domain tests into `models/settings`.
- [x] Move mechanical settings-file tests into `infrastructure/filesystem`.
- [x] Update CLI and HTTP controller tests with Settings Model fakes.
- [x] Add a concurrent controller orchestration test that prevents stale runtime
      settings from winning after a newer persisted update.
- [x] Delete `internal/services/config`.
- [x] Delete remaining settings files from `internal/repositories`.
- [x] Confirm no compatibility adapter remains.

## Acceptance criteria

- Settings API, defaults, validation, and file compatibility are unchanged.
- `wade config`, foreground server, managed server, status, and stop continue to
  behave as before.
- Concurrent Update and Reload requests cannot leave disk and runtime settings
  out of order.
- No Settings or bootstrap code imports old services or repositories.

## Validation

```sh
mise run test
mise run lint:openapi
mise run lint:fmt
mise run lint:vet
```

Run the documented isolated lifecycle smoke test after this slice.

## Handoff

- Last completed: Migrated persisted settings and runtime resolution into the
  Settings Model; added environment and settings-file infrastructure; rewired
  HTTP, CLI, server bootstrap, and application composition; and removed the
  legacy config Service and settings repositories.
- Next action: Review `04-final-cleanup.md` before beginning final cleanup.
- Current failures: None.
- Last validation: `mise run test`, `mise run lint:openapi`,
  `mise run lint:fmt`, and `mise run lint:vet` passed. The focused Settings
  Model and controller race tests passed, and the isolated managed and
  foreground lifecycle smoke test passed.
- Important context: The Settings controller is stored by pointer and holds its
  orchestration mutex across persistence and all runtime Model configuration,
  while the Settings Model independently serialises file operations.
