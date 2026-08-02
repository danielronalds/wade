<script setup lang="ts">
import { computed, ref } from 'vue';
import { createRepositoryWorktree, type Worktree } from '@/api/generated/wade';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import { usePaletteRequestState } from '@/features/command-palette/composables/usePaletteRequestState';
import { useWorktreeNavigation } from '@/features/command-palette/composables/useWorktreeNavigation';
import type { PaletteResult } from '@/features/command-palette/types';

const props = defineProps<{
  repositoryId: string;
}>();

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const {
  closeReservedWorktreeTab,
  openWorktree: navigateToWorktree,
  reserveWorktreeTab
} = useWorktreeNavigation();
const createdWorktree = ref<Worktree | undefined>();
const copyWarnings = computed(() => createdWorktree.value?.ignoredFileCopyWarnings ?? []);
const hasCopyWarnings = computed(() => copyWarnings.value.length > 0);
const {
  clearActionError,
  isActing: isCreating,
  notice,
  query,
  setActionError,
  updateQuery
} = usePaletteRequestState({
  errorTitle: 'Worktree request failed',
  warningTitle: 'Worktree created with copy warnings',
  warningMessages: copyWarnings
});

const branchName = computed(() => query.value.trim());

const paletteSummary = computed(() => {
  if (isCreating.value) {
    return `Creating worktree for ${props.repositoryId}`;
  }

  return createdWorktree.value?.workspaceId ?? props.repositoryId;
});

const openWorktree = async (worktree: Worktree, reservedTab?: Window) => {
  await navigateToWorktree(worktree, reservedTab);
  emit('close', false);
};

const createOrOpenWorktree = async () => {
  if (branchName.value === '' || isCreating.value) {
    return;
  }

  isCreating.value = true;
  clearActionError();
  createdWorktree.value = undefined;

  const reservedTab = reserveWorktreeTab();

  try {
    const worktree = await createRepositoryWorktree(props.repositoryId, { branchRef: branchName.value });
    if ((worktree.ignoredFileCopyWarnings?.length ?? 0) > 0) {
      closeReservedWorktreeTab(reservedTab);
      createdWorktree.value = worktree;
      query.value = '';
      return;
    }

    await openWorktree(worktree, reservedTab);
  } catch (requestError) {
    closeReservedWorktreeTab(reservedTab);
    setActionError(requestError, 'Worktree request failed');
  } finally {
    isCreating.value = false;
  }
};

const paletteResults = computed<PaletteResult[]>(() => {
  if (createdWorktree.value && hasCopyWarnings.value) {
    return [{
      id: 'open-created-worktree',
      label: `Open ${createdWorktree.value.workspaceId}`,
      actionLabel: 'Open worktree',
      isDisabled: false,
      run: () => {
        if (createdWorktree.value) {
          void openWorktree(createdWorktree.value, reserveWorktreeTab());
        }
      }
    }];
  }

  if (branchName.value === '') {
    return [];
  }

  return [{
    id: 'create-worktree',
    label: branchName.value,
    actionLabel: isCreating.value ? 'Creating' : 'Create or open worktree',
    isDisabled: isCreating.value,
    run: () => {
      void createOrOpenWorktree();
    }
  }];
});
</script>

<template>
  <PaletteShell
    title="Create/Open Worktree"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Type a branch name"
    results-aria-label="Worktree branch actions"
    status-message="Type a branch name"
    :results="paletteResults"
    :notice="notice"
    @update:query="updateQuery"
    @close="emit('close', $event)"
  />
</template>
