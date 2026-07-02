<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import { useSettingsForm } from '../composables/useSettingsForm';

const {
  form,
  isLoading,
  isSaving,
  error,
  statusMessage,
  hasInvalidProjectDirectories,
  hasInvalidAgentPaneCommand,
  canSave,
  isValidProjectDirectory,
  updateProjectDirectory,
  addProjectDirectory,
  removeProjectDirectory,
  updateAgentPaneCommand,
  submit
} = useSettingsForm();

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

        <section id="agent-pane-command-section" aria-labelledby="agent-pane-command-title">
          <header class="settings-section-header">
            <section>
              <h2 id="agent-pane-command-title">Agent Pane command</h2>
              <p>Runs through your shell when the Agent pane starts. Reload Agent to apply changes.</p>
            </section>
          </header>

          <label class="single-setting-row" for="agent-pane-command">
            <span>Command</span>
            <input
              id="agent-pane-command"
              :value="form.agentPaneCommand"
              type="text"
              spellcheck="false"
              autocomplete="off"
              placeholder="pi -c"
              :aria-invalid="!isLoading && hasInvalidAgentPaneCommand"
              @input="updateAgentPaneCommand"
            >
          </label>
        </section>

        <footer id="settings-actions">
          <p v-if="!isLoading && hasInvalidProjectDirectories" class="settings-error">
            Project directories must use ~ or an absolute path.
          </p>
          <p v-else-if="!isLoading && hasInvalidAgentPaneCommand" class="settings-error">
            Agent Pane command cannot be empty.
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
#project-directories-section,
#agent-pane-command-section {
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

.project-directory-row,
.single-setting-row {
  display: grid;
  align-items: center;
  gap: 10px;
}

.project-directory-row {
  grid-template-columns: 120px minmax(0, 1fr) auto;
}

.single-setting-row {
  grid-template-columns: 120px minmax(0, 1fr);
}

.project-directory-row label,
.single-setting-row span {
  color: var(--muted);
  font-size: 13px;
}

.project-directory-row input,
.single-setting-row input {
  min-width: 0;
  border: 1px solid rgb(248 248 242 / 30%);
  border-radius: 0;
  background: rgb(0 0 0 / 18%);
  color: var(--text);
  font: inherit;
  padding: 9px 10px;
}

.project-directory-row input:focus,
.single-setting-row input:focus {
  border-color: var(--text);
  outline: none;
}

.project-directory-row input[aria-invalid="true"],
.single-setting-row input[aria-invalid="true"] {
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

  .project-directory-row,
  .single-setting-row {
    grid-template-columns: 1fr;
  }
}
</style>
