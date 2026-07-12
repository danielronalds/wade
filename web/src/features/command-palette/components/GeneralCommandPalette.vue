<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  closeProjectSession,
  getProjectDetails,
  reloadConfig as requestReloadConfig
} from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';
import type { ProjectDetails } from '@/views/project/composables/useProjectDetails';
import { useProjects } from '@/features/projects/composables/useProjects';
import { useRecentProjects } from '@/features/projects/composables/useRecentProjects';
import { useSettingsStore } from '@/stores/useSettingsStore';
import { isReviewInProgressState, useReviewSessionState } from '@/views/project/tabs/review/composables/useReviewSessionState';
import { dispatchCancelReviewEvent } from '@/views/project/tabs/review/events/cancelReview';
import { dispatchStartReviewEvent } from '@/views/project/tabs/review/events/startReview';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import type { PaletteNotice, PaletteResult } from '@/features/command-palette/types';

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
  openProjectPicker: [];
  openActiveSessionPicker: [];
  openRemoteProjectPicker: [];
  openCreateWorktree: [];
  openRemoteWorktreePicker: [];
  openRemoveWorktree: [];
}>();

const route = useRoute();
const router = useRouter();
const { syncProjects } = useProjects();
const { removeUnavailableRecentProjects } = useRecentProjects();
const { loadSettings } = useSettingsStore();
const query = ref('');
const projectDetails = ref<ProjectDetails | undefined>();
const isProjectDetailsLoading = ref(false);
const isClosingSession = ref(false);
const closeSessionError = ref('');
let detailsLoadRun = 0;

const currentProjectName = computed(() => route.name === 'project'
  ? String(route.params.projectName ?? '')
  : '');
const currentReviewState = useReviewSessionState(currentProjectName);
const isReviewInProgress = computed(() => isReviewInProgressState(currentReviewState.value));

const closeSessionActionLabel = computed(() => {
  if (currentProjectName.value === '') {
    return 'No project open';
  }

  if (isClosingSession.value) {
    return 'Closing';
  }

  return 'Close terminals';
});

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

const errorMessage = (error: unknown, fallback: string) => error instanceof Error
  ? error.message
  : fallback;

const updateQuery = (nextQuery: string) => {
  query.value = nextQuery;
  closeSessionError.value = '';
};

const closeCurrentSession = async () => {
  const projectName = currentProjectName.value;
  if (projectName === '' || isClosingSession.value) {
    return;
  }

  isClosingSession.value = true;
  closeSessionError.value = '';

  try {
    await closeProjectSession(projectName);
    closePaletteWithoutRestoringFocus();
    await router.push({ name: 'home' });
  } catch (error) {
    closeSessionError.value = errorMessage(error, 'Session close failed');
  } finally {
    isClosingSession.value = false;
  }
};

const startReview = () => {
  if (currentProjectName.value === '') {
    return;
  }

  closePaletteWithoutRestoringFocus();
  dispatchStartReviewEvent(currentProjectName.value);
};

const cancelReview = () => {
  if (currentProjectName.value === '' || !isReviewInProgress.value) {
    return;
  }

  closePaletteWithoutRestoringFocus();
  dispatchCancelReviewEvent(currentProjectName.value);
};

const openCreateWorktree = () => {
  if (currentProjectName.value === '') {
    return;
  }

  emit('openCreateWorktree');
};

const openRemoteWorktreePicker = () => {
  if (currentProjectName.value === '') {
    return;
  }

  emit('openRemoteWorktreePicker');
};

const openRemoveWorktree = () => {
  if (currentProjectName.value === '') {
    return;
  }

  emit('openRemoveWorktree');
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
    await requestReloadConfig();

    try {
      await loadSettings({ force: true });
    } catch (settingsError) {
      console.error(settingsError);
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

const reviewCommand = computed<PaletteResult>(() => {
  if (isReviewInProgress.value) {
    return {
      id: 'cancel-review',
      label: 'Cancel Review',
      actionLabel: currentReviewState.value === 'loading' ? 'Stop starting review' : 'Discard review',
      isDisabled: false,
      run: cancelReview
    };
  }

  return {
    id: 'start-review',
    label: 'Start Review',
    actionLabel: currentProjectName.value === '' ? 'No project open' : 'Open review tab',
    isDisabled: currentProjectName.value === '',
    run: startReview
  };
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
    id: 'open-active-session-picker',
    label: 'Open Active Session Picker',
    actionLabel: 'Open picker',
    isDisabled: false,
    run: () => emit('openActiveSessionPicker')
  },
  {
    id: 'clone-remote-project',
    label: 'Clone Remote Project',
    actionLabel: 'Pick repo',
    isDisabled: false,
    run: () => emit('openRemoteProjectPicker')
  },
  reviewCommand.value,
  {
    id: 'close-current-session',
    label: 'Close Current Session',
    actionLabel: closeSessionActionLabel.value,
    isDisabled: currentProjectName.value === '' || isClosingSession.value,
    run: () => {
      void closeCurrentSession();
    }
  },
  {
    id: 'create-open-worktree',
    label: 'Create/Open Worktree',
    actionLabel: currentProjectName.value === '' ? 'No project open' : 'Enter branch',
    isDisabled: currentProjectName.value === '',
    run: openCreateWorktree
  },
  {
    id: 'checkout-remote-worktree',
    label: 'Checkout Remote Branch as Worktree',
    actionLabel: currentProjectName.value === '' ? 'No project open' : 'Pick branch',
    isDisabled: currentProjectName.value === '',
    run: openRemoteWorktreePicker
  },
  {
    id: 'remove-worktree',
    label: 'Remove Worktree',
    actionLabel: currentProjectName.value === '' ? 'No project open' : 'Pick worktree',
    isDisabled: currentProjectName.value === '',
    run: openRemoveWorktree
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

const notice = computed<PaletteNotice | undefined>(() => closeSessionError.value === ''
  ? undefined
  : {
    tone: 'error',
    title: 'Session close failed',
    messages: [closeSessionError.value]
  });

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
    const details = await getProjectDetails({ project: currentProjectName.value });
    if (detailsLoadRun !== run) {
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
    :notice="notice"
    @update:query="updateQuery"
    @close="emit('close', $event)"
  />
</template>
