# Plan: Configurable Linear Integration

## Objective

Replace WADE's hardcoded Linear workspace with an optional, runtime-configurable integration stored in the main settings resource.

The persisted shape is:

```json
"linear": {
  "enabled": true,
  "workspace": "signinsolutions"
}
```

No implementation has been started. This document records the agreed behaviour and a file-level implementation plan for a new session.

## Agreed behaviour

- New settings and settings files without a `linear` key default to:

  ```json
  "linear": {
    "enabled": false,
    "workspace": ""
  }
  ```

- Remove the existing hardcoded `signinsolutions` workspace. Do not migrate existing users to an enabled integration.
- Always trim surrounding whitespace from `linear.workspace`.
- Preserve the workspace when Linear is disabled so it can be re-enabled later.
- Only validate the workspace when Linear is enabled.
- An enabled integration requires a non-empty workspace slug.
- Valid slugs contain only RFC 3986 unreserved characters: `A-Z`, `a-z`, `0-9`, `-`, `.`, `_`, and `~`.
- Preserve the slug's case. Do not automatically lowercase it.
- Apply Linear configuration immediately after API settings updates and settings reloads. A server restart must not be required.
- Do not attempt Linear ticket resolution while the integration is disabled.
- Hide both the workspace topbar Ticket pill and the command palette's Open Issue command while disabled.
- When enabled but no branch ticket is found, retain the existing visible-but-disabled behaviour.
- Add a **Linear Integration** section after **Worktrees** and before the settings form actions.
- The section contains an enable checkbox and a workspace input. Disable the input when Linear is disabled.
- After Linear configuration changes, clear cached Linear issue links immediately so an old workspace URL cannot be opened.
- If a workspace is active and Linear remains enabled, refetch that workspace's details in the background. Show the existing loading-style disabled state while this happens.
- Inactive cached workspaces remain invalidated and are fetched normally when next opened.

## Architecture

Keep provider enablement and workspace selection in the `settings` and `workspaces` Models, not in infrastructure or frontend-only policy.

The current Linear infrastructure client stores an immutable workspace. Change it into a stateless provider helper that receives the workspace slug for each ticket-resolution call. This matches the documented infrastructure boundary and allows `workspaces.Model.Configure` to apply settings safely at runtime.

Suggested backend flow:

1. The settings Model parses, defaults, normalises, validates, persists, and resolves Linear settings into `RuntimeConfiguration`.
2. Application startup maps runtime Linear configuration into `workspaces.Configuration`.
3. The settings controller maps updated or reloaded runtime Linear configuration into the same `workspaces.Configuration`.
4. The workspaces Model snapshots its configuration under its existing mutex. It calls Linear infrastructure only when enabled, passing the configured workspace slug.
5. The frontend reads enablement from the app-wide settings store for visibility and invalidates cached issue links when the configuration changes.

## Implementation steps

### 1. Extend the settings domain and persistence

Update these files:

- `internal/models/settings/types.go`
- `internal/models/settings/persistence.go`
- `internal/models/settings/clones.go`
- `internal/models/settings/validators.go`
- `internal/models/settings/runtime.go`

Tasks:

- Add an exported nested settings value, preferably named `LinearSettings`, with `enabled` and `workspace` JSON fields and an explicit Swaggo schema name.
- Add `Linear LinearSettings \`json:"linear"\`` to `Settings`.
- Add neutral Linear fields to `RuntimeConfiguration`. Either a nested value or explicit enabled/workspace fields is acceptable, but keep mapping clear and avoid leaking provider infrastructure types into settings.
- Include disabled and empty Linear defaults in `defaultSettings()`.
- Add an optional nested pointer to `settingsFile` so a missing `linear` object receives defaults. A present partial object should receive normal Go zero values, meaning `{}` is disabled with an empty workspace.
- Parse configured values while preserving the existing top-level unknown-key behaviour.
- Always write the complete `linear` object from `encodeSettings`, including when disabled.
- Trim `workspace` during normalisation. Ensure API updates persist the trimmed value.
- Validate only when enabled using a full-string expression equivalent to `^[A-Za-z0-9._~-]+$`.
- Return useful validation errors for an empty enabled workspace and for unsupported characters.
- Ensure startup runtime resolution and out-of-band reload validation reject invalid enabled configurations as `InvalidSettingsError`.
- Carry the normalised Linear values into cloned settings, update results, and runtime configuration. The nested struct is currently value-only, but keep clone handling explicit if that changes.

Important persistence semantics:

- Missing `linear` means disabled, including existing user files.
- A disabled workspace may contain characters that would be invalid when enabled. Trim and preserve it, but do not block unrelated settings updates or server startup.
- Enabling that same value must surface validation until it is corrected.

### 2. Make Linear infrastructure stateless

Update:

- `internal/infrastructure/linear/client.go`
- Add `internal/infrastructure/linear/client_test.go`

Tasks:

- Remove the workspace field from `Client`.
- Change `NewClient` so it no longer accepts a hardcoded workspace.
- Change `TicketForBranch` to receive both the workspace and branch, with a clear argument order used consistently by its interface and callers.
- Keep defensive empty-workspace handling even though the settings Model validates enabled configuration.
- Preserve current ticket-key extraction and uppercase URL key behaviour.
- Continue returning `nil, nil` when the branch contains no ticket key.

Test URL construction, key normalisation, branches without tickets, and the defensive empty-workspace error.

### 3. Put Linear runtime policy in the workspaces Model

Update:

- `internal/models/workspaces/interfaces.go`
- `internal/models/workspaces/model.go`
- `internal/models/workspaces/model_test.go`

Tasks:

- Update the consumer-owned `Linear` interface to match the stateless client signature.
- Extend `workspaces.Configuration` with a cohesive Linear configuration value containing `Enabled` and `Workspace`.
- Include Linear configuration in `cloneConfiguration` and configuration snapshots.
- In `ResolveLinks`, read a consistent configuration snapshot under the existing `configurationMu`.
- Call Linear only when the snapshot says it is enabled, passing the snapshot workspace.
- Preserve best-effort enrichment. Provider errors still join the optional link errors, and a missing branch ticket still leaves `links.issue` null.
- Ensure `Configure` changes affect later `ResolveLinks` calls without reconstructing the Model.

Add a Linear stub and tests proving:

- Disabled configuration does not call the provider and returns no issue.
- Enabled configuration passes the configured workspace and maps the returned ticket.
- Reconfiguration changes subsequent resolution to the new workspace.
- Existing repository and pull request enrichment remains unaffected.

### 4. Wire startup and runtime reconfiguration

Update:

- `internal/app/app.go`
- `internal/app/configuration.go`
- `internal/controllers/settings.go`
- `internal/controllers/settings_test.go`
- Any affected controller fakes in `internal/controllers/model_fakes_test.go`

Tasks:

- Construct the stateless Linear client without `signinsolutions`.
- Map settings runtime Linear values into `workspaces.Configuration` during application startup.
- Map the same values in `Settings.applyRuntimeConfiguration` for both update and reload paths.
- Keep Linear changes within the settings controller's existing orchestration mutex so persistence and runtime application remain ordered.
- Extend controller tests to assert the Linear configuration reaches the workspaces Model.

Do not add a package-global client or mutable provider singleton.

### 5. Regenerate the API contract and update frontend settings helpers

After the Go schema changes, regenerate:

- `internal/openapi/swagger.json`
- `internal/openapi/swagger.yaml`
- `web/src/api/generated/wade.ts`

Use the existing generation tasks rather than hand-editing generated files.

Update:

- `web/src/types/settings.ts`
- `web/src/views/settings/composables/useSettingsForm.ts`

Tasks:

- Include the nested Linear value in `createEmptySettings`, `cloneSettings`, and `normaliseSettings`.
- Export the generated nested Linear type if useful to callers.
- Add a pure `isValidLinearWorkspace` helper matching backend validation.
- Extend change detection and form replacement to include both Linear fields.
- Add `hasInvalidLinearWorkspace`, which is false whenever Linear is disabled.
- Include Linear validation in `canSave`.
- Add form update methods for enablement and workspace input. Clear status/error messages consistently with existing controls.
- Ensure the workspace value is trimmed before save while preserving user-entered text during editing.

### 6. Add the Settings page section

Update:

- `web/src/views/settings/SettingsView.vue`

Tasks:

- Add **Linear Integration** after the Worktrees section.
- Use the existing shared `Checkbox` component for enablement.
- Add a labelled text input for the workspace URL slug.
- Disable the input when `form.linear.enabled` is false.
- Use helper copy that makes the expected value clear, for example that `signinsolutions` forms `linear.app/signinsolutions`.
- Disable spellcheck and autocomplete consistently with other technical settings inputs.
- Set `aria-invalid` only when enabled and invalid.
- Add a specific footer validation message before generic request errors.
- Extend the existing section and row CSS rather than introducing a route-only component for two controls.
- Include disabled-input styling and responsive behaviour consistent with the Shell row.

### 7. Hide disabled Linear actions

Update:

- `web/src/views/workspace/components/WorkspaceTopbar.vue`
- `web/src/features/command-palette/components/GeneralCommandPalette.vue`

Tasks:

- Read the reactive settings store rather than copying the enabled value.
- Wrap the complete Ticket action, including its copy button, in an enabled check.
- Conditionally omit the Open Issue command definition when disabled. Do not leave a disabled command labelled "No issue found" when resolution was intentionally skipped.
- Preserve current disabled and loading behaviour when enabled but no issue URL is available.
- Since the frontend default is disabled, both actions remain hidden until settings load and confirm enablement.

### 8. Invalidate stale issue links and background-refresh the active workspace

Update:

- `web/src/stores/useWorkspaceDetailsStore.ts`
- `web/src/stores/useSettingsStore.ts`
- `web/src/views/workspace/WorkspaceView.vue`
- Potentially the topbar and command palette loading computations listed above

Recommended implementation:

- Add a workspace-details-store action that replaces cached workspace snapshots with detached copies whose `links.issue` is `null`, while preserving repository, pull request, branch, and other details.
- In the settings store, compare the previous and replacement `linear.enabled` and `linear.workspace` values. Invalidate issue links only when that configuration changes.
- Add a background refresh action that guarantees a fresh request after any already-running request completes. The current `loadWorkspaceDetails` deduplication must not cause a pre-configuration request to be reused as the only refresh.
- In `WorkspaceView`, watch the reactive Linear configuration. When it changes to an enabled state, trigger the fresh workspace-details request for the active `workspaceId`. Do not block navigation or terminal interaction.
- When disabling Linear, invalidate but do not refetch solely for Linear because the actions are hidden and backend resolution is disabled.
- Keep inactive workspaces invalidated. Their existing mount-time load will fetch current details when opened.
- Treat an enabled issue refresh as loading even when the rest of a cached workspace snapshot remains present. The Ticket controls and Open Issue command should show loading, not "No ticket found", until the background request settles.
- Prevent a late response from an older in-flight request from restoring a stale issue URL. This can be handled with a refresh queue/generation in the workspace details store rather than duplicating requests in components.

Avoid clearing the entire workspace cache, since GitHub links, repository metadata, and workspace display data remain valid and should not flicker.

### 9. Documentation

Update:

- `README.md`
- `docs/backend-domain.md`
- `docs/backend-models.md`
- `docs/backend-infrastructure.md`
- `docs/backend-controllers.md` if needed to describe the expanded runtime reconfiguration

Tasks:

- Add the disabled Linear object to the example settings JSON.
- Explain that the workspace is the slug from `linear.app/<workspace>` and is required only when enabled.
- Record that settings own optional provider configuration, the workspaces Model owns enablement policy, and Linear infrastructure is stateless.
- Remove or correct any wording that implies a fixed configured workspace at client construction.

## Backend test checklist

Extend existing Go tests rather than relying only on manual verification:

- Missing `linear` persistence defaults to disabled and empty.
- Configured Linear values parse and round-trip.
- New default files include the complete disabled object.
- Updates trim workspace whitespace before returning and persisting.
- Disabled malformed workspaces are accepted and preserved after trimming.
- Enabled empty workspaces are rejected without writing.
- Enabled workspaces containing reserved or whitespace characters are rejected.
- Enabled workspaces containing each allowed unreserved character class are accepted.
- Runtime configuration carries Linear values on startup, update, and reload.
- Settings controller applies Linear configuration in order with other Model configuration.
- Linear infrastructure builds the expected URL.
- Workspaces skip Linear when disabled and use the latest workspace after reconfiguration.

## Frontend verification checklist

There is currently no frontend unit-test framework in the repository, so do not introduce one solely for this feature unless the project direction changes. Cover pure logic through typechecking and verify the UI manually:

1. Start with no `linear` key and confirm the checkbox is off, the input is disabled, and no Linear actions appear.
2. Enable Linear with an empty workspace and confirm saving is blocked with a useful message.
3. Confirm surrounding whitespace is trimmed on save.
4. Confirm invalid reserved characters block saving, while mixed-case RFC unreserved characters are accepted and preserved.
5. Save a valid enabled workspace and confirm the topbar Ticket pill and command palette Open Issue action appear without restarting.
6. Use a branch containing a ticket key and confirm both actions open the configured Linear workspace URL.
7. Use a branch without a ticket key and confirm enabled actions are visible but disabled.
8. Change the workspace and confirm the old URL disappears immediately, a background refresh occurs, and the new URL is used.
9. Disable Linear and confirm both actions disappear immediately while the workspace input retains its value.
10. Re-enable Linear and confirm the retained workspace is used.
11. Edit the JSON out of band and run Reload Settings. Confirm the same runtime, visibility, invalidation, and refresh behaviour.

## Validation commands

Run from the repository root:

```sh
mise run fmt
mise run gen:openapi
mise run lint:openapi
npm --prefix web run typecheck
mise run test
```

`mise run test` regenerates OpenAPI and embedded frontend assets before running Go tests, so review generated diffs afterwards. Confirm no stale `signinsolutions` hardcoding remains:

```sh
rg -n 'signinsolutions|linear.NewClient' internal web/src README.md docs
```

The README example may intentionally contain `signinsolutions` as an explanatory example only if clearly marked as user-provided rather than a default.

## Acceptance criteria

- WADE starts successfully with old settings files and treats Linear as disabled.
- Newly created settings files contain disabled Linear defaults.
- Invalid enabled Linear settings fail through the normal settings validation path.
- Save and reload reconfigure Linear without restarting WADE.
- Backend workspace responses never contain Linear issues while disabled.
- The Ticket pill and Open Issue command are absent while disabled.
- Enabled integrations resolve branch ticket URLs against the configured workspace.
- No stale Linear URL is usable after a configuration change.
- Existing workspace, GitHub, pull request, worktree, shell, agent, and terminal behaviour remains unchanged.
- OpenAPI, generated frontend types, documentation, formatting, typechecking, and the full test suite are current.
