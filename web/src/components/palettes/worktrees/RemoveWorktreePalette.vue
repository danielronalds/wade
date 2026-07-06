<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { listWorktrees, removeWorktree as requestRemoveWorktree } from '../../../api/worktrees';
import { useFuzzyItems } from '../../../composables/useFuzzyProjects';
import { useProjects } from '../../../composables/useProjects';
import type { Worktree } from '../../../types/worktree';
import PaletteShell from '../PaletteShell.vue';
import type { PaletteResult } from '../types';
import { usePaletteRequestState } from '../composables/usePaletteRequestState';

const props = defineProps<{
  projectName: string;
}>();

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { syncProjects } = useProjects();
const worktrees = ref<Worktree[]>([]);
const targetWorktree = ref<Worktree | undefined>();
const {
  clearErrors,
  clearActionError,
  isActing: isRemoving,
  isLoading,
  loadError,
  notice,
  query,
  setActionError,
  setLoadError,
  updateQuery
} = usePaletteRequestState({
  errorTitle: 'Worktree removal failed'
});

const removableWorktrees = computed(() => worktrees.value.filter((worktree) => worktree.isRemovable));
const baseWorktree = computed(() => worktrees.value.find((worktree) => worktree.isBase));

const paletteSummary = computed(() => {
  if (isLoading.value) {
    return `Loading worktrees for ${props.projectName}`;
  }

  if (isRemoving.value) {
    return 'Removing worktree';
  }

  if (targetWorktree.value) {
    return targetWorktree.value.projectName;
  }

  const worktreeLabel = removableWorktrees.value.length === 1 ? 'worktree' : 'worktrees';
  return `${removableWorktrees.value.length} removable ${worktreeLabel}`;
});

const statusMessage = computed(() => {
  if (isLoading.value) {
    return 'Loading worktrees';
  }

  if (loadError.value !== '' && removableWorktrees.value.length === 0) {
    return 'Worktrees unavailable';
  }

  if (removableWorktrees.value.length === 0) {
    return 'No removable worktrees';
  }

  return 'No matching worktrees';
});

const loadWorktrees = async () => {
  isLoading.value = true;
  clearErrors();

  try {
    worktrees.value = await listWorktrees(props.projectName);
  } catch (requestError) {
    setLoadError(requestError, 'Worktrees request failed');
  } finally {
    isLoading.value = false;
  }
};

const navigateAfterRemovingCurrentWorktree = async () => {
  emit('close', false);

  if (baseWorktree.value) {
    await router.push({ name: 'project', params: { projectName: baseWorktree.value.projectName } });
    return;
  }

  await router.push({ name: 'home' });
};

const removeSelectedWorktree = async () => {
  if (!targetWorktree.value || isRemoving.value) {
    return;
  }

  const target = targetWorktree.value;
  isRemoving.value = true;
  clearActionError();

  try {
    await requestRemoveWorktree(props.projectName, target.projectName);
    await syncProjects();

    if (target.isCurrent) {
      await navigateAfterRemovingCurrentWorktree();
      return;
    }

    emit('close');
  } catch (requestError) {
    setActionError(requestError, 'Worktree removal failed');
  } finally {
    isRemoving.value = false;
  }
};

const selectWorktree = (worktree: Worktree) => {
  targetWorktree.value = worktree;
  query.value = '';
  clearActionError();
};

const cancelRemoval = () => {
  targetWorktree.value = undefined;
  clearActionError();
};

const { matchingItems: matchingWorktrees } = useFuzzyItems(
  removableWorktrees,
  query,
  (worktree) => `${worktree.projectName} ${worktree.name} ${worktree.branch}`
);

const paletteResults = computed<PaletteResult[]>(() => {
  if (targetWorktree.value) {
    return [
      {
        id: 'confirm-remove-worktree',
        label: targetWorktree.value.branch === ''
          ? `Remove ${targetWorktree.value.projectName}`
          : `Remove ${targetWorktree.value.projectName} and local branch ${targetWorktree.value.branch}`,
        actionLabel: isRemoving.value ? 'Removing' : 'Confirm remove',
        isDisabled: isRemoving.value,
        run: () => {
          void removeSelectedWorktree();
        }
      },
      {
        id: 'cancel-remove-worktree',
        label: 'Cancel',
        actionLabel: 'Back to worktrees',
        isDisabled: isRemoving.value,
        run: cancelRemoval
      }
    ];
  }

  return matchingWorktrees.value.map((match) => ({
    id: `worktree:${match.item.projectName}`,
    label: match.item.projectName,
    actionLabel: match.item.branch === '' ? 'Remove worktree' : match.item.branch,
    isDisabled: isLoading.value,
    run: () => selectWorktree(match.item)
  }));
});

onMounted(() => {
  void loadWorktrees();
});
</script>

<template>
  <PaletteShell
    title="Remove Worktree"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Search worktrees"
    results-aria-label="Removable worktrees"
    :status-message="statusMessage"
    :results="paletteResults"
    :notice="notice"
    @update:query="updateQuery"
    @close="emit('close', $event)"
  />
</template>
