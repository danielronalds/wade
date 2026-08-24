<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import {
  BookOpen,
  Check,
  Code2,
  Columns2,
  EyeOff,
  ListCollapse,
  MessageSquarePlus,
  NotebookPen,
  Rows2,
  Send,
  TextWrap,
  X
} from '@lucide/vue';
import { storeToRefs } from 'pinia';
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { createReviewSnapshot, deleteReviewSnapshot, getReviewSnapshotFileContents } from '@/api/generated/wade';
import { pasteIntoAgentTerminal } from '@/features/terminal-session/composables/useAgentTerminalInput';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';
import {
  isReviewInProgressState,
  type CommentSide,
  type DraftReviewComment,
  ReviewComment,
  ReviewCommentKind,
  ReviewData,
  ReviewFile,
  ReviewFileComparison,
  ReviewFileContents,
  ReviewScope
} from '@/types/review';
import ReviewDiffViewer from '@/views/workspace/tabs/review/components/ReviewDiffViewer.vue';
import ReviewMarkdownViewer from '@/views/workspace/tabs/review/components/ReviewMarkdownViewer.vue';

const props = defineProps<{
  workspaceId: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  requestTerminalTab: [];
}>();

type FileRequestState = {
  contents: ReviewFileContents | null;
  error: string;
  isLoading: boolean;
};

type ReviewFileTreeNode = {
  name: string;
  path: string;
  kind: 'dir' | 'file';
  children: Map<string, ReviewFileTreeNode>;
  file: ReviewFile | null;
};

type ReviewFileTreeRow = {
  id: string;
  name: string;
  path: string;
  depth: number;
} & ({ kind: 'dir' } | { kind: 'file'; file: ReviewFile });

const workspaceSessionStore = useWorkspaceSessionStore();
const {
  isReviewOverallNoteOpen: isOverallNoteOpen,
  reviewActiveFileId: activeFileId,
  reviewActiveScope: activeScope,
  reviewCollapsedDirectories: collapsedDirectories,
  reviewComments: comments,
  reviewData,
  reviewDraftComment: draftComment,
  reviewDraftCommentBody: draftCommentBody,
  reviewDraftCommentKind: draftCommentKind,
  reviewFilterText: filterText,
  reviewHideUnchanged: hideUnchanged,
  reviewOverallComment: overallComment,
  reviewOverallNoteDraft: overallNoteDraft,
  reviewRenderSideBySide: renderSideBySide,
  reviewReviewedFiles: reviewedFiles,
  reviewState: state,
  reviewWrapLines: wrapLines
} = storeToRefs(workspaceSessionStore);

const errorMessage = ref('');
const sendErrorMessage = ref('');
const fileRequestStates = ref<Record<string, FileRequestState>>({});
const isSendingPrompt = ref(false);
const renderMarkdown = ref(true);
const startButton = ref<HTMLButtonElement | null>(null);
const searchInput = ref<HTMLInputElement | null>(null);
const draftCommentTextarea = ref<HTMLTextAreaElement | null>(null);
const overallNoteTextarea = ref<HTMLTextAreaElement | null>(null);
let reviewLoadRun = 0;
let reviewLoadAbortController: AbortController | null = null;
const fileRequestAbortControllers = new Map<string, AbortController>();

const scopeOptions = computed<Array<{ id: ReviewScope; label: string }>>(() => {
  const localScopeOptions: Array<{ id: ReviewScope; label: string }> = [
    { id: 'working-tree', label: 'Working tree' },
    { id: 'last-commit', label: 'Last commit' },
    { id: 'current', label: 'Current files' }
  ];

  if (!reviewData.value?.pullRequest) {
    return localScopeOptions;
  }

  return [
    localScopeOptions[0],
    { id: 'pull-request', label: `PR #${reviewData.value.pullRequest.number}` },
    ...localScopeOptions.slice(1)
  ];
});

const cacheKey = (scope: ReviewScope, fileId: string) => `${scope}:${fileId}`;

const scopeLabel = (scope: ReviewScope) => {
  switch (scope) {
    case 'pull-request':
      return 'Pull request';
    case 'working-tree':
      return 'Working tree';
    case 'last-commit':
      return 'Last commit';
    default:
      return 'Current files';
  }
};

const promptScopeLabel = (scope: ReviewScope) => scopeLabel(scope).toLowerCase();

const commentKindLabel = (kind: ReviewCommentKind) => (kind === 'question' ? 'Question' : 'Feedback');

const commentSideLabel = (side: CommentSide) => {
  if (side === 'original') return 'original';
  if (side === 'modified') return 'modified';
  return 'whole file';
};

const getComparison = (file: ReviewFile | null, scope: ReviewScope): ReviewFileComparison | null => {
  if (!file) {
    return null;
  }

  if (scope === 'pull-request') {
    return file.pullRequest;
  }

  if (scope === 'working-tree') {
    return file.gitDiff;
  }

  if (scope === 'last-commit') {
    return file.lastCommit;
  }

  return null;
};

const scopeCounts = computed<Record<ReviewScope, number>>(() => ({
  'pull-request': reviewData.value?.files.filter((file) => file.inPullRequest).length ?? 0,
  'working-tree': reviewData.value?.files.filter((file) => file.inGitDiff).length ?? 0,
  'last-commit': reviewData.value?.files.filter((file) => file.inLastCommit).length ?? 0,
  current: reviewData.value?.files.filter((file) => file.hasWorkingTreeFile).length ?? 0
}));

const scopedFiles = computed(() => {
  const files = reviewData.value?.files ?? [];
  if (activeScope.value === 'pull-request') {
    return files.filter((file) => file.inPullRequest);
  }

  if (activeScope.value === 'working-tree') {
    return files.filter((file) => file.inGitDiff);
  }

  if (activeScope.value === 'last-commit') {
    return files.filter((file) => file.inLastCommit);
  }

  return files.filter((file) => file.hasWorkingTreeFile);
});

const normalizeQuery = (query: string) => query.trim().toLowerCase().replace(/\s+/g, '');

const getBaseName = (filePath: string) => filePath.split('/').pop() || filePath;

const scoreSubsequence = (query: string, candidate: string) => {
  if (!query) {
    return 0;
  }

  let queryIndex = 0;
  let score = 0;
  let firstMatchIndex = -1;
  let previousMatchIndex = -2;

  for (let index = 0; index < candidate.length && queryIndex < query.length; index += 1) {
    if (candidate[index] !== query[queryIndex]) {
      continue;
    }

    if (firstMatchIndex === -1) {
      firstMatchIndex = index;
    }

    score += 10;
    if (index === previousMatchIndex + 1) {
      score += 8;
    }

    const previousCharacter = index > 0 ? candidate[index - 1] : '';
    if (
      index === 0 ||
      previousCharacter === '/' ||
      previousCharacter === '_' ||
      previousCharacter === '-' ||
      previousCharacter === '.'
    ) {
      score += 12;
    }

    previousMatchIndex = index;
    queryIndex += 1;
  }

  if (queryIndex !== query.length) {
    return -1;
  }

  if (firstMatchIndex >= 0) {
    score += Math.max(0, 20 - firstMatchIndex);
  }

  return score;
};

const getFileSearchScore = (query: string, file: ReviewFile) => {
  const normalizedQuery = normalizeQuery(query);
  if (!normalizedQuery) {
    return 0;
  }

  const filePath = file.path.toLowerCase();
  const baseName = getBaseName(filePath);
  const pathScore = scoreSubsequence(normalizedQuery, filePath);
  const baseScore = scoreSubsequence(normalizedQuery, baseName);
  let score = Math.max(pathScore, baseScore >= 0 ? baseScore + 40 : -1);

  if (score < 0) {
    return -1;
  }

  if (baseName === normalizedQuery) {
    score += 200;
  } else if (baseName.startsWith(normalizedQuery)) {
    score += 120;
  } else if (filePath.includes(normalizedQuery)) {
    score += 35;
  }

  return score;
};

const normalizedFilterText = computed(() => normalizeQuery(filterText.value));
const filteredFiles = computed(() => {
  if (!normalizedFilterText.value) {
    return [...scopedFiles.value];
  }

  return scopedFiles.value
    .map((file) => ({ file, score: getFileSearchScore(filterText.value, file) }))
    .filter((entry) => entry.score >= 0)
    .sort((a, b) => b.score - a.score || a.file.path.localeCompare(b.file.path))
    .map((entry) => entry.file);
});

const buildFileTree = (files: ReviewFile[]) => {
  const root: ReviewFileTreeNode = { name: '', path: '', kind: 'dir', children: new Map(), file: null };

  for (const file of files) {
    const parts = file.path.split('/');
    let node = root;
    let currentPath = '';

    parts.forEach((part, index) => {
      const isLeaf = index === parts.length - 1;
      currentPath = currentPath ? `${currentPath}/${part}` : part;

      if (!node.children.has(part)) {
        node.children.set(part, {
          name: part,
          path: currentPath,
          kind: isLeaf ? 'file' : 'dir',
          children: new Map(),
          file: isLeaf ? file : null
        });
      }

      const child = node.children.get(part);
      if (!child) {
        return;
      }

      if (isLeaf) {
        child.file = file;
      }

      node = child;
    });
  }

  return root;
};

const flattenFileTree = (node: ReviewFileTreeNode, depth = 0): ReviewFileTreeRow[] =>
  [...node.children.values()]
    .sort((a, b) => {
      if (a.kind !== b.kind) {
        return a.kind === 'dir' ? -1 : 1;
      }

      return a.name.localeCompare(b.name);
    })
    .flatMap((child): ReviewFileTreeRow[] => {
      if (child.kind === 'file' && child.file) {
        return [{ kind: 'file', id: child.file.id, name: child.name, path: child.path, depth, file: child.file }];
      }

      const row: ReviewFileTreeRow = { kind: 'dir', id: child.path, name: child.name, path: child.path, depth };
      if (collapsedDirectories.value[child.path]) {
        return [row];
      }

      return [row, ...flattenFileTree(child, depth + 1)];
    });

const fileTreeRows = computed(() =>
  normalizedFilterText.value ? [] : flattenFileTree(buildFileTree(filteredFiles.value))
);

const activeFile = computed(() => reviewData.value?.files.find((file) => file.id === activeFileId.value) ?? null);
const activeComparison = computed(() => getComparison(activeFile.value, activeScope.value));
const activeFilePath = computed(() => activeComparison.value?.displayPath ?? activeFile.value?.path ?? '');
const isActiveFileMarkdown = computed(() => /\.(md|markdown|mdown|mkd)$/i.test(activeFilePath.value));
const showRenderedMarkdown = computed(() => isActiveFileMarkdown.value && renderMarkdown.value);
const activeCacheKey = computed(() => (activeFile.value ? cacheKey(activeScope.value, activeFile.value.id) : ''));
const activeFileRequestState = computed(() =>
  activeCacheKey.value ? fileRequestStates.value[activeCacheKey.value] : undefined
);
const activeContents = computed(() => activeFileRequestState.value?.contents ?? null);
const isActiveFileLoading = computed(() => activeFileRequestState.value?.isLoading === true);
const activeFileError = computed(() => activeFileRequestState.value?.error ?? '');
const visibleErrorMessage = computed(() => sendErrorMessage.value || activeFileError.value);
const startErrorMessage = computed(
  () =>
    errorMessage.value ||
    (state.value === 'error' ? 'Could not restore the saved review. Reload to retry, or start a new review.' : '')
);
const canStartReview = computed(() => state.value === 'idle' || state.value === 'error');
const canCancelReview = computed(() => isReviewInProgressState(state.value));
const hasReviewableFiles = computed(() => (reviewData.value?.files.length ?? 0) > 0);
const activeFileComments = computed(() =>
  comments.value.filter((comment) => comment.fileId === activeFileId.value && comment.scope === activeScope.value)
);
const activeFileInlineComments = computed(() => activeFileComments.value.filter((comment) => comment.side !== 'file'));
const activeFileFileComments = computed(() => activeFileComments.value.filter((comment) => comment.side === 'file'));
const isActiveFileReviewed = computed(
  () => activeFileId.value != null && reviewedFiles.value[activeFileId.value] === true
);
const reviewedFileCount = computed(
  () => scopedFiles.value.filter((file) => reviewedFiles.value[file.id] === true).length
);
const trimmedComments = computed(() =>
  comments.value
    .map((comment) => ({ ...comment, body: comment.body.trim() }))
    .filter((comment) => comment.body.length > 0)
);
const canFinishReview = computed(
  () => !isSendingPrompt.value && (trimmedComments.value.length > 0 || overallComment.value.trim().length > 0)
);
const reviewScrollKey = computed(() => (activeFileId.value ? `${activeScope.value}:${activeFileId.value}` : ''));
const hideUnchangedButtonLabel = computed(() => (hideUnchanged.value ? 'Show full file' : 'Show changed areas only'));
const renderSideBySideButtonLabel = computed(() =>
  renderSideBySide.value ? 'Use inline diff' : 'Use side-by-side diff'
);
const wrapLinesButtonLabel = computed(() => (wrapLines.value ? 'Disable line wrap' : 'Enable line wrap'));
const markdownViewButtonLabel = computed(() =>
  showRenderedMarkdown.value ? 'Show Markdown source' : 'Show rendered Markdown'
);
const reviewInstructions = computed(() =>
  showRenderedMarkdown.value
    ? 'Click a rendered content block to comment. Use j/k or arrows for files, r to mark reviewed, / to search.'
    : 'Click a line number to comment. Use j/k or arrows for files, r to mark reviewed, / to search.'
);
const draftCommentTitle = computed(() => {
  if (!draftComment.value) {
    return '';
  }

  if (draftComment.value.side === 'file') {
    return `Comment on ${draftComment.value.filePath}`;
  }

  return `Comment on ${draftComment.value.filePath}:${draftComment.value.startLine}`;
});
const draftCommentDescription = computed(() => {
  if (!draftComment.value) {
    return '';
  }

  return `${scopeLabel(draftComment.value.scope)} · ${commentSideLabel(draftComment.value.side)}`;
});

const statusLabel = (status: string | null) => {
  if (!status) {
    return '';
  }

  return status.charAt(0).toUpperCase() + status.slice(1);
};

const getFileStatus = (file: ReviewFile) => getComparison(file, activeScope.value)?.status ?? file.worktreeStatus;

const isFileReviewed = (file: ReviewFile) => reviewedFiles.value[file.id] === true;

const commentCountForFile = (file: ReviewFile) =>
  comments.value.filter(
    (comment) => comment.fileId === file.id && comment.scope === activeScope.value && comment.body.trim().length > 0
  ).length;

const toggleActiveFileReviewed = () => {
  const file = activeFile.value;
  if (!file) {
    return;
  }

  reviewedFiles.value = {
    ...reviewedFiles.value,
    [file.id]: !isFileReviewed(file)
  };
};

const toggleHideUnchanged = () => {
  if (!activeComparison.value) {
    return;
  }

  hideUnchanged.value = !hideUnchanged.value;
};

const toggleRenderSideBySide = () => {
  if (!activeComparison.value) {
    return;
  }

  renderSideBySide.value = !renderSideBySide.value;
};

const toggleWrapLines = () => {
  wrapLines.value = !wrapLines.value;
};

const toggleMarkdownView = () => {
  if (!isActiveFileMarkdown.value) {
    return;
  }

  renderMarkdown.value = !renderMarkdown.value;
};

const toggleDirectoryCollapsed = (directoryPath: string) => {
  collapsedDirectories.value = {
    ...collapsedDirectories.value,
    [directoryPath]: !collapsedDirectories.value[directoryPath]
  };
};

const selectInitialScope = (data: ReviewData): ReviewScope => {
  if (data.files.some((file) => file.inGitDiff)) {
    return 'working-tree';
  }

  if (data.pullRequest && data.files.some((file) => file.inPullRequest)) {
    return 'pull-request';
  }

  if (data.files.some((file) => file.inLastCommit)) {
    return 'last-commit';
  }

  return 'current';
};

const ensureActiveFile = () => {
  if (scopedFiles.value.some((file) => file.id === activeFileId.value)) {
    return;
  }

  activeFileId.value = scopedFiles.value[0]?.id ?? null;
};

const selectScope = (scope: ReviewScope) => {
  if (scopeCounts.value[scope] === 0) {
    return;
  }

  activeScope.value = scope;
  ensureActiveFile();
  void loadActiveFileContents();
};

const selectFile = (file: ReviewFile) => {
  activeFileId.value = file.id;
  sendErrorMessage.value = '';
  void loadActiveFileContents();
};

const selectFileByIndex = (index: number) => {
  const file = filteredFiles.value[index];
  if (!file) {
    return;
  }

  selectFile(file);
};

const selectAdjacentFile = (offset: number) => {
  if (filteredFiles.value.length === 0) {
    return;
  }

  const currentIndex = filteredFiles.value.findIndex((file) => file.id === activeFileId.value);
  const nextIndex =
    currentIndex < 0 ? 0 : (currentIndex + offset + filteredFiles.value.length) % filteredFiles.value.length;
  selectFileByIndex(nextIndex);
};

const isKeyboardTextTarget = (target: EventTarget | null) => {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  const tagName = target.tagName.toLowerCase();
  return tagName === 'input' || tagName === 'textarea' || tagName === 'select' || target.isContentEditable;
};

const handleReviewKeydown = (event: KeyboardEvent) => {
  if (
    !props.isActive ||
    state.value !== 'ready' ||
    draftComment.value ||
    isOverallNoteOpen.value ||
    isKeyboardTextTarget(event.target)
  ) {
    return;
  }

  if (event.key === '/' && !event.metaKey && !event.ctrlKey && !event.altKey) {
    event.preventDefault();
    searchInput.value?.focus();
    return;
  }

  if (event.key === 'ArrowDown' || event.key.toLowerCase() === 'j') {
    event.preventDefault();
    selectAdjacentFile(1);
    return;
  }

  if (event.key === 'ArrowUp' || event.key.toLowerCase() === 'k') {
    event.preventDefault();
    selectAdjacentFile(-1);
    return;
  }

  if (event.key.toLowerCase() === 'r') {
    event.preventDefault();
    toggleActiveFileReviewed();
  }
};

const isAbortError = (error: unknown) =>
  typeof error === 'object' && error !== null && 'name' in error && (error as { name?: unknown }).name === 'AbortError';

const abortReviewRequests = () => {
  reviewLoadAbortController?.abort();
  reviewLoadAbortController = null;

  fileRequestAbortControllers.forEach((abortController) => {
    abortController.abort();
  });
  fileRequestAbortControllers.clear();
};

const resetReview = () => {
  reviewLoadRun += 1;
  abortReviewRequests();
  workspaceSessionStore.clearReview(props.workspaceId);
  errorMessage.value = '';
  sendErrorMessage.value = '';
  fileRequestStates.value = {};
};

const deleteActiveSnapshot = async () => {
  const snapshotId = reviewData.value?.id;
  if (!snapshotId) {
    return;
  }

  await deleteReviewSnapshot(snapshotId);
};

const cancelReview = async () => {
  if (!canCancelReview.value) {
    return;
  }

  try {
    await deleteActiveSnapshot();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Could not cancel review';
    return;
  }

  resetReview();
  emit('requestTerminalTab');
  await nextTick();
};

const setFileRequestState = (key: string, value: FileRequestState) => {
  fileRequestStates.value = {
    ...fileRequestStates.value,
    [key]: value
  };
};

const startReview = async () => {
  if (!canStartReview.value) {
    return;
  }

  reviewLoadRun += 1;
  const run = reviewLoadRun;
  abortReviewRequests();
  const abortController = new AbortController();
  reviewLoadAbortController = abortController;

  workspaceSessionStore.beginReview(props.workspaceId);
  errorMessage.value = '';
  sendErrorMessage.value = '';
  fileRequestStates.value = {};

  try {
    const data = await createReviewSnapshot(props.workspaceId, { signal: abortController.signal });
    if (reviewLoadRun !== run || abortController.signal.aborted) {
      void deleteReviewSnapshot(data.id);
      return;
    }

    workspaceSessionStore.initialiseReview(props.workspaceId, data, selectInitialScope(data));
    ensureActiveFile();
    await loadActiveFileContents();
  } catch (error) {
    if (reviewLoadRun !== run || abortController.signal.aborted || isAbortError(error)) {
      return;
    }

    errorMessage.value = error instanceof Error ? error.message : 'Could not start review';
    workspaceSessionStore.setReviewState(props.workspaceId, 'error');
    await nextTick();
    startButton.value?.focus();
  } finally {
    if (reviewLoadAbortController === abortController) {
      reviewLoadAbortController = null;
    }
  }
};

const loadActiveFileContents = async () => {
  const file = activeFile.value;
  if (!file) {
    return;
  }

  const key = cacheKey(activeScope.value, file.id);
  const existingState = fileRequestStates.value[key];
  if (existingState?.contents || existingState?.isLoading) {
    return;
  }

  const abortController = new AbortController();
  fileRequestAbortControllers.set(key, abortController);
  setFileRequestState(key, { contents: null, error: '', isLoading: true });

  try {
    const snapshotId = reviewData.value?.id;
    if (!snapshotId) {
      return;
    }

    const contents = await getReviewSnapshotFileContents(
      snapshotId,
      file.id,
      { scope: activeScope.value },
      { signal: abortController.signal }
    );
    if (abortController.signal.aborted || state.value !== 'ready') {
      return;
    }

    setFileRequestState(key, { contents, error: '', isLoading: false });
  } catch (error) {
    if (abortController.signal.aborted || state.value !== 'ready' || isAbortError(error)) {
      return;
    }

    setFileRequestState(key, {
      contents: null,
      error: error instanceof Error ? error.message : 'Could not load file',
      isLoading: false
    });
  } finally {
    if (fileRequestAbortControllers.get(key) === abortController) {
      fileRequestAbortControllers.delete(key);
    }
  }
};

const createCommentId = () => `${Date.now()}:${Math.random().toString(16).slice(2)}`;

const addLineComment = (payload: { side: Exclude<CommentSide, 'file'>; lineNumber: number; endLine?: number }) => {
  const file = activeFile.value;
  if (!file) {
    return;
  }

  comments.value = [
    ...comments.value,
    {
      id: createCommentId(),
      fileId: file.id,
      scope: activeScope.value,
      side: payload.side,
      kind: 'feedback',
      startLine: payload.lineNumber,
      endLine: payload.endLine ?? payload.lineNumber,
      body: ''
    }
  ];
};

const updateCommentBody = (payload: { commentId: string; body: string }) => {
  comments.value = comments.value.map((comment) =>
    comment.id === payload.commentId ? { ...comment, body: payload.body } : comment
  );
};

const toggleCommentKind = (commentId: string) => {
  comments.value = comments.value.map((comment) =>
    comment.id === commentId ? { ...comment, kind: comment.kind === 'feedback' ? 'question' : 'feedback' } : comment
  );
};

const openDraftComment = async (draft: DraftReviewComment) => {
  draftComment.value = draft;
  draftCommentBody.value = '';
  draftCommentKind.value = 'feedback';
  await nextTick();
  draftCommentTextarea.value?.focus();
};

const openFileComment = () => {
  const file = activeFile.value;
  if (!file) {
    return;
  }

  void openDraftComment({
    fileId: file.id,
    filePath: activeFilePath.value,
    scope: activeScope.value,
    side: 'file',
    startLine: null,
    endLine: null
  });
};

const closeDraftComment = () => {
  draftComment.value = null;
  draftCommentBody.value = '';
  draftCommentKind.value = 'feedback';
};

const saveDraftComment = () => {
  const draft = draftComment.value;
  const body = draftCommentBody.value.trim();
  if (!draft || body.length === 0) {
    closeDraftComment();
    return;
  }

  comments.value = [
    ...comments.value,
    {
      id: createCommentId(),
      fileId: draft.fileId,
      scope: draft.scope,
      side: draft.side,
      kind: draftCommentKind.value,
      startLine: draft.startLine,
      endLine: draft.endLine,
      body
    }
  ];
  closeDraftComment();
};

const deleteComment = (commentId: string) => {
  comments.value = comments.value.filter((comment) => comment.id !== commentId);
};

const toggleDraftCommentKind = () => {
  draftCommentKind.value = draftCommentKind.value === 'feedback' ? 'question' : 'feedback';
};

const openOverallNote = async () => {
  overallNoteDraft.value = overallComment.value;
  isOverallNoteOpen.value = true;
  await nextTick();
  overallNoteTextarea.value?.focus();
};

const closeOverallNote = () => {
  isOverallNoteOpen.value = false;
  overallNoteDraft.value = '';
};

const saveOverallNote = () => {
  overallComment.value = overallNoteDraft.value.trim();
  closeOverallNote();
};

const getCommentFilePath = (comment: ReviewComment) => {
  const file = reviewData.value?.files.find((candidate) => candidate.id === comment.fileId) ?? null;
  if (!file) {
    return '(unknown file)';
  }

  return getComparison(file, comment.scope)?.displayPath ?? file.path;
};

const formatCommentLocation = (comment: ReviewComment) => {
  const filePath = getCommentFilePath(comment);
  const prefix = `[${promptScopeLabel(comment.scope)}]`;
  if (comment.side === 'file' || comment.startLine == null) {
    return `${prefix} ${filePath}`;
  }

  const lineRange =
    comment.endLine != null && comment.endLine !== comment.startLine
      ? `${comment.startLine}-${comment.endLine}`
      : `${comment.startLine}`;
  const sideSuffix = comment.scope === 'current' ? '' : comment.side === 'original' ? ' (old)' : ' (new)';

  return `${prefix} ${filePath}:${lineRange}${sideSuffix}`;
};

const appendCommentSection = (lines: string[], title: string, sectionComments: ReviewComment[]) => {
  if (sectionComments.length === 0) {
    return;
  }

  lines.push(title);
  sectionComments.forEach((comment, index) => {
    lines.push(`${index + 1}. ${formatCommentLocation(comment)}`);
    lines.push(`   ${comment.body.trim()}`);
    lines.push('');
  });
};

const composeReviewPrompt = () => {
  const reviewComments = trimmedComments.value;
  const feedbackComments = reviewComments.filter((comment) => comment.kind === 'feedback');
  const questionComments = reviewComments.filter((comment) => comment.kind === 'question');
  const lines: string[] = [];

  lines.push('Please address the following review comments.');
  lines.push('');
  lines.push('Instructions:');
  lines.push(
    '- Feedback/improvement items request code changes. Action them. If an item is unclear, ask a clarifying question before changing code.'
  );
  lines.push(
    '- Question items are reviewer questions. Answer them briefly and clearly. Do not make code changes for question items.'
  );
  lines.push('');

  const trimmedOverallComment = overallComment.value.trim();
  if (trimmedOverallComment.length > 0) {
    lines.push('Reviewer note:');
    lines.push(trimmedOverallComment);
    lines.push('');
  }

  appendCommentSection(lines, 'Feedback/improvements:', feedbackComments);
  appendCommentSection(lines, 'Questions:', questionComments);

  return lines.join('\n').trim();
};

const finishReview = async () => {
  if (!canFinishReview.value) {
    return;
  }

  isSendingPrompt.value = true;
  sendErrorMessage.value = '';

  try {
    await pasteIntoAgentTerminal(props.workspaceId, composeReviewPrompt());
    const snapshotId = reviewData.value?.id;
    resetReview();
    emit('requestTerminalTab');
    if (snapshotId) {
      void deleteReviewSnapshot(snapshotId);
    }
  } catch (error) {
    sendErrorMessage.value = error instanceof Error ? error.message : 'Could not send the review prompt';
  } finally {
    isSendingPrompt.value = false;
  }
};

const focusActiveTerminal = async () => {
  if (!props.isActive) {
    return;
  }

  await nextTick();
  if (draftComment.value) {
    draftCommentTextarea.value?.focus();
    return;
  }

  if (isOverallNoteOpen.value) {
    overallNoteTextarea.value?.focus();
    return;
  }

  startButton.value?.focus();
};

const switchToNextTerminal = async () => {
  emit('requestTerminalTab');
};

onMounted(() => {
  window.addEventListener('keydown', handleReviewKeydown, true);

  if (state.value === 'ready') {
    ensureActiveFile();
    void loadActiveFileContents();
  }

  void focusActiveTerminal();
});

onBeforeUnmount(() => {
  reviewLoadRun += 1;
  abortReviewRequests();
  if (workspaceSessionStore.getReviewState(props.workspaceId) === 'loading') {
    workspaceSessionStore.clearReview(props.workspaceId);
  }
  window.removeEventListener('keydown', handleReviewKeydown, true);
});

watch(activeScope, () => {
  ensureActiveFile();
  void loadActiveFileContents();
});

watch(filteredFiles, (files) => {
  if (state.value !== 'ready' || files.length === 0 || files.some((file) => file.id === activeFileId.value)) {
    return;
  }

  selectFile(files[0]);
});

watch(
  () => props.isActive,
  (isActive) => {
    if (isActive) {
      void focusActiveTerminal();
    }
  }
);

defineExpose({
  cancelReview,
  focusActiveTerminal,
  startReview,
  switchToNextTerminal
});
</script>

<template>
  <section id="review-tab" aria-label="Workspace review">
    <section v-if="state !== 'ready'" class="review-empty-state" aria-label="Review start">
      <div class="review-empty-card">
        <p class="review-kicker">Code review</p>
        <h2>Review this workspace from inside WADE.</h2>
        <p>Start a local review of an open pull request, current working tree, last commit or full working tree.</p>
        <button ref="startButton" type="button" :disabled="state === 'loading'" @click="startReview">
          {{ state === 'loading' ? 'Starting review' : 'Start Review' }}
        </button>
        <p v-if="startErrorMessage" class="review-error" role="alert">{{ startErrorMessage }}</p>
      </div>
    </section>

    <section v-else class="review-workspace" aria-label="Review workspace">
      <aside class="review-sidebar" aria-label="Review files">
        <header class="review-sidebar-header">
          <p class="review-kicker">Review</p>
          <h2>{{ reviewData?.branch?.name || 'git' }}</h2>
          <p :title="workspaceId">{{ workspaceId }}</p>
          <p>{{ reviewedFileCount }} of {{ scopedFiles.length }} reviewed</p>
        </header>

        <section class="review-scopes" aria-label="Review scopes">
          <button
            v-for="scope in scopeOptions"
            :key="scope.id"
            type="button"
            :disabled="scopeCounts[scope.id] === 0"
            :data-active="String(activeScope === scope.id)"
            @click="selectScope(scope.id)"
          >
            <span>{{ scope.label }}</span>
            <span>{{ scopeCounts[scope.id] }}</span>
          </button>
        </section>

        <label class="review-search">
          <span>Filter files</span>
          <input ref="searchInput" v-model="filterText" type="search" placeholder="Search files" spellcheck="false" />
        </label>

        <section class="review-file-list" aria-label="Files">
          <p v-if="!hasReviewableFiles" class="review-muted">No reviewable files found.</p>
          <p v-else-if="filteredFiles.length === 0" class="review-muted">No files match this filter.</p>
          <template v-else-if="normalizedFilterText">
            <button
              v-for="file in filteredFiles"
              :key="file.id"
              class="review-file-row"
              type="button"
              :data-active="String(activeFileId === file.id)"
              :data-reviewed="String(isFileReviewed(file))"
              @click="selectFile(file)"
            >
              <span class="review-file-name">{{ file.path }}</span>
              <span class="review-file-meta">
                <span v-if="isFileReviewed(file)" class="review-reviewed-marker">Reviewed</span>
                <span v-if="commentCountForFile(file) > 0" class="review-comment-count">{{
                  commentCountForFile(file)
                }}</span>
                <span v-if="getFileStatus(file)" class="review-file-status" :data-status="getFileStatus(file)">
                  {{ statusLabel(getFileStatus(file)) }}
                </span>
              </span>
            </button>
          </template>
          <template v-else>
            <button
              v-for="row in fileTreeRows"
              :key="row.id"
              class="review-file-row"
              type="button"
              :data-active="String(row.kind === 'file' && activeFileId === row.file.id)"
              :data-kind="row.kind"
              :data-reviewed="String(row.kind === 'file' && isFileReviewed(row.file))"
              :style="{ paddingLeft: `${row.depth * 14 + 8}px` }"
              @click="row.kind === 'dir' ? toggleDirectoryCollapsed(row.path) : selectFile(row.file)"
            >
              <span class="review-file-name">
                <span v-if="row.kind === 'dir'" class="review-directory-chevron">{{
                  collapsedDirectories[row.path] ? '›' : '⌄'
                }}</span>
                <span>{{ row.name }}</span>
              </span>
              <span v-if="row.kind === 'file'" class="review-file-meta">
                <span v-if="isFileReviewed(row.file)" class="review-reviewed-marker">Reviewed</span>
                <span v-if="commentCountForFile(row.file) > 0" class="review-comment-count">{{
                  commentCountForFile(row.file)
                }}</span>
                <span v-if="getFileStatus(row.file)" class="review-file-status" :data-status="getFileStatus(row.file)">
                  {{ statusLabel(getFileStatus(row.file)) }}
                </span>
              </span>
            </button>
          </template>
        </section>
      </aside>

      <main class="review-main" aria-label="Review file">
        <header class="review-file-header">
          <section>
            <p class="review-kicker">{{ scopeLabel(activeScope) }}</p>
            <h2>{{ activeFilePath || 'No file selected' }}</h2>
            <p>{{ reviewInstructions }}</p>
          </section>
          <section class="review-header-actions" aria-label="Review actions">
            <section class="review-action-group" aria-label="Comment actions">
              <button
                class="review-icon-button"
                type="button"
                title="Overall note"
                aria-label="Overall note"
                @click="openOverallNote"
              >
                <NotebookPen :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <button
                class="review-icon-button"
                type="button"
                :disabled="!activeFile"
                title="Add file comment"
                aria-label="Add file comment"
                @click="openFileComment"
              >
                <MessageSquarePlus :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
            </section>
            <section class="review-action-group" aria-label="Editor view actions">
              <button
                v-if="isActiveFileMarkdown"
                class="review-icon-button"
                type="button"
                :data-active="String(showRenderedMarkdown)"
                :title="markdownViewButtonLabel"
                :aria-label="markdownViewButtonLabel"
                @click="toggleMarkdownView"
              >
                <Code2 v-if="showRenderedMarkdown" :size="15" :stroke-width="1.8" aria-hidden="true" />
                <BookOpen v-else :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <button
                v-if="!showRenderedMarkdown"
                class="review-icon-button"
                type="button"
                :disabled="!activeComparison"
                :data-active="String(hideUnchanged)"
                :title="hideUnchangedButtonLabel"
                :aria-label="hideUnchangedButtonLabel"
                @click="toggleHideUnchanged"
              >
                <ListCollapse v-if="hideUnchanged" :size="15" :stroke-width="1.8" aria-hidden="true" />
                <EyeOff v-else :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <button
                v-if="!showRenderedMarkdown"
                class="review-icon-button"
                type="button"
                :data-active="String(wrapLines)"
                :title="wrapLinesButtonLabel"
                :aria-label="wrapLinesButtonLabel"
                @click="toggleWrapLines"
              >
                <TextWrap :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <button
                v-if="!showRenderedMarkdown"
                class="review-icon-button"
                type="button"
                :disabled="!activeComparison"
                :data-active="String(renderSideBySide)"
                :title="renderSideBySideButtonLabel"
                :aria-label="renderSideBySideButtonLabel"
                @click="toggleRenderSideBySide"
              >
                <Columns2 v-if="renderSideBySide" :size="15" :stroke-width="1.8" aria-hidden="true" />
                <Rows2 v-else :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <button
                class="review-icon-button"
                type="button"
                :disabled="!activeFile"
                :data-active="String(isActiveFileReviewed)"
                :title="isActiveFileReviewed ? 'Mark not reviewed' : 'Mark reviewed'"
                :aria-label="isActiveFileReviewed ? 'Mark not reviewed' : 'Mark reviewed'"
                @click="toggleActiveFileReviewed"
              >
                <Check :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
            </section>
            <section class="review-action-group" aria-label="Review lifecycle actions">
              <button
                class="review-icon-button"
                type="button"
                title="Cancel review"
                aria-label="Cancel review"
                @click="cancelReview"
              >
                <X :size="15" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <button class="review-finish-button" type="button" :disabled="!canFinishReview" @click="finishReview">
                <Send :size="14" :stroke-width="1.8" aria-hidden="true" />
                <span>{{ isSendingPrompt ? 'Sending' : 'Finish' }}</span>
              </button>
            </section>
          </section>
        </header>
        <p
          class="review-file-error"
          :data-visible="String(Boolean(visibleErrorMessage))"
          :role="visibleErrorMessage ? 'alert' : undefined"
          :aria-hidden="!visibleErrorMessage"
        >
          {{ visibleErrorMessage }}
        </p>
        <section
          class="review-comments-panel"
          :data-visible="String(activeFileFileComments.length > 0)"
          aria-label="File comments for selected file"
        >
          <article
            v-for="comment in activeFileFileComments"
            :key="comment.id"
            class="review-comment-card"
            :data-kind="comment.kind"
          >
            <header>
              <span>{{ commentKindLabel(comment.kind) }}</span>
              <span>{{
                comment.side === 'file' ? 'File' : `${commentSideLabel(comment.side)}:${comment.startLine}`
              }}</span>
            </header>
            <p>{{ comment.body }}</p>
            <button type="button" @click="deleteComment(comment.id)">Delete</button>
          </article>
        </section>
        <ReviewMarkdownViewer
          v-if="showRenderedMarkdown"
          :comments="activeFileInlineComments"
          :contents="activeContents"
          :is-loading="isActiveFileLoading"
          @add-line-comment="addLineComment"
          @delete-comment="deleteComment"
          @toggle-comment-kind="toggleCommentKind"
          @update-comment-body="updateCommentBody"
        />
        <ReviewDiffViewer
          v-else
          :comments="activeFileInlineComments"
          :contents="activeContents"
          :file-path="activeFilePath"
          :hide-unchanged="hideUnchanged"
          :is-diff="Boolean(activeComparison)"
          :is-loading="isActiveFileLoading"
          :render-side-by-side="renderSideBySide"
          :scroll-key="reviewScrollKey"
          :wrap-lines="wrapLines"
          @add-line-comment="addLineComment"
          @delete-comment="deleteComment"
          @toggle-comment-kind="toggleCommentKind"
          @update-comment-body="updateCommentBody"
        />
      </main>
    </section>

    <section
      v-if="draftComment"
      class="review-modal-backdrop"
      aria-label="Add review comment"
      @click.self="closeDraftComment"
    >
      <article class="review-modal-card" :data-kind="draftCommentKind">
        <header>
          <p class="review-kicker">{{ commentKindLabel(draftCommentKind) }}</p>
          <h2>{{ draftCommentTitle }}</h2>
          <p>{{ draftCommentDescription }}</p>
        </header>
        <textarea
          ref="draftCommentTextarea"
          v-model="draftCommentBody"
          spellcheck="true"
          placeholder="Write a review comment"
          @keydown.shift.tab.prevent.stop="toggleDraftCommentKind"
        ></textarea>
        <footer>
          <button
            class="review-kind-toggle"
            type="button"
            :data-kind="draftCommentKind"
            @click="toggleDraftCommentKind"
          >
            {{ commentKindLabel(draftCommentKind) }}
          </button>
          <button type="button" @click="closeDraftComment">Cancel</button>
          <button type="button" :disabled="draftCommentBody.trim().length === 0" @click="saveDraftComment">
            Add comment
          </button>
        </footer>
      </article>
    </section>

    <section
      v-if="isOverallNoteOpen"
      class="review-modal-backdrop"
      aria-label="Overall review note"
      @click.self="closeOverallNote"
    >
      <article class="review-modal-card">
        <header>
          <p class="review-kicker">Review note</p>
          <h2>Overall note</h2>
          <p>This note appears above the generated review comments.</p>
        </header>
        <textarea
          ref="overallNoteTextarea"
          v-model="overallNoteDraft"
          spellcheck="true"
          placeholder="Write an overall review note"
        ></textarea>
        <footer>
          <button type="button" @click="closeOverallNote">Cancel</button>
          <button type="button" @click="saveOverallNote">Save note</button>
        </footer>
      </article>
    </section>
  </section>
</template>

<style scoped>
#review-tab {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--window);
  color: var(--text);
}

.review-empty-state {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
  padding: 24px;
}

.review-empty-card {
  width: min(520px, 100%);
  display: grid;
  gap: 14px;
  padding: 28px;
  border: 1px solid var(--text);
  border-radius: 0;
  background: var(--window);
  text-align: center;
}

.review-kicker {
  margin: 0;
  color: var(--muted);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.review-empty-card h2,
.review-sidebar-header h2,
.review-file-header h2,
.review-modal-card h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  line-height: 1.2;
}

.review-empty-card p:not(.review-kicker):not(.review-error) {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}

button {
  border: 1px solid var(--text);
  background: transparent;
  color: var(--text);
  font: inherit;
  cursor: pointer;
}

button:disabled {
  color: var(--muted);
  cursor: not-allowed;
  opacity: 0.55;
}

.review-empty-card button {
  justify-self: center;
  min-width: 148px;
  height: 38px;
  border-radius: 0;
}

button:not(:disabled):hover,
button:not(:disabled):focus-visible {
  background: rgb(var(--accent-rgb) / 10%);
}

.review-error,
.review-file-error {
  margin: 0;
  color: #ff6e6e;
  font-size: 12px;
  line-height: 1.4;
}

.review-workspace {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  overflow: hidden;
}

.review-sidebar {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto auto minmax(0, 1fr);
  border-right: 1px solid var(--text);
  overflow: hidden;
}

.review-sidebar-header {
  display: grid;
  gap: 6px;
  padding: 16px;
  border-bottom: 1px solid var(--text);
}

.review-sidebar-header p:not(.review-kicker) {
  margin: 0;
  overflow: hidden;
  color: var(--muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-scopes {
  display: grid;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--text);
}

.review-scopes button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  height: 32px;
  padding: 0 10px;
  border-radius: 0;
  font-size: 12px;
}

.review-scopes button[data-active='true'] {
  border-color: var(--text);
  background: rgb(var(--accent-rgb) / 10%);
  color: var(--text);
}

.review-search {
  display: grid;
  gap: 7px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--text);
  color: var(--muted);
  font-size: 11px;
}

.review-search input,
.review-modal-card textarea {
  width: 100%;
  border: 1px solid var(--text);
  border-radius: 0;
  outline: none;
  background: rgb(0 0 0 / 18%);
  color: var(--text);
  font: inherit;
  font-size: 12px;
}

.review-search input {
  height: 34px;
  padding: 0 10px;
}

.review-search input:focus,
.review-modal-card textarea:focus {
  border-color: var(--text);
}

.review-file-list {
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.review-file-row {
  width: 100%;
  min-height: 34px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border: 0;
  border-radius: 0;
  text-align: left;
}

.review-file-row[data-active='true'] {
  background: rgb(var(--accent-rgb) / 14%);
}

.review-file-row[data-kind='dir'] {
  color: var(--muted);
  font-weight: 500;
}

.review-file-row[data-reviewed='true'] .review-file-name {
  color: var(--muted);
}

.review-file-name {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-file-name > span:last-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-directory-chevron {
  width: 10px;
  flex: 0 0 auto;
  color: var(--muted);
}

.review-file-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.review-reviewed-marker,
.review-comment-count {
  min-width: 16px;
  padding: 1px 4px;
  border: 1px solid var(--text);
  color: var(--text);
  font-size: 10px;
  text-align: center;
}

.review-reviewed-marker {
  border-color: #50fa7b;
  color: #50fa7b;
}

.review-file-status {
  color: #8be9fd;
  font-size: 10px;
}

.review-file-status[data-status='added'] {
  color: #69ff94;
}

.review-file-status[data-status='deleted'] {
  color: #ff6e6e;
}

.review-file-status[data-status='renamed'] {
  color: #f1fa8c;
}

.review-muted {
  margin: 8px;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.5;
}

.review-main {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto auto minmax(0, 1fr);
  overflow: hidden;
}

.review-file-header {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--text);
}

.review-file-header > section:first-child {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.review-file-header h2 {
  overflow: hidden;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-file-header p:not(.review-kicker) {
  margin: 0;
  color: var(--muted);
  font-size: 11px;
}

.review-header-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 10px;
}

.review-action-group {
  display: flex;
  align-items: center;
  gap: 4px;
}

.review-action-group + .review-action-group {
  padding-left: 10px;
  border-left: 1px solid rgb(var(--accent-rgb) / 35%);
}

.review-header-actions button {
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 0;
  font-size: 12px;
}

.review-icon-button {
  width: 30px;
  padding: 0;
}

.review-finish-button {
  padding: 0 11px;
}

.review-header-actions button[data-active='true'] {
  border-color: #50fa7b;
  background: rgb(80 250 123 / 10%);
  color: #50fa7b;
}

.review-file-error {
  padding: 10px 16px;
  border-bottom: 1px solid rgb(255 85 85 / 30%);
  background: rgb(255 85 85 / 8%);
}

.review-file-error[data-visible='false'] {
  height: 0;
  padding-block: 0;
  border-bottom: 0;
  overflow: hidden;
}

.review-comments-panel {
  max-height: 190px;
  display: grid;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--text);
  overflow: auto;
}

.review-comments-panel[data-visible='false'] {
  height: 0;
  padding-block: 0;
  border-bottom: 0;
  overflow: hidden;
}

.review-comment-card {
  display: grid;
  gap: 6px;
  padding: 8px;
  border: 1px solid rgb(var(--accent-rgb) / 45%);
}

.review-comment-card header,
.review-comment-card footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.review-comment-card header {
  color: var(--muted);
  font-size: 11px;
  text-transform: uppercase;
}

.review-comment-card p {
  margin: 0;
  color: var(--text);
  font-size: 12px;
  line-height: 1.45;
  white-space: pre-wrap;
}

.review-comment-card button {
  justify-self: end;
  height: 24px;
  padding: 0 8px;
  font-size: 11px;
}

.review-modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgb(23 24 28 / 82%);
}

.review-modal-card {
  width: min(720px, 100%);
  min-width: 0;
  display: grid;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--text);
  background: var(--window);
  overflow: hidden;
}

.review-modal-card header {
  min-width: 0;
  display: grid;
  gap: 6px;
  overflow: hidden;
}

.review-modal-card h2 {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-modal-card header p:not(.review-kicker) {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
}

.review-modal-card textarea {
  min-height: 160px;
  resize: vertical;
  padding: 10px;
  line-height: 1.45;
}

.review-modal-card footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.review-modal-card footer button {
  height: 32px;
  padding: 0 10px;
  font-size: 12px;
}

.review-kind-toggle[data-kind='question'] {
  border-color: #d29922;
  background: rgb(210 153 34 / 14%);
  color: #d29922;
}

.review-kind-toggle[data-kind='feedback'] {
  border-color: #ff6e6e;
  background: rgb(255 110 110 / 14%);
  color: #ff6e6e;
}
</style>
