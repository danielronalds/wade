<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterView } from 'vue-router';
import CommandPalette from '@/features/command-palette/CommandPalette.vue';
import { useSettingsStore } from '@/stores/useSettingsStore';
import { applyThemeAccentColor, storedThemeAccentColor } from '@/utils/theme';

const { loadSettings } = useSettingsStore();

applyThemeAccentColor(storedThemeAccentColor());

onMounted(async () => {
  try {
    await loadSettings();
  } catch {
    return;
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
