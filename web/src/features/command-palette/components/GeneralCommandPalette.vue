<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { deleteWorkspaceTerminals } from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import type { PaletteNotice, PaletteResult } from '@/features/command-palette/types';
import { useActiveWorkspaces } from '@/features/workspaces/composables/useActiveWorkspaces';
import { useRecentWorkspaces } from '@/features/workspaces/composables/useRecentWorkspaces';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';
import { useSettingsStore } from '@/stores/useSettingsStore';
import { useWorkspaceDetailsStore } from '@/stores/useWorkspaceDetailsStore';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';
import { isReviewInProgressState } from '@/types/review';
import { dispatchCancelReviewEvent } from '@/views/workspace/tabs/review/events/cancelReview';
import { dispatchStartReviewEvent } from '@/views/workspace/tabs/review/events/startReview';
import { dispatchSubmitReviewEvent } from '@/views/workspace/tabs/review/events/submitReview';

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
  openWorkspacePicker: [];
  openActiveWorkspacePicker: [];
  openRemoteRepositoryPicker: [];
  openCreateWorktree: [];
  openRemoteWorktreePicker: [];
  openRemoveWorktree: [];
}>();

const route = useRoute();
const router = useRouter();
const { syncActiveWorkspaces } = useActiveWorkspaces();
const { syncWorkspaces } = useWorkspaces();
const { removeUnavailableRecentWorkspaces } = useRecentWorkspaces();
const settingsStore = useSettingsStore();
const workspaceDetailsStore = useWorkspaceDetailsStore();
const workspaceSessionStore = useWorkspaceSessionStore();

const query = ref('');
const isClosingTerminals = ref(false);
const closeTerminalsError = ref('');

const currentWorkspaceId = computed(() => (route.name === 'workspace' ? String(route.params.workspaceId ?? '') : ''));
const workspaceDetails = computed(() => workspaceDetailsStore.getWorkspaceDetails(currentWorkspaceId.value));
const isWorkspaceDetailsLoading = computed(() =>
  workspaceDetailsStore.isWorkspaceDetailsLoading(currentWorkspaceId.value)
);
const isWaitingForWorkspaceDetails = computed(() => isWorkspaceDetailsLoading.value && !workspaceDetails.value);
const currentReviewState = computed(() => workspaceSessionStore.getReviewState(currentWorkspaceId.value));
const isReviewInProgress = computed(() => isReviewInProgressState(currentReviewState.value));
const hasPendingReviewComments = computed(() =>
  workspaceSessionStore.hasPendingReviewComments(currentWorkspaceId.value)
);
const hasRepository = computed(() => Boolean(workspaceDetails.value?.repositoryId));

const closeTerminalsActionLabel = computed(() => {
  if (currentWorkspaceId.value === '') {
    return 'No workspace open';
  }

  if (isClosingTerminals.value) {
    return 'Closing';
  }

  return 'Close terminals';
});

const unavailableCommandLabel = (fallback: string) => {
  if (currentWorkspaceId.value === '') {
    return 'No workspace open';
  }

  if (isWaitingForWorkspaceDetails.value) {
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

const errorMessage = (error: unknown, fallback: string) => (error instanceof Error ? error.message : fallback);

const updateQuery = (nextQuery: string) => {
  query.value = nextQuery;
  closeTerminalsError.value = '';
};

const closeCurrentWorkspaceTerminals = async () => {
  const workspaceId = currentWorkspaceId.value;
  if (workspaceId === '' || isClosingTerminals.value) {
    return;
  }

  isClosingTerminals.value = true;
  closeTerminalsError.value = '';

  try {
    await deleteWorkspaceTerminals(workspaceId);
    workspaceSessionStore.clearWorkspaceSession(workspaceId);
    void syncActiveWorkspaces();
    closePaletteWithoutRestoringFocus();
    await router.push({ name: 'home' });
  } catch (error) {
    closeTerminalsError.value = errorMessage(error, 'Terminal close failed');
  } finally {
    isClosingTerminals.value = false;
  }
};

const startReview = () => {
  if (currentWorkspaceId.value === '') {
    return;
  }

  closePaletteWithoutRestoringFocus();
  dispatchStartReviewEvent(currentWorkspaceId.value);
};

const cancelReview = () => {
  if (currentWorkspaceId.value === '' || !isReviewInProgress.value) {
    return;
  }

  closePaletteWithoutRestoringFocus();
  dispatchCancelReviewEvent(currentWorkspaceId.value);
};

const submitReview = () => {
  if (currentWorkspaceId.value === '' || !hasPendingReviewComments.value) {
    return;
  }

  closePaletteWithoutRestoringFocus();
  dispatchSubmitReviewEvent(currentWorkspaceId.value);
};

const openCreateWorktree = () => {
  if (hasRepository.value) {
    emit('openCreateWorktree');
  }
};

const openRemoteWorktreePicker = () => {
  if (hasRepository.value) {
    emit('openRemoteWorktreePicker');
  }
};

const openRemoveWorktree = () => {
  if (hasRepository.value) {
    emit('openRemoveWorktree');
  }
};

const openSettings = async () => {
  if (route.name === 'settings') {
    emit('close');
    return;
  }

  emit('close', false);
  await router.push({ name: 'settings' });
};

const reloadSettings = async () => {
  try {
    await settingsStore.reloadSettingsFromDisk();

    const availableWorkspaces = await syncWorkspaces();
    if (availableWorkspaces) {
      removeUnavailableRecentWorkspaces(availableWorkspaces);
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
    actionLabel: currentWorkspaceId.value === '' ? 'No workspace open' : 'Open review tab',
    isDisabled: currentWorkspaceId.value === '',
    run: startReview
  };
});

const submitReviewCommand = computed<PaletteResult | undefined>(() => {
  if (!hasPendingReviewComments.value) {
    return undefined;
  }

  return {
    id: 'submit-review',
    label: 'Submit Review',
    actionLabel: 'Send comments',
    isDisabled: false,
    run: submitReview
  };
});

const repositoryActionLabel = computed(() =>
  unavailableCommandLabel(hasRepository.value ? 'Select' : 'Not a Git workspace')
);

const commandDefinitions = computed<PaletteResult[]>(() => [
  {
    id: 'open-workspace-picker',
    label: 'Open Workspace Picker',
    actionLabel: 'Open picker',
    isDisabled: false,
    run: () => emit('openWorkspacePicker')
  },
  {
    id: 'open-active-workspace-picker',
    label: 'Open Active Workspace Picker',
    actionLabel: 'Open picker',
    isDisabled: false,
    run: () => emit('openActiveWorkspacePicker')
  },
  {
    id: 'clone-remote-repository',
    label: 'Clone Remote Repository',
    actionLabel: 'Pick repository',
    isDisabled: false,
    run: () => emit('openRemoteRepositoryPicker')
  },
  reviewCommand.value,
  ...(submitReviewCommand.value ? [submitReviewCommand.value] : []),
  {
    id: 'close-workspace-terminals',
    label: 'Close Workspace',
    actionLabel: closeTerminalsActionLabel.value,
    isDisabled: currentWorkspaceId.value === '' || isClosingTerminals.value,
    run: () => {
      void closeCurrentWorkspaceTerminals();
    }
  },
  {
    id: 'create-open-worktree',
    label: 'Create/Open Worktree',
    actionLabel: repositoryActionLabel.value,
    isDisabled: !hasRepository.value,
    run: openCreateWorktree
  },
  {
    id: 'checkout-remote-worktree',
    label: 'Checkout Remote Branch as Worktree',
    actionLabel: repositoryActionLabel.value,
    isDisabled: !hasRepository.value,
    run: openRemoteWorktreePicker
  },
  {
    id: 'remove-worktree',
    label: 'Remove Worktree',
    actionLabel: repositoryActionLabel.value,
    isDisabled: !hasRepository.value,
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
    id: 'reload-settings',
    label: 'Reload Settings',
    actionLabel: 'Reload settings',
    isDisabled: false,
    run: () => {
      void reloadSettings();
    }
  },
  ...(settingsStore.settings.linear.enabled
    ? [
        createExternalCommand(
          'open-issue',
          'Open Issue',
          'Open issue',
          isWorkspaceDetailsLoading.value ? 'Loading' : unavailableCommandLabel('No issue found'),
          isWorkspaceDetailsLoading.value ? '' : (workspaceDetails.value?.links.issue?.url ?? '')
        )
      ]
    : []),
  createExternalCommand(
    'open-pr',
    'Open PR',
    'Open PR',
    unavailableCommandLabel('No PR found'),
    workspaceDetails.value?.links.pullRequest ?? ''
  ),
  createExternalCommand(
    'open-github-page',
    'Open GitHub Page',
    'Open page',
    unavailableCommandLabel('No GitHub remote'),
    workspaceDetails.value?.links.repository ?? ''
  )
]);

const { matchingItems: matchingCommands } = useFuzzyItems(commandDefinitions, query, (command) => command.label);

const paletteSummary = computed(() => {
  if (currentWorkspaceId.value === '') {
    return 'No workspace open';
  }

  return isWaitingForWorkspaceDetails.value ? `Loading ${currentWorkspaceId.value}` : currentWorkspaceId.value;
});

const paletteResults = computed<PaletteResult[]>(() =>
  matchingCommands.value.map((match) => ({
    ...match.item,
    id: `command:${match.item.id}`
  }))
);

const notice = computed<PaletteNotice | undefined>(() =>
  closeTerminalsError.value === ''
    ? undefined
    : {
        tone: 'error',
        title: 'Terminal close failed',
        messages: [closeTerminalsError.value]
      }
);
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
