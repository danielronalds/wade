<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterView } from 'vue-router';
import { getSettings } from '@/api/generated/wade';
import CommandPalette from '@/features/command-palette/CommandPalette.vue';
import type { Settings } from '@/types/settings';
import { applyThemeAccentColor, storedThemeAccentColor } from '@/utils/theme';

applyThemeAccentColor(storedThemeAccentColor());

onMounted(async () => {
  try {
    const settings = await getSettings() as Settings;
    applyThemeAccentColor(settings.themeAccentColor);
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
