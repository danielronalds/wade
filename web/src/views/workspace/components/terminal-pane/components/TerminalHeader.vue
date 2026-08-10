<script setup lang="ts">
import { computed } from 'vue';
import { ArrowDownToLine, ChevronDown, Maximize2, Minimize2, RefreshCw, X } from '@lucide/vue';
import type { Agent } from '@/types/settings';

const props = withDefaults(
  defineProps<{
    label: string;
    isActive: boolean;
    showCloseIcon?: boolean;
    showZoomIcon?: boolean;
    isZoomed?: boolean;
    agents?: Agent[];
    selectedAgentName?: string;
  }>(),
  {
    showCloseIcon: false,
    showZoomIcon: false,
    isZoomed: false,
    agents: undefined,
    selectedAgentName: undefined
  }
);

const emit = defineEmits<{
  scrollToBottom: [];
  reload: [];
  close: [];
  toggleZoom: [];
  agentChange: [agentName: string];
}>();

const zoomButtonLabel = computed(() => (props.isZoomed ? 'Restore split view' : `Zoom ${props.label} terminal`));

const scrollToBottom = () => {
  emit('scrollToBottom');
};

const reload = () => {
  emit('reload');
};

const close = () => {
  emit('close');
};

const toggleZoom = () => {
  emit('toggleZoom');
};

const updateAgent = (event: Event) => {
  if (!(event.target instanceof HTMLSelectElement)) {
    return;
  }

  emit('agentChange', event.target.value);
};
</script>

<template>
  <header class="terminal-header" :data-active="String(isActive)">
    <section class="terminal-header-title">
      <h2>{{ label }}</h2>
      <span v-if="agents && agents.length > 1 && selectedAgentName !== undefined" class="agent-selector-shell">
        <span class="agent-selector-sizer" aria-hidden="true">{{ selectedAgentName }}</span>
        <select
          class="agent-selector"
          :value="selectedAgentName"
          aria-label="Select agent"
          @change="updateAgent"
          @click.stop
        >
          <option v-for="agent in agents" :key="agent.name" :value="agent.name">
            {{ agent.name }}
          </option>
        </select>
        <ChevronDown class="agent-selector-chevron" :size="12" :stroke-width="1.7" aria-hidden="true" />
      </span>
    </section>
    <menu class="terminal-header-actions" :aria-label="`${label} terminal actions`">
      <li>
        <button
          class="terminal-header-icon-button"
          type="button"
          :aria-label="`Scroll ${label} terminal to bottom`"
          :title="`Scroll ${label} terminal to bottom`"
          @click.stop="scrollToBottom"
        >
          <ArrowDownToLine :size="14" :stroke-width="1.7" aria-hidden="true" />
        </button>
      </li>
      <li>
        <button
          class="terminal-header-icon-button"
          type="button"
          :aria-label="`Reload ${label} terminal`"
          :title="`Reload ${label} terminal`"
          @click.stop="reload"
        >
          <RefreshCw :size="14" :stroke-width="1.7" aria-hidden="true" />
        </button>
      </li>
      <li v-if="showCloseIcon">
        <button
          class="terminal-header-icon-button"
          type="button"
          :aria-label="`Close ${label} terminal`"
          :title="`Close ${label} terminal`"
          @click.stop="close"
        >
          <X :size="14" :stroke-width="1.7" aria-hidden="true" />
        </button>
      </li>
      <li v-if="showZoomIcon">
        <button
          class="terminal-header-icon-button"
          type="button"
          :aria-label="zoomButtonLabel"
          :title="zoomButtonLabel"
          @click.stop="toggleZoom"
        >
          <Minimize2 v-if="isZoomed" :size="14" :stroke-width="1.7" aria-hidden="true" />
          <Maximize2 v-else :size="14" :stroke-width="1.7" aria-hidden="true" />
        </button>
      </li>
    </menu>
  </header>
</template>

<style scoped>
.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 8px 0 12px;
  border-bottom: 1px solid rgb(var(--accent-rgb) / 45%);
  color: var(--muted);
  user-select: none;
}

.terminal-header[data-active='true'] {
  color: var(--text);
}

.terminal-header-title {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 7px;
}

h2 {
  margin: 0;
  font-size: 12px;
  font-weight: 400;
  line-height: 1;
}

.agent-selector-shell {
  position: relative;
  display: inline-grid;
  align-items: center;
  color: var(--muted);
}

.agent-selector-sizer,
.agent-selector {
  grid-area: 1 / 1;
  font: inherit;
  font-size: 12px;
  line-height: 1;
  white-space: pre;
}

.agent-selector-sizer {
  padding-right: 13px;
  visibility: hidden;
}

.agent-selector {
  width: 100%;
  min-width: 0;
  appearance: none;
  border: 0;
  background: transparent;
  color: inherit;
  outline: none;
  padding: 0 13px 0 0;
}

.agent-selector-chevron {
  position: absolute;
  right: 0;
  pointer-events: none;
}

.agent-selector-shell:focus-within,
.agent-selector-shell:hover {
  color: var(--text);
}

.terminal-header-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.terminal-header-actions li {
  display: flex;
}

.terminal-header-icon-button {
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.terminal-header-icon-button:hover,
.terminal-header-icon-button:focus-visible {
  color: var(--text);
}
</style>
