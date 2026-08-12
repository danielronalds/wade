import { defineStore } from 'pinia';
import { computed, reactive, readonly } from 'vue';
import { getSettings, reloadSettings, updateSettings } from '@/api/generated/wade';
import { useRecentWorkspaces } from '@/features/workspaces/composables/useRecentWorkspaces';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';
import { useWorkspaceDetailsStore } from '@/stores/useWorkspaceDetailsStore';
import { cloneSettings, createEmptySettings, normaliseSettings, type Settings } from '@/types/settings';
import { applyThemeAccentColor } from '@/utils/theme';

const errorMessage = (error: unknown, fallback: string) => (error instanceof Error ? error.message : fallback);

export const useSettingsStore = defineStore('settings', () => {
  const { syncWorkspaces } = useWorkspaces();
  const { removeUnavailableRecentWorkspaces } = useRecentWorkspaces();
  const workspaceDetailsStore = useWorkspaceDetailsStore();

  const state = reactive({
    hasLoaded: false,
    load: {
      error: '',
      isRunning: false
    },
    save: {
      error: '',
      isRunning: false
    },
    settings: createEmptySettings(),
    statusMessage: ''
  });

  let loadRequest: Promise<Settings> | undefined;
  let saveRequest: Promise<Settings> | undefined;

  const replaceSettings = (nextSettings: Settings) => {
    const replacement = cloneSettings(normaliseSettings(nextSettings));
    const linearConfigurationChanged =
      state.settings.linear.enabled !== replacement.linear.enabled ||
      state.settings.linear.workspace !== replacement.linear.workspace;

    state.settings = replacement;
    if (linearConfigurationChanged) {
      workspaceDetailsStore.invalidateLinearIssueLinks();
    }
  };

  const currentSettings = () => cloneSettings(state.settings);

  const loadSettings = ({ force = false } = {}) => {
    if (loadRequest) {
      return loadRequest;
    }

    if (state.hasLoaded && !force) {
      return Promise.resolve(currentSettings());
    }

    state.load.error = '';
    state.load.isRunning = true;

    loadRequest = (async () => {
      try {
        const nextSettings = await getSettings();

        replaceSettings(nextSettings);
        state.hasLoaded = true;
        applyThemeAccentColor(state.settings.themeAccentColor);

        return currentSettings();
      } catch (requestError) {
        state.load.error = errorMessage(requestError, 'Settings request failed');
        throw requestError;
      } finally {
        state.load.isRunning = false;
        loadRequest = undefined;
      }
    })();

    return loadRequest;
  };

  const saveSettings = (nextSettings: Settings) => {
    if (saveRequest) {
      return saveRequest;
    }

    const settingsToSave = normaliseSettings(nextSettings);

    state.save.error = '';
    state.save.isRunning = true;
    state.statusMessage = '';

    saveRequest = (async () => {
      try {
        const savedSettings = await updateSettings(cloneSettings(settingsToSave));

        replaceSettings(savedSettings);
        state.hasLoaded = true;
        applyThemeAccentColor(state.settings.themeAccentColor);

        const availableWorkspaces = await syncWorkspaces();
        if (availableWorkspaces) {
          removeUnavailableRecentWorkspaces(availableWorkspaces);
        }

        state.statusMessage = 'Settings saved';

        return currentSettings();
      } catch (requestError) {
        state.save.error = errorMessage(requestError, 'Settings save failed');
        throw requestError;
      } finally {
        state.save.isRunning = false;
        saveRequest = undefined;
      }
    })();

    return saveRequest;
  };

  const reloadSettingsFromDisk = async () => {
    const reloadedSettings = await reloadSettings();
    replaceSettings(reloadedSettings);
    state.hasLoaded = true;
    applyThemeAccentColor(state.settings.themeAccentColor);

    return currentSettings();
  };

  const agents = computed(() => state.settings.agents);
  const hasLoaded = computed(() => state.hasLoaded);
  const isLoading = computed(() => state.load.isRunning);
  const isSaving = computed(() => state.save.isRunning);
  const loadError = computed(() => state.load.error);
  const workspaceDirectories = computed(() => state.settings.workspaceDirectories);
  const saveError = computed(() => state.save.error);
  const settings = computed(() => state.settings);
  const statusMessage = computed(() => state.statusMessage);

  return {
    agents: readonly(agents),
    hasLoaded: readonly(hasLoaded),
    isLoading: readonly(isLoading),
    isSaving: readonly(isSaving),
    loadError: readonly(loadError),
    loadSettings,
    reloadSettingsFromDisk,
    workspaceDirectories: readonly(workspaceDirectories),
    saveError: readonly(saveError),
    saveSettings,
    settings: readonly(settings),
    statusMessage: readonly(statusMessage)
  };
});
