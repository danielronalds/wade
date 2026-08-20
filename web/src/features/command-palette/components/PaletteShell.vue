<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import type { PaletteNotice, PaletteResult } from '@/features/command-palette/types';

const props = defineProps<{
  title: string;
  summary: string;
  query: string;
  searchPlaceholder: string;
  resultsAriaLabel: string;
  statusMessage: string;
  results: PaletteResult[];
  notice?: PaletteNotice;
}>();

const emit = defineEmits<{
  close: [restoreFocus?: boolean];
  'update:query': [query: string];
}>();

const resultId = (index: number) => `command-palette-result-${index}`;

const selectedIndex = ref(0);
const searchInput = ref<HTMLInputElement | null>(null);
const selectedResult = computed(() => props.results[selectedIndex.value]);
const activeDescendant = computed(() => (selectedResult.value ? resultId(selectedIndex.value) : undefined));

const updateQuery = (event: Event) => {
  if (!(event.target instanceof HTMLInputElement)) {
    return;
  }

  emit('update:query', event.target.value);
};

const closePalette = (restoreFocus = true) => {
  emit('close', restoreFocus);
};

const scrollSelectedResultIntoView = () => {
  void nextTick(() => {
    document.getElementById(resultId(selectedIndex.value))?.scrollIntoView({ block: 'nearest' });
  });
};

const moveSelection = (offset: number) => {
  const resultCount = props.results.length;
  if (resultCount === 0) {
    return;
  }

  selectedIndex.value = (selectedIndex.value + offset + resultCount) % resultCount;
  scrollSelectedResultIntoView();
};

const runResult = (result: PaletteResult | undefined) => {
  if (!result || result.isDisabled) {
    return;
  }

  result.run();
};

const runSelectedResult = () => {
  runResult(selectedResult.value);
};

const handleKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'Escape':
      event.preventDefault();
      event.stopImmediatePropagation();
      closePalette();
      break;
    case 'ArrowDown':
      event.preventDefault();
      event.stopImmediatePropagation();
      moveSelection(1);
      break;
    case 'ArrowUp':
      event.preventDefault();
      event.stopImmediatePropagation();
      moveSelection(-1);
      break;
    case 'Enter':
      event.preventDefault();
      event.stopImmediatePropagation();
      runSelectedResult();
      break;
  }
};

watch(
  () => props.query,
  () => {
    selectedIndex.value = 0;
  }
);

watch(
  () => props.results,
  () => {
    if (selectedIndex.value < props.results.length) {
      return;
    }

    selectedIndex.value = Math.max(0, props.results.length - 1);
  }
);

onMounted(() => {
  window.addEventListener('keydown', handleKeydown, true);
  void nextTick(() => {
    searchInput.value?.focus();
    searchInput.value?.select();
  });
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown, true);
});
</script>

<template>
  <Teleport to="body">
    <section id="command-palette-backdrop" aria-label="Command palette backdrop" @mousedown.self="closePalette()">
      <section id="command-palette" role="dialog" aria-modal="true" aria-labelledby="command-palette-title">
        <header id="command-palette-header">
          <h2 id="command-palette-title">{{ title }}</h2>
          <p>{{ summary }}</p>
        </header>
        <form id="command-palette-search-form" role="search" @submit.prevent="runSelectedResult">
          <label for="command-palette-search">{{ searchPlaceholder }}</label>
          <input
            id="command-palette-search"
            ref="searchInput"
            :value="query"
            type="search"
            autocomplete="off"
            spellcheck="false"
            role="combobox"
            aria-autocomplete="list"
            aria-controls="command-palette-results"
            :aria-activedescendant="activeDescendant"
            aria-expanded="true"
            :placeholder="searchPlaceholder"
            @input="updateQuery"
          />
        </form>
        <section v-if="notice" id="command-palette-notice" role="status" :data-tone="notice.tone">
          <h3>{{ notice.title }}</h3>
          <ul>
            <li v-for="(message, index) in notice.messages" :key="`${index}:${message}`">{{ message }}</li>
          </ul>
        </section>
        <nav id="command-palette-results-region" :aria-label="resultsAriaLabel">
          <ul v-if="results.length > 0" id="command-palette-results">
            <li v-for="(result, index) in results" :key="result.id">
              <button
                :id="resultId(index)"
                class="command-palette-result"
                type="button"
                :disabled="result.isDisabled"
                :data-selected="String(index === selectedIndex)"
                :title="result.title"
                @mouseenter="selectedIndex = index"
                @click="runResult(result)"
              >
                <span class="command-palette-result-presentation">
                  <span>{{ result.label }}</span>
                  <span v-if="result.secondaryLabel" class="command-palette-result-secondary">
                    <component
                      v-if="result.icon"
                      :is="result.icon"
                      :size="14"
                      :stroke-width="1.75"
                      aria-hidden="true"
                    />
                    <span>{{ result.secondaryLabel }}</span>
                  </span>
                </span>
                <span v-if="result.actionLabel" class="command-palette-result-action">{{ result.actionLabel }}</span>
              </button>
            </li>
          </ul>
          <p v-else id="command-palette-empty">{{ statusMessage }}</p>
        </nav>
      </section>
    </section>
  </Teleport>
</template>

<style scoped>
#command-palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: grid;
  align-items: start;
  justify-items: center;
  padding: min(18vh, 140px) 16px 16px;
  background: rgb(0 0 0 / 28%);
  backdrop-filter: blur(2px);
}

#command-palette {
  width: min(680px, 100%);
  overflow: hidden;
  border: 1px solid var(--text);
  border-radius: 0;
  background: rgb(23 24 28 / 94%);
  box-shadow: 0 24px 80px rgb(0 0 0 / 38%);
  color: var(--text);
}

#command-palette-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 11px 14px;
  border-bottom: 1px solid rgb(var(--accent-rgb) / 14%);
  color: var(--muted);
  font-size: 12px;
  line-height: 1;
}

#command-palette-header h2,
#command-palette-header p {
  margin: 0;
}

#command-palette-header h2 {
  color: var(--text);
  font-size: 13px;
  font-weight: 700;
}

#command-palette-search-form {
  border-bottom: 1px solid rgb(var(--accent-rgb) / 14%);
}

#command-palette-search-form label {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

#command-palette-search {
  width: 100%;
  height: 58px;
  padding: 0 18px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 18px;
}

#command-palette-search::placeholder {
  color: var(--muted);
}

#command-palette-search::-webkit-search-cancel-button {
  display: none;
}

#command-palette-notice {
  display: grid;
  gap: 8px;
  margin: 8px;
  padding: 11px 12px;
  border: 1px solid;
  background: rgb(var(--accent-rgb) / 6%);
  font-size: 12px;
  line-height: 1.4;
}

#command-palette-notice[data-tone='warning'] {
  border-color: #d29922;
  background: rgb(210 153 34 / 14%);
}

#command-palette-notice[data-tone='error'] {
  border-color: #ff6e6e;
  background: rgb(255 110 110 / 14%);
}

#command-palette-notice h3,
#command-palette-notice ul {
  margin: 0;
}

#command-palette-notice h3 {
  font-size: 12px;
  font-weight: 700;
}

#command-palette-notice[data-tone='warning'] h3 {
  color: #d29922;
}

#command-palette-notice[data-tone='error'] h3 {
  color: #ff6e6e;
}

#command-palette-notice ul {
  max-height: 120px;
  overflow: auto;
  padding-left: 18px;
  color: var(--text);
  scrollbar-width: thin;
}

#command-palette-results-region {
  max-height: min(52vh, 420px);
  overflow: auto;
  padding: 6px;
  scrollbar-width: thin;
}

#command-palette-results {
  display: grid;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.command-palette-result {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 11px 12px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.command-palette-result[data-selected='true'] {
  background: rgb(var(--accent-rgb) / 10%);
}

.command-palette-result:hover,
.command-palette-result:focus-visible {
  background: rgb(var(--accent-rgb) / 12%);
}

.command-palette-result:disabled {
  color: var(--muted);
  cursor: not-allowed;
  opacity: 0.55;
}

.command-palette-result-presentation {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-palette-result-secondary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 12px;
  color: var(--muted);
  font-size: 12px;
  vertical-align: middle;
}

.command-palette-result-secondary svg {
  flex: 0 0 auto;
}

.command-palette-result-action {
  flex: 0 0 auto;
  color: var(--muted);
  font-size: 12px;
}

#command-palette-empty {
  margin: 0;
  padding: 28px 12px;
  color: var(--muted);
  text-align: center;
}
</style>
