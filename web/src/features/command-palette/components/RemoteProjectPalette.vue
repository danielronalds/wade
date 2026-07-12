<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
  cloneRemoteProject,
  listRemoteProjects
} from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';
import { useProjects } from '@/features/projects/composables/useProjects';
import { useSettingsStore } from '@/stores/useSettingsStore';
import type { RemoteProject } from '@/types/remoteProject';
import PaletteShell from '@/features/command-palette/components/PaletteShell.vue';
import type { PaletteResult } from '@/features/command-palette/types';
import { usePaletteRequestState } from '@/features/command-palette/composables/usePaletteRequestState';

type DirectoryChoice = {
  index: number;
  directory: string;
};

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
}>();

const router = useRouter();
const { syncProjects } = useProjects();
const { loadSettings } = useSettingsStore();
const remoteProjects = ref<RemoteProject[]>([]);
const projectDirectories = ref<string[]>([]);
const selectedRemoteProject = ref<RemoteProject | undefined>();
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
  errorTitle: 'Remote project request failed'
});

const directoryChoices = computed<DirectoryChoice[]>(() => projectDirectories.value.map((directory, index) => ({
  index,
  directory
})));

const isChoosingDirectory = computed(() => Boolean(selectedRemoteProject.value));

const directoryLabel = (choice: DirectoryChoice) => {
  const directory = choice.directory.trim();

  return directory === '' ? `Project directory ${choice.index + 1}` : directory;
};

const paletteSummary = computed(() => {
  if (isLoading.value) {
    return 'Fetching GitHub projects';
  }

  if (isCloning.value) {
    return selectedRemoteProject.value
      ? `Cloning ${selectedRemoteProject.value.nameWithOwner}`
      : 'Cloning GitHub project';
  }

  if (selectedRemoteProject.value) {
    return `Clone ${selectedRemoteProject.value.nameWithOwner}`;
  }

  const label = remoteProjects.value.length === 1 ? 'project' : 'projects';
  return `${remoteProjects.value.length} GitHub ${label}`;
});

const statusMessage = computed(() => {
  if (isLoading.value) {
    return 'Fetching GitHub projects';
  }

  if (loadError.value !== '' && remoteProjects.value.length === 0) {
    return 'GitHub projects unavailable';
  }

  if (isChoosingDirectory.value && directoryChoices.value.length === 0) {
    return 'No project directories configured';
  }

  if (isChoosingDirectory.value) {
    return 'No matching project directories';
  }

  if (remoteProjects.value.length === 0) {
    return 'No GitHub projects found';
  }

  return 'No matching GitHub projects';
});

const searchPlaceholder = computed(() => isChoosingDirectory.value
  ? 'Search project directories'
  : 'Search GitHub projects');

const resultsAriaLabel = computed(() => isChoosingDirectory.value
  ? 'Project directories'
  : 'GitHub projects');

const openProject = async (projectName: string, syncFirst = false) => {
  if (syncFirst) {
    await syncProjects();
  }

  const currentRoute = router.currentRoute.value;
  const currentProjectName = String(currentRoute.params.projectName ?? '');
  if (currentRoute.name === 'project' && currentProjectName === projectName) {
    emit('close');
    return;
  }

  emit('close', false);
  await router.push({ name: 'project', params: { projectName } });
};

const openLocalProject = async (project: RemoteProject) => {
  if (!project.isLocal || project.localName === '') {
    return;
  }

  await openProject(project.localName);
};

const cloneProject = async (project: RemoteProject, directoryIndex: number) => {
  if (isCloning.value || project.isLocal) {
    return;
  }

  isCloning.value = true;
  clearActionError();

  try {
    const { project: clonedProject } = await cloneRemoteProject({ nameWithOwner: project.nameWithOwner, directoryIndex });
    await openProject(clonedProject.name, true);
  } catch (requestError) {
    setActionError(requestError, 'Remote project clone failed');
  } finally {
    isCloning.value = false;
  }
};

const selectRemoteProject = async (project: RemoteProject) => {
  if (project.isLocal) {
    await openLocalProject(project);
    return;
  }

  if (projectDirectories.value.length === 0) {
    return;
  }

  if (projectDirectories.value.length === 1) {
    await cloneProject(project, 0);
    return;
  }

  selectedRemoteProject.value = project;
  query.value = '';
  clearActionError();
};

const loadRemoteProjects = async () => {
  isLoading.value = true;
  clearErrors();

  try {
    const [{ projects }, settings] = await Promise.all([
      listRemoteProjects(),
      loadSettings({ force: true })
    ]);

    remoteProjects.value = projects;
    projectDirectories.value = settings.projectDirectories;
  } catch (requestError) {
    setLoadError(requestError, 'Remote projects request failed');
  } finally {
    isLoading.value = false;
  }
};

const remoteProjectActionLabel = (project: RemoteProject) => {
  if (isCloning.value) {
    return 'Cloning';
  }

  if (project.isLocal) {
    return project.localName === '' ? 'Already local' : 'Open local';
  }

  if (projectDirectories.value.length === 0) {
    return 'No project directory';
  }

  return projectDirectories.value.length === 1 ? 'Clone project' : 'Choose location';
};

const isRemoteProjectDisabled = (project: RemoteProject) => isLoading.value
  || isCloning.value
  || (project.isLocal && project.localName === '')
  || (!project.isLocal && projectDirectories.value.length === 0);

const { matchingItems: matchingRemoteProjects } = useFuzzyItems(
  remoteProjects,
  query,
  (project) => project.nameWithOwner
);

const { matchingItems: matchingDirectoryChoices } = useFuzzyItems(
  directoryChoices,
  query,
  directoryLabel
);

const remoteProjectResults = computed<PaletteResult[]>(() => matchingRemoteProjects.value.map((match) => ({
  id: `remote-project:${match.item.nameWithOwner}`,
  label: match.item.nameWithOwner,
  actionLabel: remoteProjectActionLabel(match.item),
  isDisabled: isRemoteProjectDisabled(match.item),
  run: () => {
    void selectRemoteProject(match.item);
  }
})));

const directoryResults = computed<PaletteResult[]>(() => matchingDirectoryChoices.value.map((match) => ({
  id: `remote-project-directory:${match.item.index}`,
  label: directoryLabel(match.item),
  actionLabel: isCloning.value ? 'Cloning' : 'Clone here',
  isDisabled: isCloning.value,
  run: () => {
    if (selectedRemoteProject.value) {
      void cloneProject(selectedRemoteProject.value, match.item.index);
    }
  }
})));

const paletteResults = computed<PaletteResult[]>(() => isChoosingDirectory.value
  ? directoryResults.value
  : remoteProjectResults.value);

onMounted(() => {
  void loadRemoteProjects();
});
</script>

<template>
  <PaletteShell
    title="Clone Remote Project"
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
