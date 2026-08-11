<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import { computed, onMounted, ref, type DeepReadonly } from 'vue';
import { useRouter } from 'vue-router';
import { listRemoteRepositories, materialiseWorkspace, type RemoteRepository } from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import { usePaletteRequestState } from '@/features/command-palette/composables/usePaletteRequestState';
import type { PaletteResult } from '@/features/command-palette/types';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';
import { useSettingsStore } from '@/stores/useSettingsStore';

type DirectoryChoice = {
  index: number;
  directory: string;
};

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { syncWorkspaces } = useWorkspaces();
const { loadSettings } = useSettingsStore();
const remoteRepositories = ref<RemoteRepository[]>([]);
const workspaceDirectories = ref<string[]>([]);
const selectedRemoteRepository = ref<DeepReadonly<RemoteRepository> | undefined>();
const {
  clearActionError,
  clearErrors,
  isActing: isCloning,
  isLoading,
  loadError,
  notice,
  query,
  setActionError,
  setLoadError,
  updateQuery
} = usePaletteRequestState({
  errorTitle: 'Remote repository request failed'
});

const directoryChoices = computed<DirectoryChoice[]>(() =>
  workspaceDirectories.value.map((directory, index) => ({
    index,
    directory
  }))
);

const isChoosingDirectory = computed(() => Boolean(selectedRemoteRepository.value));

const directoryLabel = (choice: DirectoryChoice) => {
  const directory = choice.directory.trim();

  return directory === '' ? `Workspace directory ${choice.index + 1}` : directory;
};

const paletteSummary = computed(() => {
  if (isLoading.value) {
    return 'Fetching GitHub repositories';
  }

  if (isCloning.value) {
    return selectedRemoteRepository.value
      ? `Cloning ${selectedRemoteRepository.value.id}`
      : 'Cloning GitHub repository';
  }

  if (selectedRemoteRepository.value) {
    return `Clone ${selectedRemoteRepository.value.id}`;
  }

  const label = remoteRepositories.value.length === 1 ? 'repository' : 'repositories';
  return `${remoteRepositories.value.length} GitHub ${label}`;
});

const statusMessage = computed(() => {
  if (isLoading.value) {
    return 'Fetching GitHub repositories';
  }

  if (loadError.value !== '' && remoteRepositories.value.length === 0) {
    return 'GitHub repositories unavailable';
  }

  if (isChoosingDirectory.value && directoryChoices.value.length === 0) {
    return 'No workspace directories configured';
  }

  if (isChoosingDirectory.value) {
    return 'No matching workspace directories';
  }

  if (remoteRepositories.value.length === 0) {
    return 'No GitHub repositories found';
  }

  return 'No matching GitHub repositories';
});

const searchPlaceholder = computed(() =>
  isChoosingDirectory.value ? 'Search workspace directories' : 'Search GitHub repositories'
);

const resultsAriaLabel = computed(() => (isChoosingDirectory.value ? 'Workspace directories' : 'GitHub repositories'));

const openWorkspace = async (workspaceId: string, syncFirst = false) => {
  if (syncFirst) {
    await syncWorkspaces();
  }

  const currentRoute = router.currentRoute.value;
  const currentWorkspaceId = String(currentRoute.params.workspaceId ?? '');
  if (currentRoute.name === 'workspace' && currentWorkspaceId === workspaceId) {
    emit('close');
    return;
  }

  emit('close', false);
  await router.push({ name: 'workspace', params: { workspaceId } });
};

const cloneRepository = async (repository: DeepReadonly<RemoteRepository>, workspaceDirectory: string) => {
  if (isCloning.value || repository.localWorkspaceIds.length > 0) {
    return;
  }

  isCloning.value = true;
  clearActionError();

  try {
    const workspace = await materialiseWorkspace({
      remoteRepositoryId: repository.id,
      workspaceDirectory
    });
    await openWorkspace(workspace.id, true);
  } catch (requestError) {
    setActionError(requestError, 'Remote repository clone failed');
  } finally {
    isCloning.value = false;
  }
};

const selectRemoteRepository = async (repository: DeepReadonly<RemoteRepository>) => {
  if (repository.localWorkspaceIds.length > 0) {
    await openWorkspace(repository.localWorkspaceIds[0]!);
    return;
  }

  if (workspaceDirectories.value.length === 0) {
    return;
  }

  if (workspaceDirectories.value.length === 1) {
    await cloneRepository(repository, workspaceDirectories.value[0]!);
    return;
  }

  selectedRemoteRepository.value = repository;
  query.value = '';
  clearActionError();
};

const loadRemoteRepositories = async () => {
  isLoading.value = true;
  clearErrors();

  try {
    const [{ items }, settings] = await Promise.all([listRemoteRepositories(), loadSettings({ force: true })]);

    remoteRepositories.value = items;
    workspaceDirectories.value = settings.workspaceDirectories;
  } catch (requestError) {
    setLoadError(requestError, 'Remote repositories request failed');
  } finally {
    isLoading.value = false;
  }
};

const remoteRepositoryActionLabel = (repository: DeepReadonly<RemoteRepository>) => {
  if (isCloning.value) {
    return 'Cloning';
  }

  if (repository.localWorkspaceIds.length > 0) {
    return 'Open local workspace';
  }

  if (workspaceDirectories.value.length === 0) {
    return 'No workspace directory';
  }

  return workspaceDirectories.value.length === 1 ? 'Clone repository' : 'Choose location';
};

const isRemoteRepositoryDisabled = (repository: DeepReadonly<RemoteRepository>) =>
  isLoading.value ||
  isCloning.value ||
  (repository.localWorkspaceIds.length === 0 && workspaceDirectories.value.length === 0);

const { matchingItems: matchingRemoteRepositories } = useFuzzyItems(
  remoteRepositories,
  query,
  (repository) => repository.id
);

const { matchingItems: matchingDirectoryChoices } = useFuzzyItems(directoryChoices, query, directoryLabel);

const remoteRepositoryResults = computed<PaletteResult[]>(() =>
  matchingRemoteRepositories.value.map((match) => ({
    id: `remote-repository:${match.item.id}`,
    label: match.item.id,
    actionLabel: remoteRepositoryActionLabel(match.item),
    isDisabled: isRemoteRepositoryDisabled(match.item),
    run: () => {
      void selectRemoteRepository(match.item);
    }
  }))
);

const directoryResults = computed<PaletteResult[]>(() =>
  matchingDirectoryChoices.value.map((match) => ({
    id: `remote-workspace-directory:${match.item.index}`,
    label: directoryLabel(match.item),
    actionLabel: isCloning.value ? 'Cloning' : 'Clone here',
    isDisabled: isCloning.value,
    run: () => {
      if (selectedRemoteRepository.value) {
        void cloneRepository(selectedRemoteRepository.value, match.item.directory);
      }
    }
  }))
);

const paletteResults = computed<PaletteResult[]>(() =>
  isChoosingDirectory.value ? directoryResults.value : remoteRepositoryResults.value
);

onMounted(() => {
  void loadRemoteRepositories();
});
</script>

<template>
  <PaletteShell
    title="Clone Remote Repository"
    :summary="paletteSummary"
    :query="query"
    :search-placeholder="searchPlaceholder"
    :results-aria-label="resultsAriaLabel"
    :status-message="statusMessage"
    :results="paletteResults"
    :notice="notice"
    @update:query="updateQuery"
    @close="emit('close', $event)"
  />
</template>
