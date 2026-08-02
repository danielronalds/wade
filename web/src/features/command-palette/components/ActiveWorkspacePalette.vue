<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import type { PaletteResult } from '@/features/command-palette/types';
import { useActiveWorkspaces } from '@/features/workspaces/composables/useActiveWorkspaces';
import { useFuzzyWorkspaces } from '@/features/workspaces/composables/useFuzzyWorkspaces';

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { activeWorkspaces, isSyncing, syncActiveWorkspaces } = useActiveWorkspaces();
const query = ref('');
const { matchingWorkspaces } = useFuzzyWorkspaces(activeWorkspaces, query);

const paletteSummary = computed(() => {
  const label = activeWorkspaces.value.length === 1 ? 'active workspace' : 'active workspaces';
  const summary = `${activeWorkspaces.value.length} ${label}`;

  return isSyncing.value ? `Syncing ${summary}` : summary;
});

const statusMessage = computed(() => {
  if (activeWorkspaces.value.length === 0 && isSyncing.value) {
    return 'Loading active workspaces';
  }

  if (activeWorkspaces.value.length === 0) {
    return 'No active workspaces';
  }

  return 'No matching active workspaces';
});

const closePalette = (restoreFocus = true) => {
  emit('close', restoreFocus);
};

const openWorkspace = async (workspaceId: string) => {
  const currentRoute = router.currentRoute.value;
  const currentWorkspaceId = String(currentRoute.params.workspaceId ?? '');
  if (currentRoute.name === 'workspace' && currentWorkspaceId === workspaceId) {
    closePalette();
    return;
  }

  closePalette(false);
  await router.push({ name: 'workspace', params: { workspaceId } });
};

const paletteResults = computed<PaletteResult[]>(() => matchingWorkspaces.value.map((match) => ({
  id: `active-workspace:${match.workspace.id}`,
  label: match.workspace.name,
  actionLabel: 'Reconnect',
  isDisabled: false,
  run: () => {
    void openWorkspace(match.workspace.id);
  }
})));

onMounted(() => {
  void syncActiveWorkspaces();
});
</script>

<template>
  <PaletteShell
    title="Open active workspace"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Search active workspaces"
    results-aria-label="Active WADE workspaces"
    :status-message="statusMessage"
    :results="paletteResults"
    @update:query="query = $event"
    @close="closePalette"
  />
</template>
