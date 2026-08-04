<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterView } from 'vue-router';
import CommandPalette from '@/features/command-palette/CommandPalette.vue';
import { useActiveWorkspaces } from '@/features/workspaces/composables/useActiveWorkspaces';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';
import { useSettingsStore } from '@/stores/useSettingsStore';
import { applyThemeAccentColor, storedThemeAccentColor } from '@/utils/theme';

const { loadSettings } = useSettingsStore();
const { syncActiveWorkspaces } = useActiveWorkspaces();
const { syncWorkspaces } = useWorkspaces();

applyThemeAccentColor(storedThemeAccentColor());

onMounted(async () => {
  void syncActiveWorkspaces();
  void syncWorkspaces();

  try {
    await loadSettings();
  } catch (error) {
    console.error('Unable to load settings', error);
  }
});
</script>

<template>
  <main>
    <RouterView v-slot="{ Component, route }">
      <component :is="Component" :key="route.fullPath" />
    </RouterView>
    <CommandPalette />
  </main>
</template>

<style scoped>
main {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--window);
}
</style>
