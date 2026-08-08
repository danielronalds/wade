# Slice 3: Settings and bootstrap

## Status

Planned. Depends on Slices 1 and 2.

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

- [ ] Add environment infrastructure for home directory, environment variables,
      and inherited shell values.
- [ ] Add settings file location, existence, read, and direct-write behaviour to
      filesystem infrastructure.
- [ ] Preserve straightforward direct writes without temporary-file rename
      machinery.
- [ ] Create the Settings Model with `EnsureFile`, `Get`,
      `LoadRuntimeConfiguration`, `Update`, and `Reload`.
- [ ] Define detached `Settings`, `Agent`, `RuntimeConfiguration`, and
      `UpdateResult` types.
- [ ] Preserve defaults, legacy `projectDirectories`, unknown keys, validation,
      normalisation, and environment precedence.
- [ ] Keep Settings independent from all other Model packages.
- [ ] Serialise Settings persistence mutations inside the Model.

### HTTP orchestration

- [ ] Decode and serialise the Model-owned Settings resource directly.
- [ ] Inject Workspaces, Repositories, and Terminals Model interfaces into the
      Settings controller.
- [ ] Hold one controller orchestration mutex across Update or Reload and all
      runtime Configure calls.
- [ ] Store and call the Settings controller by pointer so its mutex is never
      copied.
- [ ] Map neutral runtime values into each aggregate's Configuration type.
- [ ] Remove `runtimeConfigApplier` after controller orchestration is active.

### CLI and startup

- [ ] Construct environment, filesystem settings access, and one Settings Model
      in `cmd/wade/main.go`.
- [ ] Pass the same Settings Model through the CLI router.
- [ ] Make `wade config` consume an injected Settings Model interface.
- [ ] Keep editor selection, process IO, and CLI presentation in the CLI
      controller.
- [ ] Load server runtime configuration through the Settings Model before
      opening the listener.
- [ ] Pass the same Settings Model and resolved configuration into
      `internal/app`.

### Tests and deletion

- [ ] Move settings domain tests into `models/settings`.
- [ ] Move mechanical settings-file tests into `infrastructure/filesystem`.
- [ ] Update CLI and HTTP controller tests with Settings Model fakes.
- [ ] Add a concurrent controller orchestration test that prevents stale runtime
      settings from winning after a newer persisted update.
- [ ] Delete `internal/services/config`.
- [ ] Delete remaining settings files from `internal/repositories`.
- [ ] Confirm no compatibility adapter remains.

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

- Last completed: Not started.
- Next action: Wait for Slice 2 acceptance criteria.
- Current failures: None.
- Last validation: Not run for this slice.
- Important context: The Settings controller owns cross-Model runtime
  orchestration, while the Settings Model owns validation, persistence, and
  resolution.
