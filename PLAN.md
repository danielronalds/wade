# Remember Workspace Session State

## Goal

Make switching between active WADE workspaces feel like resuming the same session. Each workspace should reopen with the navigation, terminal layout and review workflow that the user last left behind.

State will be checkpointed in `localStorage` so a new browser or PWA window can resume an existing daemon-backed session. Browser windows will not live-synchronise. The latest write for a workspace becomes the checkpoint used by the next window.

## Behaviour

### Restored state

For each workspace, restore:

- Active workspace tab: Terminal, Server or Review.
- Whether Scratchpad is open, while retaining the underlying active tab to return to when Scratchpad closes.
- Active Terminal pane: Agent or Misc.
- Terminal layout: split or the currently zoomed pane.
- Selected agent.
- Active review state:
  - Snapshot ID.
  - Selected scope and file.
  - File filter.
  - Reviewed-file markers.
  - Collapsed directories.
  - Saved comments and open comment draft.
  - Overall note and open overall-note draft.
  - Diff display settings.

Do not checkpoint derived or transient state:

- Terminal connection statuses.
- Terminal scroll positions or scrollback, which remain owned by the terminal backend and xterm session.
- Review file-content caches.
- Review loading, request, error or sending states.
- Review editor scroll positions.

### Session lifetime

A checkpoint is valid only while the workspace has running terminals.

Before entering a workspace route:

1. Call `listWorkspaceTerminals(workspaceId)` before terminal components mount.
2. If the response contains running terminals, hydrate the workspace checkpoint.
3. If the response is successfully returned but contains no terminals, clear the checkpoint and create fresh state.
4. If terminal validation fails, preserve the checkpoint and avoid destructive validation.

Fresh state is:

- Terminal tab selected.
- Scratchpad closed.
- Split terminal layout.
- Agent pane active.
- Default configured agent selected.
- No active review.

Clear the complete checkpoint when:

- “Close Workspace Terminals” succeeds.
- The workspace's worktree is removed.

A daemon restart naturally invalidates every checkpoint because no workspace terminals remain.

### Multi-window behaviour

- Use one versioned `localStorage` key per workspace.
- Load storage into independent Pinia state for each browser window.
- Do not subscribe to storage events or mutate an open window from another window's writes.
- Persist local changes as a complete checkpoint, with last writer winning for future restoration.
- Accept that cancelling or finishing a shared review snapshot in one window can invalidate it in another window. A stale checkpoint will be removed when next restored.

### Review restoration

When an active session has a stored review snapshot ID:

1. Load the snapshot with `getReviewSnapshot`.
2. On success, populate runtime review metadata and restore the checkpointed review workflow.
3. On any snapshot-load failure, clear only the review workflow state.
4. Keep the Review tab selected so the normal “Start Review” screen is shown.

Do not delete snapshots when `ReviewTab` unmounts during route navigation. Delete the active snapshot when the review is explicitly cancelled or finished, or best-effort when its workspace checkpoint is cleared.

## Implementation

### 1. Add the workspace session store

Create `web/src/stores/useWorkspaceSessionStore.ts` as the single owner of workspace session state.

Define a versioned, serialisable checkpoint shape similar to:

```ts
type WorkspaceSessionCheckpoint = {
  version: 1;
  activeTab: WorkspaceTab;
  isScratchpadOpen: boolean;
  terminal: {
    activePane: TerminalPaneId;
    zoomedPane: TerminalPaneId | null;
    selectedAgentName: string;
  };
  review: ReviewCheckpoint | null;
};
```

The review checkpoint will contain only the user and display fields listed under “Restored state”. Keep loaded `ReviewData` and asynchronous request state as non-serialised runtime state.

The store will:

- Create fresh workspace state.
- Parse and validate stored JSON without trusting its shape.
- Reject unknown versions and invalid enum values.
- Hydrate one workspace only after session validation.
- Serialise the allowed checkpoint fields after reactive changes.
- Avoid writes until initial hydration or reset has completed.
- Handle unavailable or full storage without breaking the active UI.
- Expose getters/actions for workspace, terminal and review consumers.
- Clear local state and best-effort delete an associated review snapshot.

Use a key such as `wade:workspace-session:v1:<workspaceId>`. Do not register a `storage` event listener.

### 2. Initialise sessions before workspace components mount

Update `web/src/router/index.ts` to prepare workspace session state before entering a workspace route.

The preparation action will:

- List the target workspace's terminals.
- Clear storage and initialise defaults after a successful empty result.
- Hydrate storage for an active session.
- Migrate an existing selected-agent value only when the session is active.
- Restore the stored review snapshot after successful terminal validation.
- Preserve stored data when terminal validation itself fails.

Keeping this in route preparation prevents `TerminalTab` and `ServerTab` from creating new terminals before WADE determines whether the previous session still exists.

### 3. Bind workspace navigation to session state

Update `web/src/views/workspace/WorkspaceView.vue` to use the store-backed state instead of local defaults for:

- `activeTab`.
- `isScratchpadOpen`.
- Derived Scratchpad mount/open state.

Keep connection status and component references local because they belong to the current component instances. Existing sidebar clicks, keyboard shortcuts, Review transitions and focus handling should update the store-backed values through the same interaction paths.

When Scratchpad is restored, mount and focus it while retaining the stored underlying tab.

### 4. Bind Terminal state to the checkpoint

Update `web/src/views/workspace/tabs/terminal/composables/useTerminalTabPaneZoom.ts` to accept store-backed refs for `activePane` and `zoomedPane` rather than creating private refs.

Update `web/src/views/workspace/tabs/terminal/TerminalTab.vue` to:

- Read and update the selected agent through the session store.
- Validate a restored agent against current settings and fall back to the configured default when necessary.
- Preserve the existing pane activation, zoom and focus behaviour.
- Continue deriving connection state locally.

Update `web/src/features/terminal-session/composables/useAgentTerminalInput.ts` to resolve the selected agent through the session store.

Retire `web/src/features/terminal-session/composables/useSelectedAgent.ts` after moving its legacy-key migration into the session store. Remove migrated legacy keys so explicit session clearing produces genuinely fresh agent state.

### 5. Move Review workflow state into the store

Refactor `web/src/views/workspace/tabs/review/ReviewTab.vue` so checkpointed review fields are store-backed while transient rendering/request fields remain component-local.

Store-backed fields include:

- Snapshot identity and loaded review metadata.
- Scope, selected file and filter.
- Reviewed markers and collapsed directories.
- Comments, overall note and both draft editors.
- Diff display options.

Component-local fields include:

- File-content request cache.
- Abort controllers and request-run counters.
- Element refs.
- Loading, error and send status.

Lifecycle changes:

- Starting a review creates and stores a new review checkpoint.
- Mounting a restored review loads the selected file contents on demand.
- Unmounting aborts requests and removes listeners, but does not delete or reset the review.
- Cancelling and finishing delete the snapshot, clear review state and return to Terminal using existing behaviour.
- Resetting one review restores the existing review defaults without clearing unrelated workspace state.

Replace `web/src/views/workspace/tabs/review/composables/useReviewState.ts` with store-derived review status so the command palette and Review tab observe the same source of truth.

### 6. Clear state when sessions are deliberately destroyed

Update `web/src/features/command-palette/components/GeneralCommandPalette.vue` so a successful `deleteWorkspaceTerminals` call clears the complete workspace checkpoint before navigation finishes.

Update `web/src/features/command-palette/components/worktrees/RemoveWorktreePalette.vue` so successful worktree removal clears the removed workspace's checkpoint, whether or not it is the currently displayed workspace.

Checkpoint clearing should not turn a best-effort review snapshot deletion failure into a failed terminal-close or worktree-removal action.

## Implementation sequencing

Keep this work on one branch and in one pull request, but implement it over two or three focused coding sessions rather than attempting the complete change at once:

1. Add the session store, persistence and route validation, then restore workspace navigation and Terminal layout state.
2. Move Review workflow state into the store and correct snapshot ownership and lifecycle.
3. Integrate terminal-close and worktree-removal cleanup, then complete typechecking, builds and manual multi-window validation.

The Review refactor is the highest-risk stage because `ReviewTab.vue` owns substantial workflow and asynchronous request state. Keeping it separate from the foundational session-state work will make regressions easier to identify and review.

## Validation

Run:

```sh
npm --prefix web run typecheck
npm --prefix web run build
mise run test
```

Manually verify:

1. Switch between two active workspaces with different Terminal, Server, Review and Scratchpad selections.
2. Restore different Agent/Misc focus and split/zoom layouts for each workspace.
3. Confirm selected agents restore and invalid configured agents fall back safely.
4. Start reviews in multiple workspaces and restore selected files, filters, reviewed markers, comments, open drafts and display options.
5. Reload the browser and open a second PWA window while the daemon remains running; confirm the latest checkpoint restores.
6. Use two windows on the same workspace and confirm neither live-updates while the latest write is used by a subsequently opened window.
7. Close a workspace through “Close Workspace Terminals”, reopen it and confirm completely fresh state.
8. Remove a worktree and confirm its local-storage checkpoint is deleted.
9. Restart the daemon, reopen a former workspace and confirm fresh state is used.
10. Delete or invalidate a review snapshot, then restore the workspace and confirm Review remains selected but shows the normal start state.
11. Simulate terminal-list failure and confirm the existing checkpoint is not erased.

## Out of scope

- Persisting terminal processes or review snapshots across daemon restarts.
- Live synchronisation or conflict resolution between browser windows.
- Restoring terminal or editor scroll positions.
- Persisting fetched diff contents or transient API errors.
- Automatically reopening a particular workspace when the PWA starts at `/`.
