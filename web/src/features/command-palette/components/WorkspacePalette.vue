<script setup lang="ts">
import { GitBranch } from '@lucide/vue';
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import type { PaletteResult } from '@/features/command-palette/types';
import { useFuzzyWorkspaces } from '@/features/workspaces/composables/useFuzzyWorkspaces';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';
import { getWorkspacePresentation } from '@/features/workspaces/workspacePresentation';

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { isSyncing, workspaces, syncWorkspaces } = useWorkspaces();
const query = ref('');
const { matchingWorkspaces } = useFuzzyWorkspaces(workspaces, query);

const paletteSummary = computed(() => {
  const label = workspaces.value.length === 1 ? 'workspace' : 'workspaces';
  const summary = `${workspaces.value.length} ${label}`;

  return isSyncing.value ? `Syncing ${summary}` : summary;
});

const statusMessage = computed(() => {
  if (workspaces.value.length === 0 && isSyncing.value) {
    return 'Loading workspaces';
  }

  if (workspaces.value.length === 0) {
    return 'No workspaces found';
  }

  return 'No matching workspaces';
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

const paletteResults = computed<PaletteResult[]>(() =>
  matchingWorkspaces.value.map((match) => {
    const presentation = getWorkspacePresentation(match.workspace);

    return {
      id: `workspace:${match.workspace.id}`,
      label: presentation.root,
      secondaryLabel: presentation.branch || undefined,
      icon: presentation.branch === '' ? undefined : GitBranch,
      title: presentation.title,
      isDisabled: false,
      run: () => {
        void openWorkspace(match.workspace.id);
      }
    };
  })
);

onMounted(() => {
  void syncWorkspaces();
});
</script>

<template>
  <PaletteShell
    title="Open workspace"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Search workspaces"
    results-aria-label="Workspaces WADE can see"
    :status-message="statusMessage"
    :results="paletteResults"
    @update:query="query = $event"
    @close="closePalette"
  />
</template>
