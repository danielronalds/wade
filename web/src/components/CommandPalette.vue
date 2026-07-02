<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import GeneralCommandPalette from './palettes/GeneralCommandPalette.vue';
import ProjectPalette from './palettes/ProjectPalette.vue';
import RemoteProjectPalette from './palettes/RemoteProjectPalette.vue';
import CreateWorktreePalette from './palettes/worktrees/CreateWorktreePalette.vue';
import RemoteWorktreePalette from './palettes/worktrees/RemoteWorktreePalette.vue';
import RemoveWorktreePalette from './palettes/worktrees/RemoveWorktreePalette.vue';

const PaletteModes = {
  Projects: 'projects',
  Commands: 'commands',
  RemoteProject: 'remote-project',
  CreateWorktree: 'create-worktree',
  RemoteWorktree: 'remote-worktree',
  RemoveWorktree: 'remove-worktree'
} as const;

type PaletteMode = typeof PaletteModes[keyof typeof PaletteModes];

const route = useRoute();
const activePalette = ref<PaletteMode | undefined>();
let previouslyFocusedElement: HTMLElement | null = null;

const currentProjectName = computed(() => route.name === 'project'
  ? String(route.params.projectName ?? '')
  : '');

const isProjectPaletteShortcut = (event: KeyboardEvent) => event.ctrlKey
  && !event.altKey
  && !event.metaKey
  && event.key.toLowerCase() === 's';

const isCommandPaletteShortcut = (event: KeyboardEvent) => event.ctrlKey
  && !event.altKey
  && !event.metaKey
  && event.key.toLowerCase() === 'p';

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

const openProjectWorktreePalette = (mode: PaletteMode) => {
  if (currentProjectName.value === '') {
    return;
  }

  openPalette(mode);
};

const handleGlobalKeydown = (event: KeyboardEvent) => {
  if (isProjectPaletteShortcut(event)) {
    event.preventDefault();
    event.stopImmediatePropagation();
    openPalette(PaletteModes.Projects);
    return;
  }

  if (isCommandPaletteShortcut(event)) {
    event.preventDefault();
    event.stopImmediatePropagation();
    openPalette(PaletteModes.Commands);
  }
};

onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown, true);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeydown, true);
});
</script>

<template>
  <ProjectPalette
    v-if="activePalette === PaletteModes.Projects"
    @close="closePalette"
  />
  <GeneralCommandPalette
    v-if="activePalette === PaletteModes.Commands"
    @close="closePalette"
    @open-project-picker="openPalette(PaletteModes.Projects)"
    @open-remote-project-picker="openPalette(PaletteModes.RemoteProject)"
    @open-create-worktree="openProjectWorktreePalette(PaletteModes.CreateWorktree)"
    @open-remote-worktree-picker="openProjectWorktreePalette(PaletteModes.RemoteWorktree)"
    @open-remove-worktree="openProjectWorktreePalette(PaletteModes.RemoveWorktree)"
  />
  <RemoteProjectPalette
    v-if="activePalette === PaletteModes.RemoteProject"
    @close="closePalette"
  />
  <CreateWorktreePalette
    v-if="activePalette === PaletteModes.CreateWorktree"
    :project-name="currentProjectName"
    @close="closePalette"
  />
  <RemoteWorktreePalette
    v-if="activePalette === PaletteModes.RemoteWorktree"
    :project-name="currentProjectName"
    @close="closePalette"
  />
  <RemoveWorktreePalette
    v-if="activePalette === PaletteModes.RemoveWorktree"
    :project-name="currentProjectName"
    @close="closePalette"
  />
</template>
