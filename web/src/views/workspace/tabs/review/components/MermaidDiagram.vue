<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useMermaidRenderer } from '../composables/useMermaidRenderer';

const props = defineProps<{
  source: string;
}>();
const { renderMermaidDiagram } = useMermaidRenderer();
const renderedSvg = ref('');
const errorMessage = ref('');
const isRendering = ref(false);
let latestRenderRequest = 0;

const renderDiagram = async () => {
  const renderRequest = ++latestRenderRequest;
  renderedSvg.value = '';
  errorMessage.value = '';
  isRendering.value = true;

  try {
    const svg = await renderMermaidDiagram(props.source);
    if (renderRequest !== latestRenderRequest) {
      return;
    }

    renderedSvg.value = svg;
  } catch (error) {
    if (renderRequest !== latestRenderRequest) {
      return;
    }

    errorMessage.value = error instanceof Error ? error.message : 'Mermaid returned an unknown rendering error.';
  } finally {
    if (renderRequest === latestRenderRequest) {
      isRendering.value = false;
    }
  }
};

const handleThemeAccentChange = () => {
  void renderDiagram();
};

watch(
  () => props.source,
  () => void renderDiagram(),
  { immediate: true }
);

onMounted(() => {
  window.addEventListener('theme-accent-color-changed', handleThemeAccentChange);
});

onBeforeUnmount(() => {
  latestRenderRequest += 1;
  window.removeEventListener('theme-accent-color-changed', handleThemeAccentChange);
});
</script>

<template>
  <figure class="mermaid-diagram" :data-state="isRendering ? 'loading' : renderedSvg ? 'rendered' : 'error'">
    <div v-if="isRendering" class="mermaid-diagram-status" role="status">Rendering Mermaid diagram...</div>
    <div v-else-if="renderedSvg" class="mermaid-diagram-svg" v-html="renderedSvg"></div>
    <section v-else class="mermaid-diagram-error" role="alert">
      <p>Mermaid diagram could not be rendered.</p>
      <p v-if="errorMessage" class="mermaid-diagram-error-message">{{ errorMessage }}</p>
      <pre><code>{{ source }}</code></pre>
    </section>
  </figure>
</template>

<style scoped>
.mermaid-diagram {
  min-width: 0;
  max-width: 100%;
  margin: 0.8em 0;
  padding: 12px;
  border: 1px solid rgb(var(--accent-rgb) / 32%);
  background: rgb(0 0 0 / 12%);
  overflow: auto;
  scrollbar-width: thin;
}

.mermaid-diagram-svg {
  min-width: 0;
  color: var(--text);
}

.mermaid-diagram-svg :deep(svg) {
  display: block;
  width: 100%;
  max-width: 100%;
  height: auto;
}

.mermaid-diagram-status,
.mermaid-diagram-error {
  color: var(--muted);
  font-size: 12px;
  line-height: 1.5;
}

.mermaid-diagram-error {
  display: grid;
  gap: 8px;
}

.mermaid-diagram-error p {
  margin: 0;
}

.mermaid-diagram-error-message {
  color: #ff6e6e;
  overflow-wrap: anywhere;
}

.mermaid-diagram-error pre {
  max-height: 320px;
  margin: 0;
  padding: 10px;
  border: 1px solid rgb(var(--accent-rgb) / 24%);
  background: rgb(0 0 0 / 18%);
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.mermaid-diagram-error code {
  color: var(--text);
  font: inherit;
  font-size: 12px;
}
</style>
