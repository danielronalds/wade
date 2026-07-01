<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import GeneralCommandPalette from './palettes/GeneralCommandPalette.vue';
import ProjectPalette from './palettes/ProjectPalette.vue';

const PaletteModes = {
  Projects: 'projects',
  Commands: 'commands'
} as const;

type PaletteMode = typeof PaletteModes[keyof typeof PaletteModes];

const activePalette = ref<PaletteMode | undefined>();
let previouslyFocusedElement: HTMLElement | null = null;

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
  />
</template>
