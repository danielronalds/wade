<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { pasteIntoAgentTerminal } from '../composables/useAgentTerminalInput';
import type {
  CommentSide,
  ReviewComment,
  ReviewCommentKind,
  ReviewData,
  ReviewFile,
  ReviewFileComparison,
  ReviewFileContents,
  ReviewScope
} from '../types/review';
import ReviewDiffViewer from './review/ReviewDiffViewer.vue';

const props = defineProps<{
  projectName: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  requestTerminalTab: [];
}>();

type ReviewState = 'idle' | 'loading' | 'ready' | 'error';

type FileRequestState = {
  contents: ReviewFileContents | null;
  error: string;
  isLoading: boolean;
};

type DraftComment = {
  fileId: string;
  filePath: string;
  scope: ReviewScope;
  side: CommentSide;
  startLine: number | null;
  endLine: number | null;
};

const state = ref<ReviewState>('idle');
const reviewData = ref<ReviewData | null>(null);
const activeScope = ref<ReviewScope>('git-diff');
const activeFileId = ref<string | null>(null);
const filterText = ref('');
const errorMessage = ref('');
const sendErrorMessage = ref('');
const fileRequestStates = ref<Record<string, FileRequestState>>({});
const comments = ref<ReviewComment[]>([]);
const overallComment = ref('');
const draftComment = ref<DraftComment | null>(null);
const draftCommentBody = ref('');
const draftCommentKind = ref<ReviewCommentKind>('feedback');
const isOverallNoteOpen = ref(false);
const overallNoteDraft = ref('');
const isSendingPrompt = ref(false);
const startButton = ref<HTMLButtonElement | null>(null);
const draftCommentTextarea = ref<HTMLTextAreaElement | null>(null);
const overallNoteTextarea = ref<HTMLTextAreaElement | null>(null);

const scopeOptions: Array<{ id: ReviewScope; label: string }> = [
  { id: 'git-diff', label: 'Git diff' },
  { id: 'last-commit', label: 'Last commit' },
  { id: 'all-files', label: 'All files' }
];

const cacheKey = (scope: ReviewScope, fileId: string) => `${scope}:${fileId}`;

const scopeLabel = (scope: ReviewScope) => {
  switch (scope) {
    case 'git-diff': return 'Git diff';
    case 'last-commit': return 'Last commit';
    default: return 'All files';
  }
};

const promptScopeLabel = (scope: ReviewScope) => scopeLabel(scope).toLowerCase();

const commentKindLabel = (kind: ReviewCommentKind) => kind === 'question' ? 'Question' : 'Feedback';

const commentSideLabel = (side: CommentSide) => {
  if (side === 'original') return 'original';
  if (side === 'modified') return 'modified';
  return 'whole file';
};

const getComparison = (file: ReviewFile | null, scope: ReviewScope): ReviewFileComparison | null => {
  if (!file) {
    return null;
  }

  if (scope === 'git-diff') {
    return file.gitDiff;
  }

  if (scope === 'last-commit') {
    return file.lastCommit;
  }

  return null;
};

const scopeCounts = computed<Record<ReviewScope, number>>(() => ({
  'git-diff': reviewData.value?.files.filter((file) => file.inGitDiff).length ?? 0,
  'last-commit': reviewData.value?.files.filter((file) => file.inLastCommit).length ?? 0,
  'all-files': reviewData.value?.files.filter((file) => file.hasWorkingTreeFile).length ?? 0
}));

const scopedFiles = computed(() => {
  const files = reviewData.value?.files ?? [];
  if (activeScope.value === 'git-diff') {
    return files.filter((file) => file.inGitDiff);
  }

  if (activeScope.value === 'last-commit') {
    return files.filter((file) => file.inLastCommit);
  }

  return files.filter((file) => file.hasWorkingTreeFile);
});

const normalizedFilterText = computed(() => filterText.value.trim().toLowerCase());
const filteredFiles = computed(() => {
  if (!normalizedFilterText.value) {
    return scopedFiles.value;
  }

  return scopedFiles.value.filter((file) => file.path.toLowerCase().includes(normalizedFilterText.value));
});

const activeFile = computed(() => reviewData.value?.files.find((file) => file.id === activeFileId.value) ?? null);
const activeComparison = computed(() => getComparison(activeFile.value, activeScope.value));
const activeFilePath = computed(() => activeComparison.value?.displayPath ?? activeFile.value?.path ?? '');
const activeCacheKey = computed(() => activeFile.value ? cacheKey(activeScope.value, activeFile.value.id) : '');
const activeFileRequestState = computed(() => activeCacheKey.value ? fileRequestStates.value[activeCacheKey.value] : undefined);
const activeContents = computed(() => activeFileRequestState.value?.contents ?? null);
const isActiveFileLoading = computed(() => activeFileRequestState.value?.isLoading === true);
const activeFileError = computed(() => activeFileRequestState.value?.error ?? '');
const visibleErrorMessage = computed(() => sendErrorMessage.value || activeFileError.value);
const canStartReview = computed(() => state.value === 'idle' || state.value === 'error');
const hasReviewableFiles = computed(() => (reviewData.value?.files.length ?? 0) > 0);
const activeFileComments = computed(() => comments.value.filter((comment) => comment.fileId === activeFileId.value && comment.scope === activeScope.value));
const activeFileInlineComments = computed(() => activeFileComments.value.filter((comment) => comment.side !== 'file'));
const activeFileFileComments = computed(() => activeFileComments.value.filter((comment) => comment.side === 'file'));
const trimmedComments = computed(() => comments.value
  .map((comment) => ({ ...comment, body: comment.body.trim() }))
  .filter((comment) => comment.body.length > 0));
const canFinishReview = computed(() => !isSendingPrompt.value && (trimmedComments.value.length > 0 || overallComment.value.trim().length > 0));
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

const commentCountForFile = (file: ReviewFile) => comments.value.filter((comment) => comment.fileId === file.id && comment.scope === activeScope.value && comment.body.trim().length > 0).length;

const selectInitialScope = (data: ReviewData): ReviewScope => {
  if (data.files.some((file) => file.inGitDiff)) {
    return 'git-diff';
  }

  if (data.files.some((file) => file.inLastCommit)) {
    return 'last-commit';
  }

  return 'all-files';
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

const resetReview = async () => {
  state.value = 'idle';
  reviewData.value = null;
  activeScope.value = 'git-diff';
  activeFileId.value = null;
  filterText.value = '';
  errorMessage.value = '';
  sendErrorMessage.value = '';
  fileRequestStates.value = {};
  comments.value = [];
  overallComment.value = '';
  draftComment.value = null;
  isOverallNoteOpen.value = false;
  await nextTick();
  startButton.value?.focus();
};

const setFileRequestState = (key: string, value: FileRequestState) => {
  fileRequestStates.value = {
    ...fileRequestStates.value,
    [key]: value
  };
};

const requestJSON = async <T>(url: string, options?: RequestInit): Promise<T> => {
  const response = await fetch(url, options);
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { message?: string } | null;
    throw new Error(body?.message || 'Review request failed');
  }

  return response.json() as Promise<T>;
};

const startReview = async () => {
  if (!canStartReview.value) {
    return;
  }

  state.value = 'loading';
  errorMessage.value = '';
  sendErrorMessage.value = '';
  fileRequestStates.value = {};
  comments.value = [];
  overallComment.value = '';

  try {
    const params = new URLSearchParams({ project: props.projectName });
    const data = await requestJSON<ReviewData>(`/api/review?${params}`);
    reviewData.value = data;
    activeScope.value = selectInitialScope(data);
    state.value = 'ready';
    ensureActiveFile();
    await loadActiveFileContents();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Could not start review';
    state.value = 'error';
    await nextTick();
    startButton.value?.focus();
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

  setFileRequestState(key, { contents: null, error: '', isLoading: true });

  try {
    const params = new URLSearchParams({ project: props.projectName });
    const contents = await requestJSON<ReviewFileContents>(`/api/review/file?${params}`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ fileId: file.id, scope: activeScope.value })
    });
    setFileRequestState(key, { contents, error: '', isLoading: false });
  } catch (error) {
    setFileRequestState(key, {
      contents: null,
      error: error instanceof Error ? error.message : 'Could not load file',
      isLoading: false
    });
  }
};

const createCommentId = () => `${Date.now()}:${Math.random().toString(16).slice(2)}`;

const addLineComment = (payload: { side: Exclude<CommentSide, 'file'>; lineNumber: number }) => {
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
      endLine: payload.lineNumber,
      body: ''
    }
  ];
};

const updateCommentBody = (payload: { commentId: string; body: string }) => {
  comments.value = comments.value.map((comment) => comment.id === payload.commentId
    ? { ...comment, body: payload.body }
    : comment);
};

const toggleCommentKind = (commentId: string) => {
  comments.value = comments.value.map((comment) => comment.id === commentId
    ? { ...comment, kind: comment.kind === 'feedback' ? 'question' : 'feedback' }
    : comment);
};

const openDraftComment = async (draft: DraftComment) => {
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

  const lineRange = comment.endLine != null && comment.endLine !== comment.startLine
    ? `${comment.startLine}-${comment.endLine}`
    : `${comment.startLine}`;
  const sideSuffix = comment.scope === 'all-files'
    ? ''
    : comment.side === 'original' ? ' (old)' : ' (new)';

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
  lines.push('- Feedback/improvement items request code changes. Action them. If an item is unclear, ask a clarifying question before changing code.');
  lines.push('- Question items are reviewer questions. Answer them briefly and clearly. Do not make code changes for question items.');
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
    await pasteIntoAgentTerminal(props.projectName, composeReviewPrompt());
    emit('requestTerminalTab');
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
  startButton.value?.focus();
};

const switchToNextTerminal = async () => {
  emit('requestTerminalTab');
};

watch(activeScope, () => {
  ensureActiveFile();
  void loadActiveFileContents();
});

watch(() => props.isActive, (isActive) => {
  if (isActive) {
    void focusActiveTerminal();
  }
});

defineExpose({
  focusActiveTerminal,
  switchToNextTerminal
});
</script>

<template>
  <section id="review-tab" aria-label="Project review">
    <section v-if="state !== 'ready'" class="review-empty-state" aria-label="Review start">
      <div class="review-empty-card">
        <p class="review-kicker">Code review</p>
        <h2>Review this project from inside WADE.</h2>
        <p>
          Start a local review of the current git diff, last commit or full working tree.
        </p>
        <button ref="startButton" type="button" :disabled="state === 'loading'" @click="startReview">
          {{ state === 'loading' ? 'Starting review' : 'Start Review' }}
        </button>
        <p v-if="errorMessage" class="review-error" role="alert">{{ errorMessage }}</p>
      </div>
    </section>

    <section v-else class="review-workspace" aria-label="Review workspace">
      <aside class="review-sidebar" aria-label="Review files">
        <header class="review-sidebar-header">
          <p class="review-kicker">Review</p>
          <h2>{{ reviewData?.branchName || 'git' }}</h2>
          <p :title="reviewData?.repoRoot">{{ reviewData?.repoRoot }}</p>
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
          <input v-model="filterText" type="search" placeholder="Search files" spellcheck="false">
        </label>

        <section class="review-file-list" aria-label="Files">
          <p v-if="!hasReviewableFiles" class="review-muted">No reviewable files found.</p>
          <p v-else-if="filteredFiles.length === 0" class="review-muted">No files match this filter.</p>
          <button
            v-for="file in filteredFiles"
            :key="file.id"
            class="review-file-row"
            type="button"
            :data-active="String(activeFileId === file.id)"
            @click="selectFile(file)"
          >
            <span class="review-file-name">{{ file.path }}</span>
            <span class="review-file-meta">
              <span v-if="commentCountForFile(file) > 0" class="review-comment-count">{{ commentCountForFile(file) }}</span>
              <span v-if="getFileStatus(file)" class="review-file-status" :data-status="getFileStatus(file)">
                {{ statusLabel(getFileStatus(file)) }}
              </span>
            </span>
          </button>
        </section>
      </aside>

      <main class="review-main" aria-label="Review file">
        <header class="review-file-header">
          <section>
            <p class="review-kicker">{{ scopeLabel(activeScope) }}</p>
            <h2>{{ activeFilePath || 'No file selected' }}</h2>
            <p>Click a line number in the diff to add an inline comment.</p>
          </section>
          <section class="review-header-actions" aria-label="Review actions">
            <button type="button" @click="openOverallNote">Overall note</button>
            <button type="button" :disabled="!activeFile" @click="openFileComment">Add file comment</button>
            <button type="button" @click="resetReview">Cancel review</button>
            <button type="button" :disabled="!canFinishReview" @click="finishReview">
              {{ isSendingPrompt ? 'Sending review' : 'Finish review' }}
            </button>
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
        <section class="review-comments-panel" :data-visible="String(activeFileFileComments.length > 0)" aria-label="File comments for selected file">
          <article v-for="comment in activeFileFileComments" :key="comment.id" class="review-comment-card" :data-kind="comment.kind">
            <header>
              <span>{{ commentKindLabel(comment.kind) }}</span>
              <span>{{ comment.side === 'file' ? 'File' : `${commentSideLabel(comment.side)}:${comment.startLine}` }}</span>
            </header>
            <p>{{ comment.body }}</p>
            <button type="button" @click="deleteComment(comment.id)">Delete</button>
          </article>
        </section>
        <ReviewDiffViewer
          :comments="activeFileInlineComments"
          :contents="activeContents"
          :file-path="activeFilePath"
          :is-diff="Boolean(activeComparison)"
          :is-loading="isActiveFileLoading"
          @add-line-comment="addLineComment"
          @delete-comment="deleteComment"
          @toggle-comment-kind="toggleCommentKind"
          @update-comment-body="updateCommentBody"
        />
      </main>
    </section>

    <section v-if="draftComment" class="review-modal-backdrop" aria-label="Add review comment" @click.self="closeDraftComment">
      <article class="review-modal-card" :data-kind="draftCommentKind">
        <header>
          <p class="review-kicker">{{ commentKindLabel(draftCommentKind) }}</p>
          <h2>{{ draftCommentTitle }}</h2>
          <p>{{ draftCommentDescription }}</p>
        </header>
        <textarea ref="draftCommentTextarea" v-model="draftCommentBody" spellcheck="true" placeholder="Write a review comment"></textarea>
        <footer>
          <button class="review-kind-toggle" type="button" :data-kind="draftCommentKind" @click="toggleDraftCommentKind">{{ commentKindLabel(draftCommentKind) }}</button>
          <button type="button" @click="closeDraftComment">Cancel</button>
          <button type="button" :disabled="draftCommentBody.trim().length === 0" @click="saveDraftComment">Add comment</button>
        </footer>
      </article>
    </section>

    <section v-if="isOverallNoteOpen" class="review-modal-backdrop" aria-label="Overall review note" @click.self="closeOverallNote">
      <article class="review-modal-card">
        <header>
          <p class="review-kicker">Review note</p>
          <h2>Overall note</h2>
          <p>This note appears above the generated review comments.</p>
        </header>
        <textarea ref="overallNoteTextarea" v-model="overallNoteDraft" spellcheck="true" placeholder="Write an overall review note"></textarea>
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
  background: rgb(248 248 242 / 10%);
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

.review-sidebar-header p:last-child {
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

.review-scopes button[data-active="true"] {
  border-color: var(--text);
  background: rgb(248 248 242 / 10%);
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

.review-file-row[data-active="true"] {
  background: rgb(248 248 242 / 14%);
}

.review-file-name {
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-file-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.review-comment-count {
  min-width: 16px;
  padding: 1px 4px;
  border: 1px solid var(--text);
  color: var(--text);
  font-size: 10px;
  text-align: center;
}

.review-file-status {
  color: #8be9fd;
  font-size: 10px;
}

.review-file-status[data-status="added"] {
  color: #69ff94;
}

.review-file-status[data-status="deleted"] {
  color: #ff6e6e;
}

.review-file-status[data-status="renamed"] {
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

.review-file-header section:first-child {
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
  gap: 8px;
}

.review-header-actions button {
  height: 30px;
  padding: 0 11px;
  border-radius: 0;
  font-size: 12px;
}

.review-file-error {
  padding: 10px 16px;
  border-bottom: 1px solid rgb(255 85 85 / 30%);
  background: rgb(255 85 85 / 8%);
}

.review-file-error[data-visible="false"] {
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

.review-comments-panel[data-visible="false"] {
  height: 0;
  padding-block: 0;
  border-bottom: 0;
  overflow: hidden;
}

.review-comment-card {
  display: grid;
  gap: 6px;
  padding: 8px;
  border: 1px solid rgb(248 248 242 / 45%);
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
  display: grid;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--text);
  background: var(--window);
}

.review-modal-card header {
  display: grid;
  gap: 6px;
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

.review-kind-toggle[data-kind="question"] {
  border-color: #d29922;
  background: rgb(210 153 34 / 14%);
  color: #d29922;
}

.review-kind-toggle[data-kind="feedback"] {
  border-color: #ff6e6e;
  background: rgb(255 110 110 / 14%);
  color: #ff6e6e;
}
</style>
