<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { useFuzzyItems } from '../../composables/useFuzzyProjects';
import { isProjectDetails, type ProjectDetails } from '../../composables/useProjectDetails';
import PaletteShell from './PaletteShell.vue';
import type { PaletteResult } from './types';

type CommandDefinition = {
  id: string;
  label: string;
  actionLabel: string;
  unavailableLabel: string;
  url: string;
};

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const route = useRoute();
const query = ref('');
const projectDetails = ref<ProjectDetails | undefined>();
const isProjectDetailsLoading = ref(false);
let detailsLoadRun = 0;

const currentProjectName = computed(() => route.name === 'project'
  ? String(route.params.projectName ?? '')
  : '');

const unavailableCommandLabel = (fallback: string) => {
  if (currentProjectName.value === '') {
    return 'No project open';
  }

  if (isProjectDetailsLoading.value) {
    return 'Loading';
  }

  return fallback;
};

const commandDefinitions = computed<CommandDefinition[]>(() => [
  {
    id: 'open-linear-ticket',
    label: 'Open Linear Ticket',
    actionLabel: 'Open ticket',
    unavailableLabel: unavailableCommandLabel('No ticket found'),
    url: projectDetails.value?.linearTicketUrl ?? ''
  },
  {
    id: 'open-pr',
    label: 'Open PR',
    actionLabel: 'Open PR',
    unavailableLabel: unavailableCommandLabel('No PR found'),
    url: projectDetails.value?.pullRequestUrl ?? ''
  },
  {
    id: 'open-github-page',
    label: 'Open Github Page',
    actionLabel: 'Open page',
    unavailableLabel: unavailableCommandLabel('No GitHub remote'),
    url: projectDetails.value?.githubUrl ?? ''
  }
]);

const { matchingItems: matchingCommands } = useFuzzyItems(
  commandDefinitions,
  query,
  (command) => command.label
);

const paletteSummary = computed(() => {
  if (currentProjectName.value === '') {
    return 'No project open';
  }

  return isProjectDetailsLoading.value ? `Loading ${currentProjectName.value}` : currentProjectName.value;
});

const openExternalUrl = (url: string) => {
  if (url === '') {
    return;
  }

  emit('close');
  window.open(url, '_blank', 'noopener,noreferrer');
};

const paletteResults = computed<PaletteResult[]>(() => matchingCommands.value.map((match) => ({
  id: `command:${match.item.id}`,
  label: match.item.label,
  actionLabel: match.item.url === '' ? match.item.unavailableLabel : match.item.actionLabel,
  isDisabled: match.item.url === '',
  run: () => openExternalUrl(match.item.url)
})));

const loadCurrentProjectDetails = async () => {
  detailsLoadRun += 1;
  const run = detailsLoadRun;

  if (currentProjectName.value === '') {
    projectDetails.value = undefined;
    isProjectDetailsLoading.value = false;
    return;
  }

  isProjectDetailsLoading.value = true;

  try {
    const params = new URLSearchParams({ project: currentProjectName.value });
    const response = await fetch(`/api/project?${params}`);

    if (!response.ok) {
      throw new Error(`Project details request failed with ${response.status}`);
    }

    const details: unknown = await response.json();
    if (!isProjectDetails(details) || detailsLoadRun !== run) {
      return;
    }

    projectDetails.value = details;
  } catch {
    if (detailsLoadRun === run) {
      projectDetails.value = undefined;
    }
  } finally {
    if (detailsLoadRun === run) {
      isProjectDetailsLoading.value = false;
    }
  }
};

onMounted(() => {
  void loadCurrentProjectDetails();
});

onBeforeUnmount(() => {
  detailsLoadRun += 1;
});
</script>

<template>
  <PaletteShell
    title="Run command"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Search commands"
    results-aria-label="Commands WADE can run"
    status-message="No matching commands"
    :results="paletteResults"
    @update:query="query = $event"
    @close="emit('close', $event)"
  />
</template>
