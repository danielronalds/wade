<script setup lang="ts">
import { ArrowDownToLine, Minimize2, RefreshCw } from '@lucide/vue';

defineProps<{
  label: string;
  side: 'left' | 'right';
}>();

const emit = defineEmits<{
  restore: [];
}>();

const restoreSplitView = () => {
  emit('restore');
};

const stopDisabledAction = () => {};
</script>

<template>
  <section
    class="collapsed-terminal-rail"
    :class="`collapsed-terminal-rail--${side}`"
    :aria-label="`${label} terminal collapsed rail`"
    @click="restoreSplitView"
  >
    <template v-if="side === 'left'">
      <menu class="collapsed-terminal-rail-actions" :aria-label="`${label} terminal actions`">
        <li>
          <button
            class="collapsed-terminal-rail-button"
            type="button"
            aria-label="Restore split view"
            title="Restore split view"
            @click.stop="restoreSplitView"
          >
            <Minimize2 :size="14" :stroke-width="1.7" aria-hidden="true" />
          </button>
        </li>
        <li>
          <button
            class="collapsed-terminal-rail-button collapsed-terminal-rail-button--disabled"
            type="button"
            :aria-label="`Reload ${label} terminal`"
            :title="`Reload ${label} terminal`"
            aria-disabled="true"
            tabindex="-1"
            @click.stop.prevent="stopDisabledAction"
          >
            <RefreshCw :size="14" :stroke-width="1.7" aria-hidden="true" />
          </button>
        </li>
        <li>
          <button
            class="collapsed-terminal-rail-button collapsed-terminal-rail-button--disabled"
            type="button"
            :aria-label="`Scroll ${label} terminal to bottom`"
            :title="`Scroll ${label} terminal to bottom`"
            aria-disabled="true"
            tabindex="-1"
            @click.stop.prevent="stopDisabledAction"
          >
            <ArrowDownToLine :size="14" :stroke-width="1.7" aria-hidden="true" />
          </button>
        </li>
      </menu>
      <section class="collapsed-terminal-rail-label" aria-hidden="true">
        <h2>{{ label }}</h2>
      </section>
    </template>
    <template v-else>
      <section class="collapsed-terminal-rail-label" aria-hidden="true">
        <h2>{{ label }}</h2>
      </section>
      <menu class="collapsed-terminal-rail-actions" :aria-label="`${label} terminal actions`">
        <li>
          <button
            class="collapsed-terminal-rail-button collapsed-terminal-rail-button--disabled"
            type="button"
            :aria-label="`Scroll ${label} terminal to bottom`"
            :title="`Scroll ${label} terminal to bottom`"
            aria-disabled="true"
            tabindex="-1"
            @click.stop.prevent="stopDisabledAction"
          >
            <ArrowDownToLine :size="14" :stroke-width="1.7" aria-hidden="true" />
          </button>
        </li>
        <li>
          <button
            class="collapsed-terminal-rail-button collapsed-terminal-rail-button--disabled"
            type="button"
            :aria-label="`Reload ${label} terminal`"
            :title="`Reload ${label} terminal`"
            aria-disabled="true"
            tabindex="-1"
            @click.stop.prevent="stopDisabledAction"
          >
            <RefreshCw :size="14" :stroke-width="1.7" aria-hidden="true" />
          </button>
        </li>
        <li>
          <button
            class="collapsed-terminal-rail-button"
            type="button"
            aria-label="Restore split view"
            title="Restore split view"
            @click.stop="restoreSplitView"
          >
            <Minimize2 :size="14" :stroke-width="1.7" aria-hidden="true" />
          </button>
        </li>
      </menu>
    </template>
  </section>
</template>

<style scoped>
.collapsed-terminal-rail {
  width: var(--terminal-header-height, 28px);
  min-width: var(--terminal-header-height, 28px);
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 0;
  overflow: hidden;
  background: var(--window);
  color: var(--muted);
  cursor: pointer;
  user-select: none;
}

.collapsed-terminal-rail--left {
  --collapsed-terminal-rail-rotation: -90deg;
  border-right: 1px solid var(--text);
}

.collapsed-terminal-rail--right {
  --collapsed-terminal-rail-rotation: 90deg;
  border-left: 1px solid var(--text);
}

.collapsed-terminal-rail-label {
  width: 100%;
  min-height: 64px;
  display: grid;
  place-items: center;
}

h2 {
  margin: 0;
  color: inherit;
  font-size: 12px;
  font-weight: 400;
  line-height: 1;
  white-space: nowrap;
  transform: rotate(var(--collapsed-terminal-rail-rotation));
}

.collapsed-terminal-rail--left h2 {
  position: relative;
  left: -4px;
}

.collapsed-terminal-rail-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.collapsed-terminal-rail-actions li {
  display: flex;
}

.collapsed-terminal-rail-button {
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  transform: rotate(var(--collapsed-terminal-rail-rotation));
}

.collapsed-terminal-rail-button:hover,
.collapsed-terminal-rail-button:focus-visible {
  color: var(--text);
}

.collapsed-terminal-rail-button--disabled {
  cursor: default;
  opacity: 0.45;
}

.collapsed-terminal-rail-button--disabled:hover,
.collapsed-terminal-rail-button--disabled:focus-visible {
  color: inherit;
}
</style>
