<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';
import type { ReviewFileContents } from '../../types/review';

type EditorLayoutDimension = {
  width: number;
  height: number;
};

type MonacoEditor = {
  dispose: () => void;
  layout: (dimension?: EditorLayoutDimension) => void;
  setModel: (model: { original: unknown; modified: unknown } | null) => void;
  updateOptions: (options: Record<string, unknown>) => void;
};

type MonacoModel = {
  dispose: () => void;
};

type MonacoApi = {
  editor: {
    createDiffEditor: (element: HTMLElement, options: Record<string, unknown>) => MonacoEditor;
    createModel: (value: string, language: string) => MonacoModel;
    defineTheme: (name: string, theme: Record<string, unknown>) => void;
    setTheme: (name: string) => void;
  };
};

type MonacoRequire = {
  config: (options: { paths: Record<string, string> }) => void;
  (modules: string[], onLoad: () => void, onError?: (error: unknown) => void): void;
};

declare global {
  interface Window {
    MonacoEnvironment?: {
      getWorkerUrl: (moduleId: string, label: string) => string;
    };
    monaco?: MonacoApi;
    require?: MonacoRequire;
  }
}

const props = defineProps<{
  contents: ReviewFileContents | null;
  filePath: string;
  isDiff: boolean;
  isLoading: boolean;
}>();

const editorElement = ref<HTMLElement | null>(null);
const monaco = shallowRef<MonacoApi | null>(null);
const statusMessage = ref('Loading editor');
const hasEditorError = ref(false);
const isEditorReady = ref(false);

let monacoLoadPromise: Promise<MonacoApi> | null = null;
let editor: MonacoEditor | null = null;
let originalModel: MonacoModel | null = null;
let modifiedModel: MonacoModel | null = null;
let resizeObserver: ResizeObserver | null = null;

const placeholderText = computed(() => {
  if (hasEditorError.value) {
    return statusMessage.value;
  }

  if (props.isLoading) {
    return 'Loading file contents';
  }

  if (!isEditorReady.value) {
    return 'Loading editor';
  }

  if (!props.contents) {
    return 'Select a file to start reviewing';
  }

  return '';
});

const loadScript = (src: string) => new Promise<void>((resolve, reject) => {
  const existingScript = document.querySelector<HTMLScriptElement>(`script[src="${src}"]`);
  if (existingScript?.dataset.loaded === 'true') {
    resolve();
    return;
  }

  const script = existingScript ?? document.createElement('script');
  script.src = src;
  script.async = true;
  script.addEventListener('load', () => {
    script.dataset.loaded = 'true';
    resolve();
  }, { once: true });
  script.addEventListener('error', () => reject(new Error('Could not load Monaco editor')), { once: true });

  if (!existingScript) {
    document.head.appendChild(script);
  }
});

const loadStylesheet = (href: string) => {
  if (document.querySelector<HTMLLinkElement>(`link[href="${href}"]`)) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = href;
  document.head.appendChild(link);
};

const configureMonacoEnvironment = () => {
  window.MonacoEnvironment = {
    getWorkerUrl: () => '/static/monaco/vs/base/worker/workerMain.js'
  };
};

const loadMonaco = () => {
  if (window.monaco) {
    return Promise.resolve(window.monaco);
  }

  if (monacoLoadPromise) {
    return monacoLoadPromise;
  }

  loadStylesheet('/static/monaco/vs/editor/editor.main.css');
  configureMonacoEnvironment();

  monacoLoadPromise = loadScript('/static/monaco/vs/loader.js')
    .then(() => new Promise<MonacoApi>((resolve, reject) => {
      if (!window.require) {
        reject(new Error('Monaco loader is unavailable'));
        return;
      }

      window.require.config({ paths: { vs: '/static/monaco/vs' } });
      window.require(['vs/editor/editor.main'], () => {
        if (!window.monaco) {
          reject(new Error('Monaco editor is unavailable'));
          return;
        }

        resolve(window.monaco);
      }, reject);
    }));

  return monacoLoadPromise;
};

const inferLanguage = (filePath: string) => {
  const lowerPath = filePath.toLowerCase();
  if (lowerPath.endsWith('.ts') || lowerPath.endsWith('.tsx')) return 'typescript';
  if (lowerPath.endsWith('.js') || lowerPath.endsWith('.jsx') || lowerPath.endsWith('.mjs') || lowerPath.endsWith('.cjs')) return 'javascript';
  if (lowerPath.endsWith('.json')) return 'json';
  if (lowerPath.endsWith('.md')) return 'markdown';
  if (lowerPath.endsWith('.css')) return 'css';
  if (lowerPath.endsWith('.html')) return 'html';
  if (lowerPath.endsWith('.sh')) return 'shell';
  if (lowerPath.endsWith('.yml') || lowerPath.endsWith('.yaml')) return 'yaml';
  if (lowerPath.endsWith('.rs')) return 'rust';
  if (lowerPath.endsWith('.java')) return 'java';
  if (lowerPath.endsWith('.kt')) return 'kotlin';
  if (lowerPath.endsWith('.py')) return 'python';
  if (lowerPath.endsWith('.go')) return 'go';
  return 'plaintext';
};

const layoutEditor = () => {
  if (!editor || !editorElement.value) {
    return;
  }

  const { clientWidth, clientHeight } = editorElement.value;
  if (clientWidth <= 0 || clientHeight <= 0) {
    return;
  }

  editor.layout({ width: clientWidth, height: clientHeight });
};

const scheduleEditorLayout = () => {
  requestAnimationFrame(() => {
    layoutEditor();
    setTimeout(layoutEditor, 50);
    setTimeout(layoutEditor, 150);
  });
};

const disposeModels = () => {
  const previousOriginalModel = originalModel;
  const previousModifiedModel = modifiedModel;

  originalModel = null;
  modifiedModel = null;
  previousOriginalModel?.dispose();
  previousModifiedModel?.dispose();
};

const clearEditorModel = () => {
  editor?.setModel(null);
  disposeModels();
};

const mountContents = () => {
  if (!editor || !monaco.value) {
    return;
  }

  if (!props.contents) {
    clearEditorModel();
    return;
  }

  const previousOriginalModel = originalModel;
  const previousModifiedModel = modifiedModel;
  const language = inferLanguage(props.filePath);
  const nextOriginalModel = monaco.value.editor.createModel(props.contents.originalContent, language);
  const nextModifiedModel = monaco.value.editor.createModel(props.contents.modifiedContent, language);

  originalModel = nextOriginalModel;
  modifiedModel = nextModifiedModel;
  editor.setModel({ original: nextOriginalModel, modified: nextModifiedModel });
  previousOriginalModel?.dispose();
  previousModifiedModel?.dispose();
  editor.updateOptions({ renderSideBySide: props.isDiff });
  nextTick(scheduleEditorLayout);
};

const createEditor = async () => {
  if (!editorElement.value) {
    return;
  }

  try {
    monaco.value = await loadMonaco();
    monaco.value.editor.defineTheme('wade-review', {
      base: 'vs-dark',
      inherit: true,
      rules: [],
      colors: {
        'editor.background': '#17181c',
        'diffEditor.insertedTextBackground': '#50fa7b26',
        'diffEditor.removedTextBackground': '#ff555526'
      }
    });
    monaco.value.editor.setTheme('wade-review');
    editor = monaco.value.editor.createDiffEditor(editorElement.value, {
      automaticLayout: true,
      readOnly: true,
      originalEditable: false,
      renderSideBySide: props.isDiff,
      scrollBeyondLastLine: false,
      lineNumbersMinChars: 4,
      minimap: { enabled: true, renderCharacters: false },
      renderOverviewRuler: true,
      wordWrap: 'on',
      diffWordWrap: 'on'
    });
    resizeObserver = new ResizeObserver(layoutEditor);
    resizeObserver.observe(editorElement.value);
    statusMessage.value = '';
    isEditorReady.value = true;
    mountContents();
    scheduleEditorLayout();
  } catch (error) {
    hasEditorError.value = true;
    statusMessage.value = error instanceof Error ? error.message : 'Could not load Monaco editor';
  }
};

watch(() => [props.contents, props.filePath, props.isDiff] as const, () => {
  mountContents();
});

onMounted(() => {
  void createEditor();
});

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  clearEditorModel();
  editor?.dispose();
});
</script>

<template>
  <section class="review-diff-viewer" aria-label="Review diff viewer">
    <div v-if="placeholderText" class="review-diff-placeholder">{{ placeholderText }}</div>
    <section ref="editorElement" class="review-diff-editor" :data-hidden="String(Boolean(placeholderText))"></section>
  </section>
</template>

<style scoped>
.review-diff-viewer {
  position: relative;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--window);
}

.review-diff-editor {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
}

.review-diff-editor :global(.monaco-diff-editor),
.review-diff-editor :global(.monaco-editor) {
  height: 100% !important;
}

.review-diff-editor[data-hidden="true"] {
  visibility: hidden;
}

.review-diff-placeholder {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: var(--muted);
  font-size: 13px;
  text-align: center;
}
</style>
