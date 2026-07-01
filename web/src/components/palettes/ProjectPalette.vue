<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useFuzzyProjects } from '../../composables/useFuzzyProjects';
import { useProjects } from '../../composables/useProjects';
import PaletteShell from './PaletteShell.vue';
import type { PaletteResult } from './types';

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { isSyncing, projects, syncProjects } = useProjects();
const query = ref('');
const { matchingProjects } = useFuzzyProjects(projects, query);

const paletteSummary = computed(() => {
  const label = projects.value.length === 1 ? 'project' : 'projects';
  const summary = `${projects.value.length} ${label}`;

  return isSyncing.value ? `Syncing ${summary}` : summary;
});

const statusMessage = computed(() => {
  if (projects.value.length === 0 && isSyncing.value) {
    return 'Loading projects';
  }

  if (projects.value.length === 0) {
    return 'No projects found';
  }

  return 'No matching projects';
});

const closePalette = (restoreFocus = true) => {
  emit('close', restoreFocus);
};

const openProject = async (projectName: string) => {
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
  id: `project:${match.projectName}`,
  label: match.projectName,
  actionLabel: 'Open project',
  isDisabled: false,
  run: () => {
    void openProject(match.projectName);
  }
})));

onMounted(() => {
  void syncProjects();
});
</script>

<template>
  <PaletteShell
    title="Open project"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Search projects"
    results-aria-label="Projects WADE can see"
    :status-message="statusMessage"
    :results="paletteResults"
    @update:query="query = $event"
    @close="closePalette"
  />
</template>
