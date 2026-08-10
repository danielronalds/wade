<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { createRepositoryWorktree, listRepositoryBranches, type Branch, type Worktree } from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';
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

const { closeReservedWorktreeTab, openWorktree: navigateToWorktree, reserveWorktreeTab } = useWorktreeNavigation();
const remoteBranches = ref<Branch[]>([]);
const createdWorktree = ref<Worktree | undefined>();
const copyWarnings = computed(() => createdWorktree.value?.ignoredFileCopyWarnings ?? []);
const hasCopyWarnings = computed(() => copyWarnings.value.length > 0);
const {
  clearErrors,
  clearActionError,
  isActing: isCreating,
  isLoading,
  loadError,
  notice,
  query,
  setActionError,
  setLoadError,
  updateQuery
} = usePaletteRequestState({
  errorTitle: 'Worktree request failed',
  warningTitle: 'Worktree created with copy warnings',
  warningMessages: copyWarnings
});

const remoteName = computed(() => remoteBranches.value.find((branch) => branch.remote)?.remote ?? '');

const paletteSummary = computed(() => {
  if (isLoading.value) {
    return `Fetching remote branches for ${props.repositoryId}`;
  }

  if (isCreating.value) {
    return `Creating worktree for ${props.repositoryId}`;
  }

  if (createdWorktree.value) {
    return createdWorktree.value.workspaceId;
  }

  const branchLabel = remoteBranches.value.length === 1 ? 'branch' : 'branches';
  return remoteName.value === ''
    ? props.repositoryId
    : `${remoteBranches.value.length} ${remoteName.value} ${branchLabel}`;
});

const statusMessage = computed(() => {
  if (isLoading.value) {
    return 'Fetching remote branches';
  }

  if (loadError.value !== '' && remoteBranches.value.length === 0) {
    return 'Remote branches unavailable';
  }

  if (remoteBranches.value.length === 0) {
    return 'No remote branches found';
  }

  return 'No matching remote branches';
});

const openWorktree = async (worktree: Worktree, reservedTab?: Window) => {
  await navigateToWorktree(worktree, reservedTab);
  emit('close', false);
};

const loadRemoteBranches = async () => {
  isLoading.value = true;
  clearErrors();

  try {
    const { items } = await listRepositoryBranches(props.repositoryId, { kind: 'remote' });
    remoteBranches.value = items;
  } catch (requestError) {
    setLoadError(requestError, 'Remote branches request failed');
  } finally {
    isLoading.value = false;
  }
};

const createOrOpenRemoteBranch = async (branch: Branch) => {
  if (isCreating.value) {
    return;
  }

  if (branch.checkedOutWorkspaceId) {
    await navigateToWorktree({ workspaceId: branch.checkedOutWorkspaceId }, reserveWorktreeTab());
    emit('close', false);
    return;
  }

  isCreating.value = true;
  clearActionError();
  createdWorktree.value = undefined;

  const reservedTab = reserveWorktreeTab();

  try {
    const worktree = await createRepositoryWorktree(props.repositoryId, { branchRef: branch.ref });
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

const remoteBranchActionLabel = (branch: Branch) => {
  if (isCreating.value) {
    return 'Creating';
  }

  if (branch.checkedOutWorkspaceId) {
    return `Open ${branch.checkedOutWorkspaceId}`;
  }

  if (branch.hasLocalBranch) {
    return 'Create from local branch';
  }

  return 'Checkout remote';
};

const { matchingItems: matchingRemoteBranches } = useFuzzyItems(remoteBranches, query, (branch) => branch.name);

const paletteResults = computed<PaletteResult[]>(() => {
  if (createdWorktree.value && hasCopyWarnings.value) {
    return [
      {
        id: 'open-created-worktree',
        label: `Open ${createdWorktree.value.workspaceId}`,
        actionLabel: 'Open worktree',
        isDisabled: false,
        run: () => {
          if (createdWorktree.value) {
            void openWorktree(createdWorktree.value, reserveWorktreeTab());
          }
        }
      }
    ];
  }

  return matchingRemoteBranches.value.map((match) => ({
    id: `remote-branch:${match.item.ref}`,
    label: match.item.name,
    actionLabel: remoteBranchActionLabel(match.item),
    isDisabled: isLoading.value || isCreating.value,
    run: () => {
      void createOrOpenRemoteBranch(match.item);
    }
  }));
});

onMounted(() => {
  void loadRemoteBranches();
});
</script>

<template>
  <PaletteShell
    title="Checkout Remote Branch"
    :summary="paletteSummary"
    :query="query"
    search-placeholder="Search remote branches"
    results-aria-label="Remote branches"
    :status-message="statusMessage"
    :results="paletteResults"
    :notice="notice"
    @update:query="updateQuery"
    @close="emit('close', $event)"
  />
</template>
