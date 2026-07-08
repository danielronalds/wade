<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import type { Agent } from '../../types/settings';

defineProps<{
  agents: Agent[];
  isLoading: boolean;
}>();

const emit = defineEmits<{
  addAgent: [];
  removeAgent: [index: number];
  setDefaultAgent: [index: number];
  updateAgentName: [index: number, event: Event];
  updateAgentCommand: [index: number, event: Event];
}>();
</script>

<template>
  <section id="agents-section" aria-labelledby="agents-title">
    <header class="settings-section-header">
      <section>
        <h2 id="agents-title">Agents</h2>
        <p>Configure the agents available in the Agent pane dropdown.</p>
      </section>
      <button type="button" class="secondary-action" @click="emit('addAgent')">Add agent</button>
    </header>

    <ul id="agents-list" aria-label="Agents">
      <li v-for="(agent, index) in agents" :key="index" class="agent-row">
        <label :for="`agent-name-${index}`">Agent {{ index + 1 }}</label>
        <input
          :id="`agent-name-${index}`"
          :value="agent.name"
          type="text"
          spellcheck="false"
          autocomplete="off"
          placeholder="Pi"
          :aria-invalid="!isLoading && agent.name.trim() === ''"
          @input="emit('updateAgentName', index, $event)"
        >
        <input
          :id="`agent-command-${index}`"
          :value="agent.command"
          type="text"
          spellcheck="false"
          autocomplete="off"
          placeholder="pi -c"
          :aria-label="`Agent ${index + 1} command`"
          :aria-invalid="!isLoading && agent.command.trim() === ''"
          @input="emit('updateAgentCommand', index, $event)"
        >
        <label class="agent-default-option" :for="`agent-default-${index}`">
          <input
            :id="`agent-default-${index}`"
            :checked="agent.default"
            type="radio"
            name="agent-default"
            @change="emit('setDefaultAgent', index)"
          >
          <span>Default</span>
        </label>
        <button
          type="button"
          class="remove-action"
          :disabled="agents.length <= 1"
          @click="emit('removeAgent', index)"
        >
          Remove
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
#agents-section {
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
.settings-section-header p {
  margin: 0;
}

.settings-section-header h2 {
  font-size: 18px;
}

.settings-section-header p {
  margin-top: 7px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}

button {
  border: 1px solid var(--text);
  border-radius: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  padding: 9px 12px;
}

button:not(:disabled):hover,
button:not(:disabled):focus-visible {
  background: var(--text);
  color: var(--window);
  outline: none;
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

#agents-list {
  display: grid;
  gap: 10px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.agent-row {
  display: grid;
  grid-template-columns: 120px minmax(0, 0.7fr) minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
}

.agent-row label,
.agent-default-option span {
  color: var(--muted);
  font-size: 13px;
}

.agent-row > input {
  min-width: 0;
  border: 1px solid rgb(var(--accent-rgb) / 30%);
  border-radius: 0;
  background: rgb(0 0 0 / 18%);
  color: var(--text);
  font: inherit;
  padding: 9px 10px;
}

.agent-row > input:focus {
  border-color: var(--text);
  outline: none;
}

.agent-row > input[aria-invalid="true"] {
  border-color: var(--disconnected);
}

.agent-default-option {
  width: fit-content;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  border: 1px solid rgb(var(--accent-rgb) / 30%);
  background: rgb(0 0 0 / 12%);
  color: var(--muted);
  cursor: pointer;
}

.agent-default-option:has(input:checked) {
  border-color: var(--text);
  background: rgb(var(--accent-rgb) / 12%);
  color: var(--text);
}

.agent-default-option input {
  width: 13px;
  height: 13px;
  appearance: none;
  display: grid;
  place-items: center;
  margin: 0;
  border: 1px solid currentColor;
  border-radius: 0;
  background: transparent;
}

.agent-default-option input:checked::before {
  width: 7px;
  height: 7px;
  background: currentColor;
  content: '';
}

.agent-default-option:focus-within,
.agent-default-option:hover {
  border-color: var(--text);
  color: var(--text);
}

.remove-action {
  border-color: rgb(var(--accent-rgb) / 35%);
}

@media (max-width: 720px) {
  .settings-section-header {
    align-items: stretch;
    flex-direction: column;
  }

  .agent-row {
    grid-template-columns: 1fr;
  }
}
</style>
