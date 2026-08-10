<script setup lang="ts">
import { computed, onMounted, ref, type DeepReadonly } from 'vue';
import { useRouter } from 'vue-router';
import { deleteRepositoryWorktree, listRepositoryWorktrees, type Worktree } from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import { usePaletteRequestState } from '@/features/command-palette/composables/usePaletteRequestState';
import type { PaletteResult } from '@/features/command-palette/types';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';

const props = defineProps<{
  repositoryId: string;
  workspaceId: string;
}>();

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { syncWorkspaces } = useWorkspaces();
const workspaceSessionStore = useWorkspaceSessionStore();
const worktrees = ref<Worktree[]>([]);
const targetWorktree = ref<DeepReadonly<Worktree> | undefined>();
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
const mainWorktree = computed(() => worktrees.value.find((worktree) => worktree.isMain));

const paletteSummary = computed(() => {
  if (isLoading.value) {
    return `Loading worktrees for ${props.repositoryId}`;
  }

  if (isRemoving.value) {
    return 'Removing worktree';
  }

  if (targetWorktree.value) {
    return targetWorktree.value.workspaceId;
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
    const { items } = await listRepositoryWorktrees(props.repositoryId);
    worktrees.value = items;
  } catch (requestError) {
    setLoadError(requestError, 'Worktrees request failed');
  } finally {
    isLoading.value = false;
  }
};

const navigateAfterRemovingCurrentWorktree = async () => {
  emit('close', false);

  if (mainWorktree.value) {
    await router.push({ name: 'workspace', params: { workspaceId: mainWorktree.value.workspaceId } });
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
    await deleteRepositoryWorktree(props.repositoryId, target.id);
    workspaceSessionStore.clearWorkspaceSession(target.workspaceId);
    await syncWorkspaces();

    if (target.workspaceId === props.workspaceId) {
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

const selectWorktree = (worktree: DeepReadonly<Worktree>) => {
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
  (worktree) => `${worktree.workspaceId} ${worktree.name} ${worktree.branch?.name ?? ''}`
);

const paletteResults = computed<PaletteResult[]>(() => {
  if (targetWorktree.value) {
    const branchName = targetWorktree.value.branch?.name ?? '';
    return [
      {
        id: 'confirm-remove-worktree',
        label:
          branchName === ''
            ? `Remove ${targetWorktree.value.workspaceId}`
            : `Remove ${targetWorktree.value.workspaceId} and local branch ${branchName}`,
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
    id: `worktree:${match.item.id}`,
    label: match.item.workspaceId,
    actionLabel: match.item.branch?.name ?? 'Remove worktree',
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
