<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useFuzzyItems } from '../../composables/useFuzzyProjects';
import { isProjectDetails, type ProjectDetails } from '../../composables/useProjectDetails';
import { useProjects } from '../../composables/useProjects';
import { useRecentProjects } from '../../composables/useRecentProjects';
import { dispatchStartReviewEvent } from '../../events/startReview';
import PaletteShell from './PaletteShell.vue';
import type { PaletteResult } from './types';

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
  openProjectPicker: [];
}>();

const route = useRoute();
const router = useRouter();
const { syncProjects } = useProjects();
const { removeUnavailableRecentProjects } = useRecentProjects();
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

const openExternalUrl = (url: string) => {
  if (url === '') {
    return;
  }

  emit('close');
  window.open(url, '_blank', 'noopener,noreferrer');
};

const closePaletteWithoutRestoringFocus = () => {
  emit('close', false);
};

const startReview = () => {
  if (currentProjectName.value === '') {
    return;
  }

  closePaletteWithoutRestoringFocus();
  dispatchStartReviewEvent(currentProjectName.value);
};

const openSettings = async () => {
  if (route.name === 'settings') {
    emit('close');
    return;
  }

  emit('close', false);
  await router.push({ name: 'settings' });
};

const reloadConfig = async () => {
  try {
    const response = await fetch('/api/config/reload', { method: 'POST' });
    if (!response.ok) {
      throw new Error(`Config reload failed with ${response.status}`);
    }

    const availableProjects = await syncProjects();
    if (availableProjects) {
      removeUnavailableRecentProjects(availableProjects);
    }
  } catch (error) {
    console.error(error);
  } finally {
    emit('close');
  }
};

const createExternalCommand = (
  id: string,
  label: string,
  actionLabel: string,
  unavailableLabel: string,
  url: string
): PaletteResult => ({
  id,
  label,
  actionLabel: url === '' ? unavailableLabel : actionLabel,
  isDisabled: url === '',
  run: () => openExternalUrl(url)
});

const commandDefinitions = computed<PaletteResult[]>(() => [
  {
    id: 'open-project-picker',
    label: 'Open Project Picker',
    actionLabel: 'Open picker',
    isDisabled: false,
    run: () => emit('openProjectPicker')
  },
  {
    id: 'start-review',
    label: 'Start Review',
    actionLabel: currentProjectName.value === '' ? 'No project open' : 'Open review tab',
    isDisabled: currentProjectName.value === '',
    run: startReview
  },
  {
    id: 'open-settings',
    label: 'Open Settings',
    actionLabel: route.name === 'settings' ? 'Already open' : 'Open settings',
    isDisabled: false,
    run: () => {
      void openSettings();
    }
  },
  {
    id: 'reload-config',
    label: 'Reload Config',
    actionLabel: 'Reload config',
    isDisabled: false,
    run: () => {
      void reloadConfig();
    }
  },
  createExternalCommand(
    'open-linear-ticket',
    'Open Linear Ticket',
    'Open ticket',
    unavailableCommandLabel('No ticket found'),
    projectDetails.value?.linearTicketUrl ?? ''
  ),
  createExternalCommand(
    'open-pr',
    'Open PR',
    'Open PR',
    unavailableCommandLabel('No PR found'),
    projectDetails.value?.pullRequestUrl ?? ''
  ),
  createExternalCommand(
    'open-github-page',
    'Open Github Page',
    'Open page',
    unavailableCommandLabel('No GitHub remote'),
    projectDetails.value?.githubUrl ?? ''
  )
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

const paletteResults = computed<PaletteResult[]>(() => matchingCommands.value.map((match) => ({
  ...match.item,
  id: `command:${match.item.id}`
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
