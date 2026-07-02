<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { useProjects } from '../composables/useProjects';
import { useRecentProjects } from '../composables/useRecentProjects';

type SettingsResponse = {
  projectDirectories: string[];
};

const { syncProjects } = useProjects();
const { removeUnavailableRecentProjects } = useRecentProjects();

const projectDirectories = ref<string[]>([]);
const savedProjectDirectories = ref<string[]>([]);
const isLoading = ref(false);
const isSaving = ref(false);
const error = ref('');
const statusMessage = ref('');

const normaliseProjectDirectories = (directories: readonly string[]) => directories.map((directory) => directory.trim());

const isSettingsResponse = (value: unknown): value is SettingsResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const response = value as Partial<SettingsResponse>;

  return Array.isArray(response.projectDirectories)
    && response.projectDirectories.every((directory) => typeof directory === 'string');
};

const isValidProjectDirectory = (directory: string) => {
  const trimmedDirectory = directory.trim();

  return trimmedDirectory === '~'
    || trimmedDirectory.startsWith('~/')
    || trimmedDirectory.startsWith('/');
};

const normalisedProjectDirectories = computed(() => normaliseProjectDirectories(projectDirectories.value));
const hasInvalidProjectDirectories = computed(() => projectDirectories.value.some(
  (directory) => !isValidProjectDirectory(directory)
));
const hasChanges = computed(() => JSON.stringify(normalisedProjectDirectories.value)
  !== JSON.stringify(savedProjectDirectories.value));
const canSave = computed(() => !isLoading.value
  && !isSaving.value
  && hasChanges.value
  && !hasInvalidProjectDirectories.value);

const loadSettings = async () => {
  isLoading.value = true;
  error.value = '';
  statusMessage.value = '';

  try {
    const response = await fetch('/api/config');
    if (!response.ok) {
      throw new Error(`Settings request failed with ${response.status}`);
    }

    const settings: unknown = await response.json();
    if (!isSettingsResponse(settings)) {
      throw new Error('Settings response was invalid');
    }

    projectDirectories.value = [...settings.projectDirectories];
    savedProjectDirectories.value = normaliseProjectDirectories(settings.projectDirectories);
  } catch (requestError) {
    error.value = requestError instanceof Error ? requestError.message : 'Settings request failed';
  } finally {
    isLoading.value = false;
  }
};

const updateProjectDirectory = (index: number, event: Event) => {
  if (!(event.target instanceof HTMLInputElement)) {
    return;
  }

  const nextDirectory = event.target.value;
  projectDirectories.value = projectDirectories.value.map((directory, directoryIndex) => (
    directoryIndex === index ? nextDirectory : directory
  ));
};

const addProjectDirectory = async () => {
  projectDirectories.value = [...projectDirectories.value, ''];
  statusMessage.value = '';
  error.value = '';

  await nextTick();
  document.getElementById(`project-directory-${projectDirectories.value.length - 1}`)?.focus();
};

const removeProjectDirectory = (index: number) => {
  projectDirectories.value = projectDirectories.value.filter((_, directoryIndex) => directoryIndex !== index);
  statusMessage.value = '';
  error.value = '';
};

const reloadConfig = async () => {
  const response = await fetch('/api/config/reload', { method: 'POST' });
  if (!response.ok) {
    throw new Error(`Config reload failed with ${response.status}`);
  }
};

const refreshProjects = async () => {
  const availableProjects = await syncProjects();
  if (availableProjects) {
    removeUnavailableRecentProjects(availableProjects);
  }
};

const saveSettings = async () => {
  if (!canSave.value) {
    return;
  }

  isSaving.value = true;
  error.value = '';
  statusMessage.value = '';

  try {
    const nextProjectDirectories = normalisedProjectDirectories.value;
    const response = await fetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectDirectories: nextProjectDirectories })
    });

    if (!response.ok) {
      const message = await response.text();
      throw new Error(message.trim() || `Settings save failed with ${response.status}`);
    }

    await reloadConfig();
    await refreshProjects();

    projectDirectories.value = [...nextProjectDirectories];
    savedProjectDirectories.value = [...nextProjectDirectories];
    statusMessage.value = 'Settings saved';
  } catch (saveError) {
    error.value = saveError instanceof Error ? saveError.message : 'Settings save failed';
  } finally {
    isSaving.value = false;
  }
};

onMounted(() => {
  document.title = 'WADE - Settings';
  void loadSettings();
});
</script>

<template>
  <section id="settings-view" aria-labelledby="settings-title">
    <header id="settings-topbar">
      <h1 id="settings-title">
        <RouterLink id="brand" :to="{ name: 'home' }">WADE</RouterLink>
        <span id="settings-name">Settings</span>
      </h1>
      <section id="settings-topbar-actions" aria-label="Settings status">
        <span id="connection-status" role="status" aria-live="polite" data-connected="true">
          <span aria-hidden="true"></span>
          <span>Connected</span>
        </span>
      </section>
    </header>

    <section id="settings-content" aria-label="Settings form">
      <form id="settings-form" @submit.prevent="saveSettings">
        <section id="project-directories-section" aria-labelledby="project-directories-title">
          <header class="settings-section-header">
            <section>
              <h2 id="project-directories-title">Project directories</h2>
              <p>Use <code>~</code> or absolute paths. Missing directories are skipped.</p>
            </section>
            <button type="button" class="secondary-action" @click="addProjectDirectory">Add directory</button>
          </header>

          <p v-if="isLoading" class="settings-message">Loading settings</p>

          <ul v-else id="project-directories-list" aria-label="Project directories">
            <li v-for="(directory, index) in projectDirectories" :key="index" class="project-directory-row">
              <label :for="`project-directory-${index}`">Directory {{ index + 1 }}</label>
              <input
                :id="`project-directory-${index}`"
                :value="directory"
                type="text"
                spellcheck="false"
                autocomplete="off"
                placeholder="~/Personal"
                :aria-invalid="!isValidProjectDirectory(directory)"
                @input="updateProjectDirectory(index, $event)"
              >
              <button type="button" class="remove-action" @click="removeProjectDirectory(index)">Remove</button>
            </li>
          </ul>

          <p v-if="!isLoading && projectDirectories.length === 0" class="settings-message">
            No project directories configured.
          </p>
        </section>

        <footer id="settings-actions">
          <p v-if="hasInvalidProjectDirectories" class="settings-error">
            Project directories must use ~ or an absolute path.
          </p>
          <p v-else-if="error" class="settings-error">{{ error }}</p>
          <p v-else-if="statusMessage" class="settings-status">{{ statusMessage }}</p>
          <button id="settings-save" type="submit" :disabled="!canSave">
            {{ isSaving ? 'Saving' : 'Save settings' }}
          </button>
        </footer>
      </form>
    </section>
  </section>
</template>

<style scoped>
#settings-view {
  width: 100vw;
  height: 100vh;
  display: grid;
  grid-template-rows: 42px minmax(0, 1fr);
  overflow: hidden;
  background: var(--window);
  color: var(--text);
}

#settings-topbar {
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 0 16px;
  border-bottom: 1px solid var(--text);
  background: var(--window);
  color: var(--text);
  user-select: none;
}

#settings-title {
  min-width: 0;
  display: flex;
  margin: 0;
  align-items: center;
  gap: 14px;
  font-weight: 400;
}

#brand {
  margin: 0;
  color: var(--text);
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
  text-decoration: none;
}

#brand:hover,
#brand:focus-visible {
  text-decoration: underline;
}

#settings-name {
  flex: 0 1 auto;
  margin: 0;
  overflow: hidden;
  color: var(--muted);
  font-size: 14px;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

#settings-topbar-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

#connection-status {
  height: 26px;
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 9px;
  border: 1px solid rgb(248 248 242 / 45%);
  border-radius: 999px;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 12px;
  letter-spacing: 0.01em;
  line-height: 1;
}

#connection-status span:first-child {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--disconnected);
}

#connection-status[data-connected="true"] span:first-child {
  background: var(--connected);
}

#settings-content {
  min-height: 0;
  overflow: auto;
  display: grid;
  align-content: start;
  justify-items: center;
  gap: 28px;
  padding: clamp(24px, 7vw, 72px);
}

#settings-form,
#project-directories-section {
  width: min(860px, 100%);
  display: grid;
  gap: 18px;
}

.settings-section-header {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: 18px;
  padding-bottom: 13px;
  border-bottom: 1px solid rgb(248 248 242 / 18%);
}

.settings-section-header h2,
.settings-section-header p,
.settings-message,
.settings-error,
.settings-status {
  margin: 0;
}

.settings-section-header h2 {
  font-size: 18px;
}

.settings-section-header p,
.settings-message {
  margin-top: 7px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}

#settings-content button {
  border: 1px solid var(--text);
  border-radius: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  padding: 9px 12px;
}

#settings-content button:not(:disabled):hover,
#settings-content button:not(:disabled):focus-visible {
  background: var(--text);
  color: var(--window);
  outline: none;
}

#settings-content button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

#project-directories-list {
  display: grid;
  gap: 10px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.project-directory-row {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.project-directory-row label {
  color: var(--muted);
  font-size: 13px;
}

.project-directory-row input {
  min-width: 0;
  border: 1px solid rgb(248 248 242 / 30%);
  border-radius: 0;
  background: rgb(0 0 0 / 18%);
  color: var(--text);
  font: inherit;
  padding: 9px 10px;
}

.project-directory-row input:focus {
  border-color: var(--text);
  outline: none;
}

.project-directory-row input[aria-invalid="true"] {
  border-color: var(--disconnected);
}

.remove-action {
  border-color: rgb(248 248 242 / 35%);
}

#settings-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-top: 6px;
}

.settings-error {
  color: var(--disconnected);
  font-size: 13px;
}

.settings-status {
  color: var(--connected);
  font-size: 13px;
}

#settings-save {
  min-width: 148px;
  margin-left: auto;
}

@media (max-width: 720px) {
  .settings-section-header,
  #settings-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .project-directory-row {
    grid-template-columns: 1fr;
  }
}
</style>
