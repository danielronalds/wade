<!-- NOTE: Vibecoded and not suppppppper reviewed -->
<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import ReviewDiffViewer from './review/ReviewDiffViewer.vue';
import type { ReviewData, ReviewFile, ReviewFileComparison, ReviewFileContents, ReviewScope } from '../types/review';

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

const state = ref<ReviewState>('idle');
const reviewData = ref<ReviewData | null>(null);
const activeScope = ref<ReviewScope>('git-diff');
const activeFileId = ref<string | null>(null);
const filterText = ref('');
const errorMessage = ref('');
const fileRequestStates = ref<Record<string, FileRequestState>>({});
const startButton = ref<HTMLButtonElement | null>(null);

const scopeOptions: Array<{ id: ReviewScope; label: string }> = [
  { id: 'git-diff', label: 'Git diff' },
  { id: 'last-commit', label: 'Last commit' },
  { id: 'all-files', label: 'All files' }
];

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
const canStartReview = computed(() => state.value === 'idle' || state.value === 'error');
const hasReviewableFiles = computed(() => (reviewData.value?.files.length ?? 0) > 0);

const cacheKey = (scope: ReviewScope, fileId: string) => `${scope}:${fileId}`;

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

const statusLabel = (status: string | null) => {
  if (!status) {
    return '';
  }

  return status.charAt(0).toUpperCase() + status.slice(1);
};

const getFileStatus = (file: ReviewFile) => getComparison(file, activeScope.value)?.status ?? file.worktreeStatus;

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
  void loadActiveFileContents();
};

const cancelReview = async () => {
  state.value = 'idle';
  reviewData.value = null;
  activeScope.value = 'git-diff';
  activeFileId.value = null;
  filterText.value = '';
  errorMessage.value = '';
  fileRequestStates.value = {};
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
  fileRequestStates.value = {};

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
            <span v-if="getFileStatus(file)" class="review-file-status" :data-status="getFileStatus(file)">
              {{ statusLabel(getFileStatus(file)) }}
            </span>
          </button>
        </section>
      </aside>

      <main class="review-main" aria-label="Review file">
        <header class="review-file-header">
          <section>
            <p class="review-kicker">{{ activeScope.replace('-', ' ') }}</p>
            <h2>{{ activeFilePath || 'No file selected' }}</h2>
          </section>
          <section class="review-header-actions" aria-label="Review actions">
            <button type="button" @click="cancelReview">Cancel review</button>
            <button type="button" disabled title="Commenting is next">Finish review</button>
          </section>
        </header>
        <p
          class="review-file-error"
          :data-visible="String(Boolean(activeFileError))"
          :role="activeFileError ? 'alert' : undefined"
          :aria-hidden="!activeFileError"
        >
          {{ activeFileError }}
        </p>
        <ReviewDiffViewer
          :contents="activeContents"
          :file-path="activeFilePath"
          :is-diff="Boolean(activeComparison)"
          :is-loading="isActiveFileLoading"
        />
      </main>
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
.review-file-header h2 {
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

.review-search input {
  width: 100%;
  height: 34px;
  padding: 0 10px;
  border: 1px solid var(--text);
  border-radius: 0;
  outline: none;
  background: rgb(0 0 0 / 18%);
  color: var(--text);
  font: inherit;
  font-size: 12px;
}

.review-search input:focus {
  border-color: rgb(139 233 253 / 70%);
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
  grid-template-rows: auto auto minmax(0, 1fr);
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
</style>
