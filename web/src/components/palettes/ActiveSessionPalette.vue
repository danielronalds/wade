<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useActiveSessions } from '../../composables/useActiveSessions';
import { useFuzzyProjects } from '../../composables/useFuzzyProjects';
import PaletteShell from './PaletteShell.vue';
import type { PaletteResult } from './types';

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { activeSessions, isSyncing, syncActiveSessions } = useActiveSessions();
const query = ref('');
const { matchingProjects } = useFuzzyProjects(activeSessions, query);

const paletteSummary = computed(() => {
  const label = activeSessions.value.length === 1 ? 'active session' : 'active sessions';
  const summary = `${activeSessions.value.length} ${label}`;

  return isSyncing.value ? `Syncing ${summary}` : summary;
});

const statusMessage = computed(() => {
  if (activeSessions.value.length === 0 && isSyncing.value) {
    return 'Loading active sessions';
  }

  if (activeSessions.value.length === 0) {
    return 'No active sessions';
  }

  return 'No matching active sessions';
});

const closePalette = (restoreFocus = true) => {
  emit('close', restoreFocus);
};

const openSession = async (projectName: string) => {
  const currentRoute = router.currentRoute.value;
  const currentProjectName = String(currentRoute.params.projectName ?? '');
  if (currentRoute.name === 'project' && currentProjectName === projectName) {
    closePalette();
    return;
  }

  closePalette(false);
  await router.push({ name: 'project', params: { projectName } });
};

const paletteResults = computed<PaletteResult[]>(() => matchingProjects.value.map((match) => ({
  id: `active-session:${match.projectName}`,
  label: match.projectName,
  actionLabel: 'Reconnect',
  isDisabled: false,
  run: () => {
    void openSession(match.projectName);
  }
})));

onMounted(() => {
  void syncActiveSessions();
});
</script>

<template>
  <PaletteShell
    title="Open active session"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Search active sessions"
    results-aria-label="Active WADE sessions"
    :status-message="statusMessage"
    :results="paletteResults"
    @update:query="query = $event"
    @close="closePalette"
  />
</template>
