<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { createWorktree, listRemoteBranches } from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';
import { useWorktreeNavigation } from '@/features/command-palette/composables/useWorktreeNavigation';
import type { RemoteBranch, Worktree } from '@/types/worktree';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import type { PaletteResult } from '@/features/command-palette/types';
import { usePaletteRequestState } from '@/features/command-palette/composables/usePaletteRequestState';

const props = defineProps<{
  projectName: string;
}>();

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const {
  closeReservedWorktreeTab,
  openWorktree: navigateToWorktree,
  reserveWorktreeTab
} = useWorktreeNavigation();
const remote = ref('');
const remoteBranches = ref<RemoteBranch[]>([]);
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

const paletteSummary = computed(() => {
  if (isLoading.value) {
    return `Fetching remote branches for ${props.projectName}`;
  }

  if (isCreating.value) {
    return `Creating worktree for ${props.projectName}`;
  }

  if (createdWorktree.value) {
    return createdWorktree.value.projectName;
  }

  const branchLabel = remoteBranches.value.length === 1 ? 'branch' : 'branches';
  return remote.value === ''
    ? props.projectName
    : `${remoteBranches.value.length} ${remote} ${branchLabel}`;
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
    const { remote: remoteName, branches } = await listRemoteBranches({ project: props.projectName });
    remote.value = remoteName;
    remoteBranches.value = branches;
  } catch (requestError) {
    setLoadError(requestError, 'Remote branches request failed');
  } finally {
    isLoading.value = false;
  }
};

const createOrOpenRemoteBranch = async (branch: RemoteBranch) => {
  if (isCreating.value) {
    return;
  }

  isCreating.value = true;
  clearActionError();
  createdWorktree.value = undefined;

  const reservedTab = reserveWorktreeTab();

  try {
    const { worktree } = await createWorktree({ project: props.projectName, branch: branch.name });
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

const remoteBranchActionLabel = (branch: RemoteBranch) => {
  if (isCreating.value) {
    return 'Creating';
  }

  if (branch.isCheckedOut) {
    return branch.worktreeProjectName === '' ? 'Open existing worktree' : `Open ${branch.worktreeProjectName}`;
  }

  if (branch.hasLocalBranch) {
    return 'Create from local branch';
  }

  return 'Checkout remote';
};

const { matchingItems: matchingRemoteBranches } = useFuzzyItems(
  remoteBranches,
  query,
  (branch) => branch.name
);

const paletteResults = computed<PaletteResult[]>(() => {
  if (createdWorktree.value && hasCopyWarnings.value) {
    return [{
      id: 'open-created-worktree',
      label: `Open ${createdWorktree.value.projectName}`,
      actionLabel: 'Open worktree',
      isDisabled: false,
      run: () => {
        if (createdWorktree.value) {
          void openWorktree(createdWorktree.value, reserveWorktreeTab());
        }
      }
    }];
  }

  return matchingRemoteBranches.value.map((match) => ({
    id: `remote-branch:${match.item.name}`,
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
