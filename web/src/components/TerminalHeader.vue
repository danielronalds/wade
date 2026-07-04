<script setup lang="ts">
import { ArrowDownToLine, RefreshCw } from '@lucide/vue';

defineProps<{
  label: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  scrollToBottom: [];
  reload: [];
}>();

const scrollToBottom = () => {
  emit('scrollToBottom');
};

const reload = () => {
  emit('reload');
};
</script>

<template>
  <header class="terminal-header" :data-active="String(isActive)">
    <h2>{{ label }}</h2>
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

.terminal-header[data-active="true"] {
  color: var(--text);
}

h2 {
  margin: 0;
  font-size: 12px;
  font-weight: 400;
  line-height: 1;
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
