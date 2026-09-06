<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import {
  computed,
  Fragment,
  h,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  render as renderVNode,
  shallowRef,
  watch
} from 'vue';
import type { ReviewAnnotation } from '@/api/generated/wade';
import type { CommentSide, ReviewComment, ReviewFileContents } from '@/types/review';
import ReviewAgentAnnotation from './ReviewAgentAnnotation.vue';
import ReviewCommentEditor from './ReviewCommentEditor.vue';

type EditorLayoutDimension = {
  width: number;
  height: number;
};

type Disposable = {
  dispose: () => void;
};

type MonacoMouseEvent = {
  target: {
    type: number;
    position?: {
      lineNumber: number;
    };
  };
};

type MutableViewZone = {
  afterLineNumber: number;
  heightInPx: number;
  domNode: HTMLElement;
};

type ViewZoneAccessor = {
  addZone: (zone: MutableViewZone) => string;
  layoutZone: (id: string) => void;
  removeZone: (id: string) => void;
};

type MonacoCodeEditor = {
  changeViewZones: (callback: (accessor: ViewZoneAccessor) => void) => void;
  deltaDecorations: (oldDecorations: string[], newDecorations: Array<Record<string, unknown>>) => string[];
  getScrollLeft: () => number;
  getScrollTop: () => number;
  onMouseDown: (callback: (event: MonacoMouseEvent) => void) => Disposable;
  setScrollLeft: (scrollLeft: number) => void;
  setScrollTop: (scrollTop: number) => void;
  updateOptions: (options: Record<string, unknown>) => void;
};

type MonacoEditor = {
  dispose: () => void;
  getModifiedEditor: () => MonacoCodeEditor;
  getOriginalEditor: () => MonacoCodeEditor;
  layout: (dimension?: EditorLayoutDimension) => void;
  setModel: (model: { original: unknown; modified: unknown } | null) => void;
  updateOptions: (options: Record<string, unknown>) => void;
};

type MonacoModel = {
  dispose: () => void;
};

type MonacoApi = {
  Range: new (startLineNumber: number, startColumn: number, endLineNumber: number, endColumn: number) => unknown;
  editor: {
    MouseTargetType: Record<string, number>;
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

type InlineCommentSide = Exclude<CommentSide, 'file'>;

type ActiveViewZone = {
  id: string;
  editor: MonacoCodeEditor;
  zone: MutableViewZone;
  contentElement: HTMLElement;
  lineNumber: number;
  side: InlineCommentSide;
  observedElements: Set<HTMLElement>;
};

type ScrollPosition = {
  originalLeft: number;
  originalTop: number;
  modifiedLeft: number;
  modifiedTop: number;
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
  annotations: ReviewAnnotation[];
  comments: ReviewComment[];
  contents: ReviewFileContents | null;
  filePath: string;
  hideUnchanged: boolean;
  isDiff: boolean;
  isLoading: boolean;
  renderSideBySide: boolean;
  scrollKey: string;
  wrapLines: boolean;
}>();

const emit = defineEmits<{
  addLineComment: [payload: { side: InlineCommentSide; lineNumber: number }];
  deleteComment: [commentId: string];
  toggleCommentKind: [commentId: string];
  updateCommentBody: [payload: { commentId: string; body: string }];
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
let editorResizeObserver: ResizeObserver | null = null;
let viewZoneResizeObserver: ResizeObserver | null = null;
let viewZoneMeasurementFrame: number | null = null;
let mouseDisposables: Disposable[] = [];
let originalDecorations: string[] = [];
let modifiedDecorations: string[] = [];
let activeViewZones: ActiveViewZone[] = [];
let mountedScrollKey = '';
const scrollPositions = new Map<string, ScrollPosition>();

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

const loadScript = (src: string) =>
  new Promise<void>((resolve, reject) => {
    const existingScript = document.querySelector<HTMLScriptElement>(`script[src="${src}"]`);
    if (existingScript?.dataset.loaded === 'true') {
      resolve();
      return;
    }

    const script = existingScript ?? document.createElement('script');
    script.src = src;
    script.async = true;
    script.addEventListener(
      'load',
      () => {
        script.dataset.loaded = 'true';
        resolve();
      },
      { once: true }
    );
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

  monacoLoadPromise = loadScript('/static/monaco/vs/loader.js').then(
    () =>
      new Promise<MonacoApi>((resolve, reject) => {
        if (!window.require) {
          reject(new Error('Monaco loader is unavailable'));
          return;
        }

        window.require.config({ paths: { vs: '/static/monaco/vs' } });
        window.require(
          ['vs/editor/editor.main'],
          () => {
            if (!window.monaco) {
              reject(new Error('Monaco editor is unavailable'));
              return;
            }

            resolve(window.monaco);
          },
          reject
        );
      })
  );

  return monacoLoadPromise;
};

const languageByFileExtension: Record<string, string> = {
  '.cjs': 'javascript',
  '.cs': 'csharp',
  '.css': 'css',
  '.go': 'go',
  '.html': 'html',
  '.java': 'java',
  '.js': 'javascript',
  '.json': 'json',
  '.jsx': 'javascript',
  '.kt': 'kotlin',
  '.md': 'markdown',
  '.mjs': 'javascript',
  '.py': 'python',
  '.rs': 'rust',
  '.sh': 'shell',
  '.sql': 'sql',
  '.ts': 'typescript',
  '.tsx': 'typescript',
  '.vue': 'html',
  '.yaml': 'yaml',
  '.yml': 'yaml'
};

const inferLanguage = (filePath: string) => {
  const lowerPath = filePath.toLowerCase();
  const languageEntry = Object.entries(languageByFileExtension).find(([extension]) => lowerPath.endsWith(extension));

  return languageEntry?.[1] ?? 'plaintext';
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

const editorReviewOptions = () => ({
  renderSideBySide: props.isDiff && props.renderSideBySide,
  diffWordWrap: props.wrapLines ? 'on' : 'off',
  hideUnchangedRegions: {
    enabled: props.isDiff && props.hideUnchanged,
    contextLineCount: 4,
    minimumLineCount: 2,
    revealLineCount: 12
  }
});

const applyEditorReviewOptions = () => {
  if (!editor) {
    return;
  }

  editor.updateOptions(editorReviewOptions());
  editor.getOriginalEditor().updateOptions({ wordWrap: props.wrapLines ? 'on' : 'off' });
  editor.getModifiedEditor().updateOptions({ wordWrap: props.wrapLines ? 'on' : 'off' });
  scheduleEditorLayout();
  measureActiveViewZones();
};

const saveScrollPosition = () => {
  if (!editor || mountedScrollKey === '') {
    return;
  }

  const originalEditor = editor.getOriginalEditor();
  const modifiedEditor = editor.getModifiedEditor();
  scrollPositions.set(mountedScrollKey, {
    originalLeft: originalEditor.getScrollLeft(),
    originalTop: originalEditor.getScrollTop(),
    modifiedLeft: modifiedEditor.getScrollLeft(),
    modifiedTop: modifiedEditor.getScrollTop()
  });
};

const restoreScrollPosition = () => {
  if (!editor || props.scrollKey === '') {
    return;
  }

  const position = scrollPositions.get(props.scrollKey);
  if (!position) {
    return;
  }

  const originalEditor = editor.getOriginalEditor();
  const modifiedEditor = editor.getModifiedEditor();
  originalEditor.setScrollLeft(position.originalLeft);
  originalEditor.setScrollTop(position.originalTop);
  modifiedEditor.setScrollLeft(position.modifiedLeft);
  modifiedEditor.setScrollTop(position.modifiedTop);
};

const scheduleScrollRestore = () => {
  requestAnimationFrame(() => {
    restoreScrollPosition();
    setTimeout(restoreScrollPosition, 50);
    setTimeout(restoreScrollPosition, 150);
  });
};

const inlineComments = () => props.comments.filter((comment) => comment.side !== 'file' && comment.startLine != null);

const inlineContentSignature = () =>
  [
    ...inlineComments().map((comment) => `comment:${comment.id}:${comment.side}:${comment.startLine}`),
    ...props.annotations.map(
      (annotation) => `annotation:${annotation.id}:${annotation.side}:${annotation.startLine}:${annotation.endLine}`
    )
  ].join('|');

const sideLabel = (side: InlineCommentSide) => (side === 'original' ? 'Original' : 'Modified');

const annotationLocationLabel = (annotation: ReviewAnnotation) => {
  const lineRange =
    annotation.endLine === annotation.startLine
      ? `${annotation.startLine}`
      : `${annotation.startLine}-${annotation.endLine}`;
  return `${sideLabel(annotation.side)}:${lineRange}`;
};

const groupedInlineContent = (side: InlineCommentSide) => {
  const groups = new Map<number, { comments: ReviewComment[]; annotations: ReviewAnnotation[] }>();

  for (const comment of inlineComments()) {
    if (comment.side !== side || comment.startLine == null) {
      continue;
    }

    const group = groups.get(comment.startLine) ?? { comments: [], annotations: [] };
    group.comments.push(comment);
    groups.set(comment.startLine, group);
  }

  for (const annotation of props.annotations) {
    if (annotation.side !== side) {
      continue;
    }

    const group = groups.get(annotation.startLine) ?? { comments: [], annotations: [] };
    group.annotations.push(annotation);
    groups.set(annotation.startLine, group);
  }

  return [...groups.entries()]
    .map(([lineNumber, content]) => ({ lineNumber, ...content }))
    .sort((a, b) => a.lineNumber - b.lineNumber);
};

const stopEditorEvent = (event: Event) => {
  event.stopPropagation();
};

const protectInteractiveElement = (element: HTMLElement) => {
  element.addEventListener('pointerdown', stopEditorEvent);
  element.addEventListener('mousedown', stopEditorEvent);
  element.addEventListener('mouseup', stopEditorEvent);
  element.addEventListener('click', stopEditorEvent);
  element.addEventListener('dblclick', stopEditorEvent);
  element.addEventListener('keydown', stopEditorEvent);
};

const viewZoneKey = (side: InlineCommentSide, lineNumber: number) => `${side}:${lineNumber}`;

const syncViewZoneResizeObserver = (activeZone: ActiveViewZone) => {
  const nextObservedElements = new Set<HTMLElement>([activeZone.contentElement]);

  for (const element of activeZone.observedElements) {
    if (!nextObservedElements.has(element)) {
      viewZoneResizeObserver?.unobserve(element);
    }
  }

  for (const element of nextObservedElements) {
    if (!activeZone.observedElements.has(element)) {
      viewZoneResizeObserver?.observe(element);
    }
  }

  activeZone.observedElements = nextObservedElements;
};

const disposeViewZoneContent = (activeZone: ActiveViewZone) => {
  for (const element of activeZone.observedElements) {
    viewZoneResizeObserver?.unobserve(element);
  }
  activeZone.observedElements.clear();
  renderVNode(null, activeZone.contentElement);
  activeZone.contentElement.remove();
};

const clearViewZones = () => {
  for (const activeZone of activeViewZones) {
    disposeViewZoneContent(activeZone);
  }

  if (!editor) {
    activeViewZones = [];
    return;
  }

  const originalEditor = editor.getOriginalEditor();
  const modifiedEditor = editor.getModifiedEditor();

  originalEditor.changeViewZones((accessor) => {
    for (const activeZone of activeViewZones) {
      if (activeZone.editor === originalEditor) {
        accessor.removeZone(activeZone.id);
      }
    }
  });

  modifiedEditor.changeViewZones((accessor) => {
    for (const activeZone of activeViewZones) {
      if (activeZone.editor === modifiedEditor) {
        accessor.removeZone(activeZone.id);
      }
    }
  });

  activeViewZones = [];
};

const renderInlineContent = (
  container: HTMLElement,
  side: InlineCommentSide,
  comments: ReviewComment[],
  annotations: ReviewAnnotation[]
) => {
  renderVNode(
    h(Fragment, null, [
      ...annotations.map((annotation) =>
        h(ReviewAgentAnnotation, {
          key: `annotation:${annotation.id}`,
          annotation,
          locationLabel: annotationLocationLabel(annotation)
        })
      ),
      ...comments.map((comment) =>
        h(ReviewCommentEditor, {
          key: `comment:${comment.id}`,
          activateOnPointerdown: true,
          comment,
          locationLabel: `${sideLabel(side)}:${comment.startLine}`,
          onDeleteComment: (commentId: string) => emit('deleteComment', commentId),
          onToggleCommentKind: (commentId: string) => emit('toggleCommentKind', commentId),
          onUpdateCommentBody: (payload: { commentId: string; body: string }) => emit('updateCommentBody', payload)
        })
      )
    ]),
    container
  );
};

const measureActiveViewZones = () => {
  if (viewZoneMeasurementFrame !== null) {
    return;
  }

  viewZoneMeasurementFrame = requestAnimationFrame(() => {
    viewZoneMeasurementFrame = null;
    let hasHeightChange = false;

    for (const activeZone of activeViewZones) {
      const measuredHeight = Math.ceil(activeZone.contentElement.getBoundingClientRect().height);
      if (!Number.isFinite(measuredHeight) || measuredHeight <= 0 || measuredHeight === activeZone.zone.heightInPx) {
        continue;
      }

      activeZone.zone.heightInPx = measuredHeight;
      activeZone.editor.changeViewZones((accessor) => accessor.layoutZone(activeZone.id));
      hasHeightChange = true;
    }

    if (hasHeightChange) {
      scheduleEditorLayout();
    }
  });
};

const createInlineViewZone = (
  codeEditor: MonacoCodeEditor,
  side: InlineCommentSide,
  entry: { lineNumber: number; comments: ReviewComment[]; annotations: ReviewAnnotation[] },
  accessor: ViewZoneAccessor
) => {
  const domNode = document.createElement('section');
  domNode.className = 'review-inline-zone';
  domNode.dataset.side = side;
  protectInteractiveElement(domNode);
  const contentElement = document.createElement('section');
  contentElement.className = 'review-inline-zone-content';
  domNode.append(contentElement);
  renderInlineContent(contentElement, side, entry.comments, entry.annotations);

  const zone: MutableViewZone = {
    afterLineNumber: entry.lineNumber,
    heightInPx: Math.max(156, 18 + entry.annotations.length * 110 + entry.comments.length * 156),
    domNode
  };
  const activeZone: ActiveViewZone = {
    id: accessor.addZone(zone),
    editor: codeEditor,
    zone,
    contentElement,
    lineNumber: entry.lineNumber,
    side,
    observedElements: new Set()
  };
  syncViewZoneResizeObserver(activeZone);
  return activeZone;
};

const reconcileInlineViewZones = () => {
  if (!editor || !props.contents) {
    return;
  }

  const nextActiveViewZones: ActiveViewZone[] = [];
  const editorSides: Array<[MonacoCodeEditor, InlineCommentSide]> = [
    [editor.getOriginalEditor(), 'original'],
    [editor.getModifiedEditor(), 'modified']
  ];

  for (const [codeEditor, side] of editorSides) {
    const entries = groupedInlineContent(side);
    const entriesByKey = new Map(entries.map((entry) => [viewZoneKey(side, entry.lineNumber), entry]));
    const existingZones = activeViewZones.filter((activeZone) => activeZone.editor === codeEditor);
    const existingZonesByKey = new Map(
      existingZones.map((activeZone) => [viewZoneKey(activeZone.side, activeZone.lineNumber), activeZone])
    );

    codeEditor.changeViewZones((accessor) => {
      for (const activeZone of existingZones) {
        const entry = entriesByKey.get(viewZoneKey(activeZone.side, activeZone.lineNumber));
        if (!entry) {
          disposeViewZoneContent(activeZone);
          accessor.removeZone(activeZone.id);
          continue;
        }

        renderInlineContent(activeZone.contentElement, side, entry.comments, entry.annotations);
        syncViewZoneResizeObserver(activeZone);
        nextActiveViewZones.push(activeZone);
      }

      for (const entry of entries) {
        if (existingZonesByKey.has(viewZoneKey(side, entry.lineNumber))) {
          continue;
        }

        nextActiveViewZones.push(createInlineViewZone(codeEditor, side, entry, accessor));
      }
    });
  }

  activeViewZones = nextActiveViewZones;
  scheduleEditorLayout();
  measureActiveViewZones();
};

const syncInlineZoneContent = () => {
  for (const activeZone of activeViewZones) {
    const comments = inlineComments().filter(
      (comment) => comment.side === activeZone.side && comment.startLine === activeZone.lineNumber
    );
    const annotations = props.annotations.filter(
      (annotation) => annotation.side === activeZone.side && annotation.startLine === activeZone.lineNumber
    );
    renderInlineContent(activeZone.contentElement, activeZone.side, comments, annotations);
    syncViewZoneResizeObserver(activeZone);
  }
  measureActiveViewZones();
};

const clearCommentDecorations = () => {
  if (!editor) {
    originalDecorations = [];
    modifiedDecorations = [];
    return;
  }

  originalDecorations = editor.getOriginalEditor().deltaDecorations(originalDecorations, []);
  modifiedDecorations = editor.getModifiedEditor().deltaDecorations(modifiedDecorations, []);
};

const syncCommentDecorations = () => {
  if (!editor || !monaco.value) {
    return;
  }

  const originalRanges = [];
  const modifiedRanges = [];

  for (const comment of inlineComments()) {
    const decoration = {
      range: new monaco.value.Range(comment.startLine ?? 1, 1, comment.startLine ?? 1, 1),
      options: {
        isWholeLine: true,
        className: comment.side === 'original' ? 'review-comment-line-original' : 'review-comment-line-modified',
        glyphMarginClassName:
          comment.side === 'original' ? 'review-comment-glyph-original' : 'review-comment-glyph-modified'
      }
    };

    if (comment.side === 'original') {
      originalRanges.push(decoration);
    } else {
      modifiedRanges.push(decoration);
    }
  }

  for (const annotation of props.annotations) {
    const decoration = {
      range: new monaco.value.Range(annotation.startLine, 1, annotation.endLine, 1),
      options: {
        isWholeLine: true,
        className: annotation.side === 'original' ? 'review-agent-line-original' : 'review-agent-line-modified',
        glyphMarginClassName:
          annotation.side === 'original' ? 'review-agent-glyph-original' : 'review-agent-glyph-modified'
      }
    };

    if (annotation.side === 'original') {
      originalRanges.push(decoration);
    } else {
      modifiedRanges.push(decoration);
    }
  }

  originalDecorations = editor.getOriginalEditor().deltaDecorations(originalDecorations, originalRanges);
  modifiedDecorations = editor.getModifiedEditor().deltaDecorations(modifiedDecorations, modifiedRanges);
};

const syncInlineReviewUI = () => {
  syncCommentDecorations();
  reconcileInlineViewZones();
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
  clearViewZones();
  clearCommentDecorations();
  editor?.setModel(null);
  disposeModels();
};

const mountContents = () => {
  if (!editor || !monaco.value) {
    return;
  }

  saveScrollPosition();

  if (!props.contents) {
    mountedScrollKey = '';
    clearEditorModel();
    return;
  }

  const previousOriginalModel = originalModel;
  const previousModifiedModel = modifiedModel;
  const language = inferLanguage(props.filePath);
  const nextOriginalModel = monaco.value.editor.createModel(props.contents.originalContent, language);
  const nextModifiedModel = monaco.value.editor.createModel(props.contents.modifiedContent, language);

  clearViewZones();
  originalModel = nextOriginalModel;
  modifiedModel = nextModifiedModel;
  editor.setModel({ original: nextOriginalModel, modified: nextModifiedModel });
  mountedScrollKey = props.scrollKey;
  previousOriginalModel?.dispose();
  previousModifiedModel?.dispose();
  applyEditorReviewOptions();
  syncInlineReviewUI();
  nextTick(() => {
    scheduleEditorLayout();
    scheduleScrollRestore();
  });
};

const isLineCommentTarget = (targetType: number) => {
  if (!monaco.value) {
    return false;
  }

  const mouseTargetType = monaco.value.editor.MouseTargetType;
  return targetType === mouseTargetType.GUTTER_LINE_NUMBERS || targetType === mouseTargetType.GUTTER_GLYPH_MARGIN;
};

const addLineCommentHandler = (codeEditor: MonacoCodeEditor, side: InlineCommentSide) =>
  codeEditor.onMouseDown((event) => {
    if (!props.contents || !isLineCommentTarget(event.target.type)) {
      return;
    }

    const lineNumber = event.target.position?.lineNumber;
    if (!lineNumber) {
      return;
    }

    emit('addLineComment', { side, lineNumber });
  });

const wireLineCommentHandlers = () => {
  if (!editor) {
    return;
  }

  mouseDisposables = [
    addLineCommentHandler(editor.getOriginalEditor(), 'original'),
    addLineCommentHandler(editor.getModifiedEditor(), 'modified')
  ];
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
      ...editorReviewOptions(),
      scrollBeyondLastLine: false,
      lineNumbersMinChars: 4,
      minimap: { enabled: true, renderCharacters: false },
      renderOverviewRuler: true,
      glyphMargin: true,
      wordWrap: 'on',
      diffWordWrap: 'on'
    });
    editorResizeObserver = new ResizeObserver(() => {
      layoutEditor();
      measureActiveViewZones();
    });
    viewZoneResizeObserver = new ResizeObserver(() => measureActiveViewZones());
    editorResizeObserver.observe(editorElement.value);
    wireLineCommentHandlers();
    statusMessage.value = '';
    isEditorReady.value = true;
    mountContents();
    scheduleEditorLayout();
  } catch (error) {
    hasEditorError.value = true;
    statusMessage.value = error instanceof Error ? error.message : 'Could not load Monaco editor';
  }
};

watch(
  () => [props.contents, props.filePath, props.scrollKey] as const,
  () => {
    mountContents();
  }
);

watch(
  () => [props.hideUnchanged, props.isDiff, props.renderSideBySide, props.wrapLines] as const,
  () => {
    applyEditorReviewOptions();
  }
);

watch(inlineContentSignature, () => {
  syncInlineReviewUI();
});

watch([() => props.comments, () => props.annotations], () => {
  syncInlineZoneContent();
});

onMounted(() => {
  void createEditor();
});

onBeforeUnmount(() => {
  saveScrollPosition();
  editorResizeObserver?.disconnect();
  viewZoneResizeObserver?.disconnect();
  if (viewZoneMeasurementFrame !== null) {
    cancelAnimationFrame(viewZoneMeasurementFrame);
  }
  mouseDisposables.forEach((disposable) => disposable.dispose());
  clearEditorModel();
  editor?.dispose();
});
</script>

<template>
  <section class="review-diff-viewer" aria-label="Review diff viewer">
    <div v-if="placeholderText" class="review-diff-placeholder">{{ placeholderText }}</div>
    <section ref="editorElement" class="review-diff-editor" :data-hidden="String(Boolean(placeholderText))"></section>
    <section class="review-diff-accessible-annotations" aria-label="Agent annotations">
      <ReviewAgentAnnotation
        v-for="annotation in props.annotations"
        :key="annotation.id"
        :annotation="annotation"
        :location-label="annotationLocationLabel(annotation)"
      />
    </section>
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

.review-diff-editor :global(.monaco-editor .view-zones) {
  z-index: 10;
}

.review-diff-editor :global(.review-comment-line-original) {
  background: rgb(241 250 140 / 18%);
}

.review-diff-editor :global(.review-comment-line-modified) {
  background: rgb(139 233 253 / 16%);
}

.review-diff-editor :global(.review-agent-line-original),
.review-diff-editor :global(.review-agent-line-modified) {
  background: rgb(189 147 249 / 14%);
}

.review-diff-editor :global(.review-comment-glyph-original),
.review-diff-editor :global(.review-comment-glyph-modified) {
  width: 8px !important;
  height: 8px !important;
  margin-left: 6px;
  margin-top: 5px;
  border-radius: 0;
}

.review-diff-editor :global(.review-comment-glyph-original) {
  background: #f1fa8c;
}

.review-diff-editor :global(.review-comment-glyph-modified) {
  background: #8be9fd;
}

.review-diff-editor :global(.review-agent-glyph-original),
.review-diff-editor :global(.review-agent-glyph-modified) {
  width: 9px !important;
  height: 9px !important;
  margin-left: 5px;
  margin-top: 5px;
  border: 1px solid #bd93f9;
  background: transparent;
  transform: rotate(45deg);
}

.review-diff-editor :global(.review-inline-zone) {
  overflow: hidden;
}

.review-diff-editor :global(.review-inline-zone-content) {
  display: grid;
  gap: 8px;
  padding: 8px 14px 10px;
  border-top: 1px solid var(--text);
  border-bottom: 1px solid var(--text);
  background: var(--window);
  color: var(--text);
}

.review-diff-editor[data-hidden='true'] {
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

.review-diff-accessible-annotations {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
