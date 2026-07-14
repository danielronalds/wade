import { computed, nextTick, onBeforeUnmount, onMounted, reactive, readonly, ref } from 'vue';
import { useSettingsStore } from '@/stores/useSettingsStore';
import { applyThemeAccentColor, type ThemeAccentColor } from '@/utils/theme';
import {
  cloneSettings,
  createEmptySettings,
  isValidAgents,
  isValidProjectDirectory,
  isValidShell,
  normaliseSettings,
  type Settings
} from '@/types/settings';

const settingsHaveChanged = (current: Settings, saved: Settings) => JSON.stringify(current.projectDirectories)
  !== JSON.stringify(saved.projectDirectories)
  || current.shell !== saved.shell
  || JSON.stringify(current.agents) !== JSON.stringify(saved.agents)
  || current.copyIgnoredFilesOnWorktreeCreation !== saved.copyIgnoredFilesOnWorktreeCreation
  || current.openWorktreesInNewTabs !== saved.openWorktreesInNewTabs
  || JSON.stringify(current.worktreeCopyExcludes) !== JSON.stringify(saved.worktreeCopyExcludes)
  || current.themeAccentColor !== saved.themeAccentColor;

const inputValue = (event: Event) => event.target instanceof HTMLInputElement
  ? event.target.value
  : undefined;

const errorMessage = (error: unknown, fallback: string) => error instanceof Error
  ? error.message
  : fallback;

export const useSettingsForm = () => {
  const {
    loadSettings: loadSavedSettings,
    saveSettings
  } = useSettingsStore();

  const form = reactive<Settings>(createEmptySettings());
  const savedSettings = ref<Settings>(createEmptySettings());
  const isLoading = ref(false);
  const isSaving = ref(false);
  const error = ref('');
  const statusMessage = ref('');

  const normalisedSettings = computed(() => normaliseSettings(form));
  const hasInvalidProjectDirectories = computed(() => form.projectDirectories.some(
    (directory) => !isValidProjectDirectory(directory)
  ));
  const hasInvalidShell = computed(() => !isValidShell(form.shell));
  const hasInvalidAgents = computed(() => !isValidAgents(normalisedSettings.value.agents));
  const hasChanges = computed(() => settingsHaveChanged(normalisedSettings.value, savedSettings.value));
  const canSave = computed(() => !isLoading.value
    && !isSaving.value
    && hasChanges.value
    && !hasInvalidProjectDirectories.value
    && !hasInvalidShell.value
    && !hasInvalidAgents.value);

  const clearMessages = () => {
    statusMessage.value = '';
    error.value = '';
  };

  const replaceForm = (settings: Settings) => {
    form.projectDirectories = [...settings.projectDirectories];
    form.shell = settings.shell;
    form.agents = settings.agents.map((agent) => ({ ...agent }));
    form.copyIgnoredFilesOnWorktreeCreation = settings.copyIgnoredFilesOnWorktreeCreation;
    form.openWorktreesInNewTabs = settings.openWorktreesInNewTabs;
    form.worktreeCopyExcludes = [...settings.worktreeCopyExcludes];
    form.themeAccentColor = settings.themeAccentColor;
  };

  const loadSettings = async () => {
    isLoading.value = true;
    clearMessages();

    try {
      const settings = await loadSavedSettings({ force: true });
      replaceForm(settings);
      savedSettings.value = normaliseSettings(settings);
      applyThemeAccentColor(settings.themeAccentColor);
    } catch (requestError) {
      error.value = errorMessage(requestError, 'Settings request failed');
    } finally {
      isLoading.value = false;
    }
  };

  const updateProjectDirectory = (index: number, event: Event) => {
    const nextDirectory = inputValue(event);
    if (nextDirectory === undefined) {
      return;
    }

    form.projectDirectories = form.projectDirectories.map((directory, directoryIndex) => (
      directoryIndex === index ? nextDirectory : directory
    ));
    clearMessages();
  };

  const addProjectDirectory = async () => {
    form.projectDirectories = [...form.projectDirectories, ''];
    clearMessages();

    await nextTick();
    document.getElementById(`project-directory-${form.projectDirectories.length - 1}`)?.focus();
  };

  const removeProjectDirectory = (index: number) => {
    form.projectDirectories = form.projectDirectories.filter((_, directoryIndex) => directoryIndex !== index);
    clearMessages();
  };

  const updateShell = (event: Event) => {
    const nextShell = inputValue(event);
    if (nextShell === undefined) {
      return;
    }

    form.shell = nextShell;
    clearMessages();
  };

  const updateAgentName = (index: number, event: Event) => {
    const nextName = inputValue(event);
    if (nextName === undefined) {
      return;
    }

    form.agents = form.agents.map((agent, agentIndex) => (
      agentIndex === index ? { ...agent, name: nextName } : agent
    ));
    clearMessages();
  };

  const updateAgentCommand = (index: number, event: Event) => {
    const nextCommand = inputValue(event);
    if (nextCommand === undefined) {
      return;
    }

    form.agents = form.agents.map((agent, agentIndex) => (
      agentIndex === index ? { ...agent, command: nextCommand } : agent
    ));
    clearMessages();
  };

  const setDefaultAgent = (index: number) => {
    form.agents = form.agents.map((agent, agentIndex) => ({
      ...agent,
      default: agentIndex === index
    }));
    clearMessages();
  };

  const addAgent = async () => {
    form.agents = [...form.agents, { name: '', command: '', default: false }];
    clearMessages();

    await nextTick();
    document.getElementById(`agent-name-${form.agents.length - 1}`)?.focus();
  };

  const removeAgent = (index: number) => {
    if (form.agents.length <= 1) {
      return;
    }

    const removedAgentWasDefault = form.agents[index]?.default ?? false;
    const remainingAgents = form.agents.filter((_, agentIndex) => agentIndex !== index);
    form.agents = removedAgentWasDefault
      ? remainingAgents.map((agent, agentIndex) => ({ ...agent, default: agentIndex === 0 }))
      : remainingAgents;
    clearMessages();
  };

  const updateCopyIgnoredFilesOnWorktreeCreation = (shouldCopyIgnoredFiles: boolean) => {
    form.copyIgnoredFilesOnWorktreeCreation = shouldCopyIgnoredFiles;
    clearMessages();
  };

  const updateOpenWorktreesInNewTabs = (shouldOpenWorktreesInNewTabs: boolean) => {
    form.openWorktreesInNewTabs = shouldOpenWorktreesInNewTabs;
    clearMessages();
  };

  const updateThemeAccentColor = (themeAccentColor: ThemeAccentColor) => {
    form.themeAccentColor = themeAccentColor;
    applyThemeAccentColor(themeAccentColor);
    clearMessages();
  };

  const updateWorktreeCopyExclude = (index: number, event: Event) => {
    const nextExclude = inputValue(event);
    if (nextExclude === undefined) {
      return;
    }

    form.worktreeCopyExcludes = form.worktreeCopyExcludes.map((exclude, excludeIndex) => (
      excludeIndex === index ? nextExclude : exclude
    ));
    clearMessages();
  };

  const addWorktreeCopyExclude = async () => {
    form.worktreeCopyExcludes = [...form.worktreeCopyExcludes, ''];
    clearMessages();

    await nextTick();
    document.getElementById(`worktree-copy-exclude-${form.worktreeCopyExcludes.length - 1}`)?.focus();
  };

  const removeWorktreeCopyExclude = (index: number) => {
    form.worktreeCopyExcludes = form.worktreeCopyExcludes.filter((_, excludeIndex) => excludeIndex !== index);
    clearMessages();
  };

  const persistSettings = async () => {
    const settings = await saveSettings(cloneSettings(normalisedSettings.value));

    replaceForm(settings);
    savedSettings.value = cloneSettings(settings);
    applyThemeAccentColor(settings.themeAccentColor);
    statusMessage.value = 'Settings saved';
  };

  const submit = async () => {
    if (!canSave.value) {
      return;
    }

    isSaving.value = true;
    clearMessages();

    try {
      await persistSettings();
    } catch (saveError) {
      error.value = errorMessage(saveError, 'Settings save failed');
    } finally {
      isSaving.value = false;
    }
  };

  onMounted(() => {
    void loadSettings();
  });

  onBeforeUnmount(() => {
    if (normalisedSettings.value.themeAccentColor !== savedSettings.value.themeAccentColor) {
      applyThemeAccentColor(savedSettings.value.themeAccentColor);
    }
  });

  return {
    form,
    isLoading: readonly(isLoading),
    isSaving: readonly(isSaving),
    error: readonly(error),
    statusMessage: readonly(statusMessage),
    hasInvalidProjectDirectories,
    hasInvalidShell,
    hasInvalidAgents,
    canSave,
    isValidProjectDirectory,
    updateProjectDirectory,
    addProjectDirectory,
    removeProjectDirectory,
    updateShell,
    updateAgentName,
    updateAgentCommand,
    setDefaultAgent,
    addAgent,
    removeAgent,
    updateCopyIgnoredFilesOnWorktreeCreation,
    updateOpenWorktreesInNewTabs,
    updateThemeAccentColor,
    updateWorktreeCopyExclude,
    addWorktreeCopyExclude,
    removeWorktreeCopyExclude,
    submit
  };
};
