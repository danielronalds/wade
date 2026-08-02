import { defineStore } from 'pinia';
import { computed, reactive, readonly } from 'vue';
import {
  getSettings,
  updateSettings
} from '@/api/generated/wade';
import { useProjects } from '@/features/projects/composables/useProjects';
import { useRecentProjects } from '@/features/projects/composables/useRecentProjects';
import {
  cloneSettings,
  createEmptySettings,
  normaliseSettings,
  type Settings
} from '@/types/settings';
import { applyThemeAccentColor } from '@/utils/theme';

const errorMessage = (error: unknown, fallback: string) => error instanceof Error
  ? error.message
  : fallback;

export const useSettingsStore = defineStore('settings', () => {
  const { syncProjects } = useProjects();
  const { removeUnavailableRecentProjects } = useRecentProjects();

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
    state.settings = cloneSettings(normaliseSettings(nextSettings));
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
        const nextSettings = await getSettings() as Settings;

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
        const savedSettings = await updateSettings(cloneSettings(settingsToSave)) as Settings;

        const availableProjects = await syncProjects();
        if (availableProjects) {
          removeUnavailableRecentProjects(availableProjects);
        }

        replaceSettings(savedSettings);
        state.hasLoaded = true;
        applyThemeAccentColor(state.settings.themeAccentColor);
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
    workspaceDirectories: readonly(workspaceDirectories),
    saveError: readonly(saveError),
    saveSettings,
    settings: readonly(settings),
    statusMessage: readonly(statusMessage)
  };
});
