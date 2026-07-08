<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import AgentSettingsEditor from '../components/settings/AgentSettingsEditor.vue';
import ThemeAccentPicker from '../components/settings/ThemeAccentPicker.vue';
import { useSettingsForm } from '../composables/useSettingsForm';

const {
  form,
  isLoading,
  isSaving,
  error,
  statusMessage,
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
  updateThemeAccentColor,
  updateWorktreeCopyExclude,
  addWorktreeCopyExclude,
  removeWorktreeCopyExclude,
  submit
} = useSettingsForm();

const selectedThemeAccentColor = computed({
  get: () => form.themeAccentColor,
  set: updateThemeAccentColor
});

onMounted(() => {
  document.title = 'WADE - Settings';
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
      <form id="settings-form" @submit.prevent="submit">
        <ThemeAccentPicker v-model:theme-accent-color="selectedThemeAccentColor" />

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
            <li v-for="(directory, index) in form.projectDirectories" :key="index" class="project-directory-row">
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

          <p v-if="!isLoading && form.projectDirectories.length === 0" class="settings-message">
            No project directories configured.
          </p>
        </section>

        <section id="shell-section" aria-labelledby="shell-title">
          <header class="settings-section-header">
            <section>
              <h2 id="shell-title">Shell</h2>
              <p>Leave blank to use the server's <code>$SHELL</code>. Existing terminals keep their current shell until reloaded.</p>
            </section>
          </header>

          <label class="shell-row" for="shell">
            <span>Shell</span>
            <input
              id="shell"
              :value="form.shell"
              type="text"
              spellcheck="false"
              autocomplete="off"
              placeholder="$SHELL"
              :aria-invalid="hasInvalidShell"
              @input="updateShell"
            >
          </label>
        </section>

        <AgentSettingsEditor
          :agents="form.agents"
          :is-loading="isLoading"
          @add-agent="addAgent"
          @remove-agent="removeAgent"
          @set-default-agent="setDefaultAgent"
          @update-agent-name="updateAgentName"
          @update-agent-command="updateAgentCommand"
        />

        <section id="worktree-copy-section" aria-labelledby="worktree-copy-title">
          <header class="settings-section-header">
            <section>
              <h2 id="worktree-copy-title">Worktree ignored file copy</h2>
              <p>Copy gitignored files, like .env files, into newly created worktrees.</p>
            </section>
            <button type="button" class="secondary-action" @click="addWorktreeCopyExclude">Add exclude</button>
          </header>

          <label class="checkbox-setting-row" for="copy-ignored-files-on-worktree-creation">
            <input
              id="copy-ignored-files-on-worktree-creation"
              :checked="form.copyIgnoredFilesOnWorktreeCreation"
              type="checkbox"
              @change="updateCopyIgnoredFilesOnWorktreeCreation"
            >
            <span>Copy ignored files when creating a worktree</span>
          </label>

          <ul id="worktree-copy-excludes-list" aria-label="Worktree ignored file copy excludes">
            <li v-for="(exclude, index) in form.worktreeCopyExcludes" :key="index" class="worktree-copy-exclude-row">
              <label :for="`worktree-copy-exclude-${index}`">Exclude {{ index + 1 }}</label>
              <input
                :id="`worktree-copy-exclude-${index}`"
                :value="exclude"
                type="text"
                spellcheck="false"
                autocomplete="off"
                placeholder="**/node_modules"
                @input="updateWorktreeCopyExclude(index, $event)"
              >
              <button type="button" class="remove-action" @click="removeWorktreeCopyExclude(index)">Remove</button>
            </li>
          </ul>
        </section>

        <footer id="settings-actions">
          <p v-if="!isLoading && hasInvalidProjectDirectories" class="settings-error">
            Project directories must use ~ or an absolute path.
          </p>
          <p v-else-if="!isLoading && hasInvalidShell" class="settings-error">
            Shell must be a program path or command without arguments.
          </p>
          <p v-else-if="!isLoading && hasInvalidAgents" class="settings-error">
            At least one agent is required. Agent names and commands cannot be empty, names must be unique, and exactly one agent must be default.
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
  border: 1px solid rgb(var(--accent-rgb) / 45%);
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

#settings-form {
  width: min(860px, 100%);
  display: grid;
  gap: 52px;
}

#project-directories-section,
#shell-section,
#worktree-copy-section {
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
  border-bottom: 1px solid rgb(var(--accent-rgb) / 18%);
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

#project-directories-list,
#worktree-copy-excludes-list {
  display: grid;
  gap: 10px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.project-directory-row,
.shell-row,
.worktree-copy-exclude-row,
.checkbox-setting-row {
  display: grid;
  align-items: center;
  gap: 10px;
}

.project-directory-row,
.worktree-copy-exclude-row {
  grid-template-columns: 120px minmax(0, 1fr) auto;
}

.shell-row {
  grid-template-columns: 120px minmax(0, 1fr);
}

.checkbox-setting-row {
  grid-template-columns: auto minmax(0, 1fr);
  justify-content: start;
}

.project-directory-row label,
.shell-row span,
.worktree-copy-exclude-row label,
.checkbox-setting-row span {
  color: var(--muted);
  font-size: 13px;
}

.project-directory-row input,
.shell-row input,
.worktree-copy-exclude-row input {
  min-width: 0;
  border: 1px solid rgb(var(--accent-rgb) / 30%);
  border-radius: 0;
  background: rgb(0 0 0 / 18%);
  color: var(--text);
  font: inherit;
  padding: 9px 10px;
}

.project-directory-row input:focus,
.shell-row input:focus,
.worktree-copy-exclude-row input:focus {
  border-color: var(--text);
  outline: none;
}

.project-directory-row input[aria-invalid="true"],
.shell-row input[aria-invalid="true"],
.worktree-copy-exclude-row input[aria-invalid="true"] {
  border-color: var(--disconnected);
}

.remove-action {
  border-color: rgb(var(--accent-rgb) / 35%);
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

  .project-directory-row,
  .shell-row,
  .worktree-copy-exclude-row {
    grid-template-columns: 1fr;
  }
}
</style>
