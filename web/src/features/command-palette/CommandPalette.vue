<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import { useRoute } from 'vue-router';
import ActiveWorkspacePalette from '@/features/command-palette/components/ActiveWorkspacePalette.vue';
import GeneralCommandPalette from '@/features/command-palette/components/GeneralCommandPalette.vue';
import RemoteRepositoryPalette from '@/features/command-palette/components/RemoteRepositoryPalette.vue';
import WorkspacePalette from '@/features/command-palette/components/WorkspacePalette.vue';
import CreateWorktreePalette from '@/features/command-palette/components/worktrees/CreateWorktreePalette.vue';
import RemoteWorktreePalette from '@/features/command-palette/components/worktrees/RemoteWorktreePalette.vue';
import RemoveWorktreePalette from '@/features/command-palette/components/worktrees/RemoveWorktreePalette.vue';
import { useCommandPaletteKeyboardShortcuts } from '@/features/command-palette/composables/useCommandPaletteKeyboardShortcuts';
import { useWorkspaceDetailsStore } from '@/stores/useWorkspaceDetailsStore';

const PaletteModes = {
  Workspaces: 'workspaces',
  ActiveWorkspaces: 'active-workspaces',
  Commands: 'commands',
  RemoteRepository: 'remote-repository',
  CreateWorktree: 'create-worktree',
  RemoteWorktree: 'remote-worktree',
  RemoveWorktree: 'remove-worktree'
} as const;

type PaletteMode = typeof PaletteModes[keyof typeof PaletteModes];

const route = useRoute();
const workspaceDetailsStore = useWorkspaceDetailsStore();
const activePalette = ref<PaletteMode | undefined>();
let previouslyFocusedElement: HTMLElement | null = null;

const currentWorkspaceId = computed(() => route.name === 'workspace'
  ? String(route.params.workspaceId ?? '')
  : '');
const currentWorkspace = computed(() => workspaceDetailsStore.getWorkspaceDetails(currentWorkspaceId.value));
const currentRepositoryId = computed(() => currentWorkspace.value?.repositoryId ?? '');

const restorePreviousFocus = () => {
  const element = previouslyFocusedElement;
  previouslyFocusedElement = null;

  if (!element || !document.contains(element)) {
    return;
  }

  element.focus();
};

const openPalette = (mode: PaletteMode) => {
  if (!activePalette.value) {
    previouslyFocusedElement = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  }

  activePalette.value = mode;
};

const closePalette = (restoreFocus = true) => {
  if (!activePalette.value) {
    return;
  }

  activePalette.value = undefined;

  if (restoreFocus) {
    void nextTick(restorePreviousFocus);
    return;
  }

  previouslyFocusedElement = null;
};

const openRepositoryWorktreePalette = (mode: PaletteMode) => {
  if (currentRepositoryId.value === '') {
    return;
  }

  openPalette(mode);
};

useCommandPaletteKeyboardShortcuts({
  openActiveWorkspacePicker: () => openPalette(PaletteModes.ActiveWorkspaces),
  openCommandPalette: () => openPalette(PaletteModes.Commands),
  openWorkspacePicker: () => openPalette(PaletteModes.Workspaces)
});
</script>

<template>
  <WorkspacePalette
    v-if="activePalette === PaletteModes.Workspaces"
    @close="closePalette"
  />
  <ActiveWorkspacePalette
    v-if="activePalette === PaletteModes.ActiveWorkspaces"
    @close="closePalette"
  />
  <GeneralCommandPalette
    v-if="activePalette === PaletteModes.Commands"
    @close="closePalette"
    @open-workspace-picker="openPalette(PaletteModes.Workspaces)"
    @open-active-workspace-picker="openPalette(PaletteModes.ActiveWorkspaces)"
    @open-remote-repository-picker="openPalette(PaletteModes.RemoteRepository)"
    @open-create-worktree="openRepositoryWorktreePalette(PaletteModes.CreateWorktree)"
    @open-remote-worktree-picker="openRepositoryWorktreePalette(PaletteModes.RemoteWorktree)"
    @open-remove-worktree="openRepositoryWorktreePalette(PaletteModes.RemoveWorktree)"
  />
  <RemoteRepositoryPalette
    v-if="activePalette === PaletteModes.RemoteRepository"
    @close="closePalette"
  />
  <CreateWorktreePalette
    v-if="activePalette === PaletteModes.CreateWorktree"
    :repository-id="currentRepositoryId"
    @close="closePalette"
  />
  <RemoteWorktreePalette
    v-if="activePalette === PaletteModes.RemoteWorktree"
    :repository-id="currentRepositoryId"
    @close="closePalette"
  />
  <RemoveWorktreePalette
    v-if="activePalette === PaletteModes.RemoveWorktree"
    :repository-id="currentRepositoryId"
    :workspace-id="currentWorkspaceId"
    @close="closePalette"
  />
</template>
