import { watchDebounced } from '@vueuse/core';
import { defineStore } from 'pinia';
import { computed, reactive, ref, shallowReactive, type Ref } from 'vue';
import {
  deleteReviewSnapshot,
  getReviewSnapshot,
  listWorkspaceTerminals,
  TerminalStatus
} from '@/api/generated/wade';
import type {
  DraftReviewComment,
  ReviewCheckpoint,
  ReviewComment,
  ReviewCommentKind,
  ReviewData,
  ReviewScope,
  ReviewState
} from '@/types/review';
import { TerminalPanes, terminalPanes, type TerminalPaneId } from '@/types/terminalPanes';
import { WorkspaceTabs, workspaceTabs, type WorkspaceTab } from '@/types/workspaceTabs';

export type WorkspaceSessionCheckpoint = {
  activeTab: WorkspaceTab;
  isScratchpadOpen: boolean;
  terminal: {
    activePane: TerminalPaneId;
    zoomedPane: TerminalPaneId | null;
    selectedAgentName: string;
  };
  review: ReviewCheckpoint | null;
};

type WorkspaceSessionEntry = {
  state: WorkspaceSessionCheckpoint;
  isInitialised: boolean;
  lastSerialisedCheckpoint: string;
  reviewData: Ref<ReviewData | null>;
  reviewState: Ref<ReviewState>;
};

export const useWorkspaceSessionStore = defineStore('workspace-session', () => {
  const workspaceSessions = shallowReactive(new Map<string, WorkspaceSessionEntry>());
  const preparationRequests = new Map<string, Promise<void>>();
  const activeWorkspaceId = ref('');
  const defaultReview = createFreshReviewCheckpoint('');

  const activeTab = computed<WorkspaceTab>({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.activeTab ?? WorkspaceTabs.Terminal,
    set: (activeTab) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.activeTab = activeTab;
      }
    }
  });
  const isScratchpadOpen = computed({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.isScratchpadOpen ?? false,
    set: (isScratchpadOpen) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.isScratchpadOpen = isScratchpadOpen;
      }
    }
  });
  const terminalActivePane = computed<TerminalPaneId>({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.terminal.activePane ?? TerminalPanes.Agent,
    set: (activePane) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.terminal.activePane = activePane;
      }
    }
  });
  const terminalZoomedPane = computed<TerminalPaneId | null>({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.terminal.zoomedPane ?? null,
    set: (zoomedPane) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.terminal.zoomedPane = zoomedPane;
      }
    }
  });
  const selectedAgentName = computed({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.terminal.selectedAgentName ?? '',
    set: (selectedAgentName) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.terminal.selectedAgentName = selectedAgentName;
      }
    }
  });
  const reviewState = computed(() => (
    workspaceSessions.get(activeWorkspaceId.value)?.reviewState.value ?? 'idle'
  ));
  const reviewData = computed(() => (
    workspaceSessions.get(activeWorkspaceId.value)?.reviewData.value ?? null
  ));
  const reviewActiveScope = computed<ReviewScope>({
    get: () => getActiveReview()?.activeScope ?? defaultReview.activeScope,
    set: (activeScope) => {
      const review = getActiveReview();
      if (review) {
        review.activeScope = activeScope;
      }
    }
  });
  const reviewActiveFileId = computed<string | null>({
    get: () => getActiveReview()?.activeFileId ?? defaultReview.activeFileId,
    set: (activeFileId) => {
      const review = getActiveReview();
      if (review) {
        review.activeFileId = activeFileId;
      }
    }
  });
  const reviewFilterText = computed({
    get: () => getActiveReview()?.filterText ?? defaultReview.filterText,
    set: (filterText) => {
      const review = getActiveReview();
      if (review) {
        review.filterText = filterText;
      }
    }
  });
  const reviewReviewedFiles = computed<Record<string, boolean>>({
    get: () => getActiveReview()?.reviewedFiles ?? defaultReview.reviewedFiles,
    set: (reviewedFiles) => {
      const review = getActiveReview();
      if (review) {
        review.reviewedFiles = reviewedFiles;
      }
    }
  });
  const reviewCollapsedDirectories = computed<Record<string, boolean>>({
    get: () => getActiveReview()?.collapsedDirectories ?? defaultReview.collapsedDirectories,
    set: (collapsedDirectories) => {
      const review = getActiveReview();
      if (review) {
        review.collapsedDirectories = collapsedDirectories;
      }
    }
  });
  const reviewComments = computed<ReviewComment[]>({
    get: () => getActiveReview()?.comments ?? defaultReview.comments,
    set: (comments) => {
      const review = getActiveReview();
      if (review) {
        review.comments = comments;
      }
    }
  });
  const reviewOverallComment = computed({
    get: () => getActiveReview()?.overallComment ?? defaultReview.overallComment,
    set: (overallComment) => {
      const review = getActiveReview();
      if (review) {
        review.overallComment = overallComment;
      }
    }
  });
  const reviewDraftComment = computed<DraftReviewComment | null>({
    get: () => getActiveReview()?.draftComment ?? defaultReview.draftComment,
    set: (draftComment) => {
      const review = getActiveReview();
      if (review) {
        review.draftComment = draftComment;
      }
    }
  });
  const reviewDraftCommentBody = computed({
    get: () => getActiveReview()?.draftCommentBody ?? defaultReview.draftCommentBody,
    set: (draftCommentBody) => {
      const review = getActiveReview();
      if (review) {
        review.draftCommentBody = draftCommentBody;
      }
    }
  });
  const reviewDraftCommentKind = computed<ReviewCommentKind>({
    get: () => getActiveReview()?.draftCommentKind ?? defaultReview.draftCommentKind,
    set: (draftCommentKind) => {
      const review = getActiveReview();
      if (review) {
        review.draftCommentKind = draftCommentKind;
      }
    }
  });
  const isReviewOverallNoteOpen = computed({
    get: () => getActiveReview()?.isOverallNoteOpen ?? defaultReview.isOverallNoteOpen,
    set: (isOverallNoteOpen) => {
      const review = getActiveReview();
      if (review) {
        review.isOverallNoteOpen = isOverallNoteOpen;
      }
    }
  });
  const reviewOverallNoteDraft = computed({
    get: () => getActiveReview()?.overallNoteDraft ?? defaultReview.overallNoteDraft,
    set: (overallNoteDraft) => {
      const review = getActiveReview();
      if (review) {
        review.overallNoteDraft = overallNoteDraft;
      }
    }
  });
  const reviewHideUnchanged = computed({
    get: () => getActiveReview()?.hideUnchanged ?? defaultReview.hideUnchanged,
    set: (hideUnchanged) => {
      const review = getActiveReview();
      if (review) {
        review.hideUnchanged = hideUnchanged;
      }
    }
  });
  const reviewRenderSideBySide = computed({
    get: () => getActiveReview()?.renderSideBySide ?? defaultReview.renderSideBySide,
    set: (renderSideBySide) => {
      const review = getActiveReview();
      if (review) {
        review.renderSideBySide = renderSideBySide;
      }
    }
  });
  const reviewWrapLines = computed({
    get: () => getActiveReview()?.wrapLines ?? defaultReview.wrapLines,
    set: (wrapLines) => {
      const review = getActiveReview();
      if (review) {
        review.wrapLines = wrapLines;
      }
    }
  });

  const activateWorkspaceSession = (workspaceId: string) => {
    ensureWorkspaceSessionEntry(workspaceId);
    activeWorkspaceId.value = workspaceId;
  };

  const beginReview = (workspaceId: string) => {
    const entry = ensureWorkspaceSessionEntry(workspaceId);
    entry.state.review = null;
    entry.reviewData.value = null;
    entry.reviewState.value = 'loading';
  };

  const initialiseReview = (
    workspaceId: string,
    data: ReviewData,
    activeScope: ReviewScope
  ) => {
    const entry = ensureWorkspaceSessionEntry(workspaceId);
    entry.state.review = createFreshReviewCheckpoint(data.id, activeScope);
    entry.reviewData.value = data;
    entry.reviewState.value = 'ready';
  };

  const setReviewState = (workspaceId: string, state: ReviewState) => {
    ensureWorkspaceSessionEntry(workspaceId).reviewState.value = state;
  };

  const clearReview = (workspaceId: string) => {
    const entry = ensureWorkspaceSessionEntry(workspaceId);
    entry.state.review = null;
    entry.reviewData.value = null;
    entry.reviewState.value = 'idle';
  };

  const clearWorkspaceSession = (workspaceId: string) => {
    const entry = ensureWorkspaceSessionEntry(workspaceId);
    const storedState = parseCheckpoint(getStoredValue(workspaceSessionStorageKey(workspaceId)));
    const snapshotIds = new Set([
      entry.state.review?.snapshotId,
      storedState?.review?.snapshotId
    ].filter((snapshotId): snapshotId is string => Boolean(snapshotId)));
    const freshState = createFreshWorkspaceSession();

    entry.isInitialised = false;
    replaceWorkspaceSession(entry.state, freshState);
    entry.reviewData.value = null;
    entry.reviewState.value = 'idle';
    removeStoredValue(workspaceSessionStorageKey(workspaceId));
    entry.lastSerialisedCheckpoint = serialiseCheckpoint(entry.state);
    entry.isInitialised = true;

    snapshotIds.forEach((snapshotId) => {
      void deleteReviewSnapshot(snapshotId).catch(() => undefined);
    });
  };

  const prepareWorkspaceSession = (workspaceId: string) => {
    const existingRequest = preparationRequests.get(workspaceId);
    if (existingRequest) {
      return existingRequest;
    }

    const preparationRequest = (async () => {
      try {
        const terminalList = await listWorkspaceTerminals(workspaceId);
        const hasRunningTerminals = terminalList.items.some((terminal) => (
          terminal.status === TerminalStatus.TerminalStatusRunning
        ));

        if (!hasRunningTerminals) {
          clearWorkspaceSession(workspaceId);
          return;
        }

        hydrateWorkspaceSession(workspaceId);
        await restoreReviewSnapshot(workspaceId);
      } catch {
        // Failed validation must not overwrite a checkpoint while daemon state is unknown.
        ensureWorkspaceSessionEntry(workspaceId);
      } finally {
        preparationRequests.delete(workspaceId);
      }
    })();

    preparationRequests.set(workspaceId, preparationRequest);
    return preparationRequest;
  };

  const getReviewState = (workspaceId: string): ReviewState => (
    workspaceSessions.get(workspaceId)?.reviewState.value ?? 'idle'
  );

  const getSelectedAgentName = (workspaceId: string) => (
    ensureWorkspaceSessionEntry(workspaceId).state.terminal.selectedAgentName
  );

  const getActiveReview = () => (
    workspaceSessions.get(activeWorkspaceId.value)?.state.review ?? null
  );

  const ensureWorkspaceSessionEntry = (workspaceId: string) => {
    const existingEntry = workspaceSessions.get(workspaceId);
    if (existingEntry) {
      return existingEntry;
    }

    const state = reactive<WorkspaceSessionCheckpoint>(createFreshWorkspaceSession());
    const entry: WorkspaceSessionEntry = {
      state,
      isInitialised: false,
      lastSerialisedCheckpoint: serialiseCheckpoint(state),
      reviewData: ref(null),
      reviewState: ref('idle')
    };

    watchDebounced(state, () => {
      if (!entry.isInitialised) {
        return;
      }

      const serialisedCheckpoint = serialiseCheckpoint(state);
      if (serialisedCheckpoint === entry.lastSerialisedCheckpoint) {
        return;
      }

      storeValue(workspaceSessionStorageKey(workspaceId), serialisedCheckpoint);
      entry.lastSerialisedCheckpoint = serialisedCheckpoint;
    }, { debounce: 300, deep: true });

    workspaceSessions.set(workspaceId, entry);
    return entry;
  };

  const hydrateWorkspaceSession = (workspaceId: string) => {
    const entry = ensureWorkspaceSessionEntry(workspaceId);
    if (entry.isInitialised) {
      return;
    }

    const storedState = parseCheckpoint(getStoredValue(workspaceSessionStorageKey(workspaceId)));
    const nextState = storedState ?? createFreshWorkspaceSession();

    replaceWorkspaceSession(entry.state, nextState);

    const serialisedCheckpoint = serialiseCheckpoint(entry.state);
    storeValue(workspaceSessionStorageKey(workspaceId), serialisedCheckpoint);
    entry.lastSerialisedCheckpoint = serialisedCheckpoint;
    entry.isInitialised = true;
  };

  const restoreReviewSnapshot = async (workspaceId: string) => {
    const entry = ensureWorkspaceSessionEntry(workspaceId);
    const snapshotId = entry.state.review?.snapshotId;
    if (!snapshotId) {
      entry.reviewData.value = null;
      entry.reviewState.value = 'idle';
      return;
    }

    if (entry.reviewData.value?.id === snapshotId) {
      entry.reviewState.value = 'ready';
      return;
    }

    entry.reviewState.value = 'loading';

    try {
      const data = await getReviewSnapshot(snapshotId);
      if (data.workspaceId !== workspaceId) {
        throw new Error('Review snapshot belongs to another workspace');
      }

      entry.reviewData.value = data;
      entry.reviewState.value = 'ready';
    } catch {
      clearReview(workspaceId);
    }
  };

  return {
    activeTab,
    activeWorkspaceId,
    activateWorkspaceSession,
    beginReview,
    clearReview,
    clearWorkspaceSession,
    getReviewState,
    getSelectedAgentName,
    initialiseReview,
    isReviewOverallNoteOpen,
    isScratchpadOpen,
    prepareWorkspaceSession,
    reviewActiveFileId,
    reviewActiveScope,
    reviewCollapsedDirectories,
    reviewComments,
    reviewData,
    reviewDraftComment,
    reviewDraftCommentBody,
    reviewDraftCommentKind,
    reviewFilterText,
    reviewHideUnchanged,
    reviewOverallComment,
    reviewOverallNoteDraft,
    reviewRenderSideBySide,
    reviewReviewedFiles,
    reviewState,
    reviewWrapLines,
    selectedAgentName,
    setReviewState,
    terminalActivePane,
    terminalZoomedPane
  };
});

const reviewScopes: readonly ReviewScope[] = ['pull-request', 'working-tree', 'last-commit', 'current'];
const reviewCommentKinds: readonly ReviewCommentKind[] = ['feedback', 'question'];
const commentSides = ['original', 'modified', 'file'] as const;
const workspaceSessionStorageKey = (workspaceId: string) => `wade:workspace-session:${workspaceId}`;

const createFreshReviewCheckpoint = (
  snapshotId: string,
  activeScope: ReviewScope = 'working-tree'
): ReviewCheckpoint => ({
  snapshotId,
  activeScope,
  activeFileId: null,
  filterText: '',
  reviewedFiles: {},
  collapsedDirectories: {},
  comments: [],
  overallComment: '',
  draftComment: null,
  draftCommentBody: '',
  draftCommentKind: 'feedback',
  isOverallNoteOpen: false,
  overallNoteDraft: '',
  hideUnchanged: true,
  renderSideBySide: true,
  wrapLines: true
});

const createFreshWorkspaceSession = (): WorkspaceSessionCheckpoint => ({
  activeTab: WorkspaceTabs.Terminal,
  isScratchpadOpen: false,
  terminal: {
    activePane: TerminalPanes.Agent,
    zoomedPane: null,
    selectedAgentName: ''
  },
  review: null
});

const cloneReviewCheckpoint = (review: ReviewCheckpoint): ReviewCheckpoint => ({
  snapshotId: review.snapshotId,
  activeScope: review.activeScope,
  activeFileId: review.activeFileId,
  filterText: review.filterText,
  reviewedFiles: { ...review.reviewedFiles },
  collapsedDirectories: { ...review.collapsedDirectories },
  comments: review.comments.map((comment) => ({ ...comment })),
  overallComment: review.overallComment,
  draftComment: review.draftComment ? { ...review.draftComment } : null,
  draftCommentBody: review.draftCommentBody,
  draftCommentKind: review.draftCommentKind,
  isOverallNoteOpen: review.isOverallNoteOpen,
  overallNoteDraft: review.overallNoteDraft,
  hideUnchanged: review.hideUnchanged,
  renderSideBySide: review.renderSideBySide,
  wrapLines: review.wrapLines
});

const createCheckpoint = (state: WorkspaceSessionCheckpoint): WorkspaceSessionCheckpoint => ({
  activeTab: state.activeTab,
  isScratchpadOpen: state.isScratchpadOpen,
  terminal: {
    activePane: state.terminal.activePane,
    zoomedPane: state.terminal.zoomedPane,
    selectedAgentName: state.terminal.selectedAgentName
  },
  review: state.review ? cloneReviewCheckpoint(state.review) : null
});

const serialiseCheckpoint = (state: WorkspaceSessionCheckpoint) => JSON.stringify(createCheckpoint(state));

const replaceWorkspaceSession = (
  state: WorkspaceSessionCheckpoint,
  nextState: WorkspaceSessionCheckpoint
) => {
  state.activeTab = nextState.activeTab;
  state.isScratchpadOpen = nextState.isScratchpadOpen;
  state.terminal.activePane = nextState.terminal.activePane;
  state.terminal.zoomedPane = nextState.terminal.zoomedPane;
  state.terminal.selectedAgentName = nextState.terminal.selectedAgentName;
  state.review = nextState.review ? cloneReviewCheckpoint(nextState.review) : null;
};

const isRecord = (value: unknown): value is Record<string, unknown> => (
  typeof value === 'object' && value !== null && !Array.isArray(value)
);

const isWorkspaceTab = (value: unknown): value is WorkspaceTab => (
  typeof value === 'string' && (workspaceTabs as readonly string[]).includes(value)
);

const isTerminalPane = (value: unknown): value is TerminalPaneId => (
  typeof value === 'string' && (terminalPanes as readonly string[]).includes(value)
);

const isReviewScope = (value: unknown): value is ReviewScope => (
  typeof value === 'string' && (reviewScopes as readonly string[]).includes(value)
);

const isReviewCommentKind = (value: unknown): value is ReviewCommentKind => (
  typeof value === 'string' && (reviewCommentKinds as readonly string[]).includes(value)
);

const isCommentSide = (value: unknown): value is ReviewComment['side'] => (
  typeof value === 'string' && (commentSides as readonly string[]).includes(value)
);

const isLineNumber = (value: unknown): value is number | null => (
  value === null || (typeof value === 'number' && Number.isInteger(value) && value > 0)
);

const parseBooleanRecord = (value: unknown): Record<string, boolean> | undefined => {
  if (!isRecord(value) || !Object.values(value).every((entry) => typeof entry === 'boolean')) {
    return undefined;
  }

  return { ...value } as Record<string, boolean>;
};

const parseReviewComment = (value: unknown): ReviewComment | undefined => {
  if (!isRecord(value)
    || typeof value.id !== 'string'
    || typeof value.fileId !== 'string'
    || !isReviewScope(value.scope)
    || !isCommentSide(value.side)
    || !isReviewCommentKind(value.kind)
    || !isLineNumber(value.startLine)
    || !isLineNumber(value.endLine)
    || typeof value.body !== 'string') {
    return undefined;
  }

  return {
    id: value.id,
    fileId: value.fileId,
    scope: value.scope,
    side: value.side,
    kind: value.kind,
    startLine: value.startLine,
    endLine: value.endLine,
    body: value.body
  };
};

const parseDraftReviewComment = (value: unknown): DraftReviewComment | null | undefined => {
  if (value === null) {
    return null;
  }

  if (!isRecord(value)
    || typeof value.fileId !== 'string'
    || typeof value.filePath !== 'string'
    || !isReviewScope(value.scope)
    || !isCommentSide(value.side)
    || !isLineNumber(value.startLine)
    || !isLineNumber(value.endLine)) {
    return undefined;
  }

  return {
    fileId: value.fileId,
    filePath: value.filePath,
    scope: value.scope,
    side: value.side,
    startLine: value.startLine,
    endLine: value.endLine
  };
};

const parseReviewCheckpoint = (value: unknown): ReviewCheckpoint | null | undefined => {
  if (value === null) {
    return null;
  }

  if (!isRecord(value)
    || typeof value.snapshotId !== 'string'
    || value.snapshotId === ''
    || !isReviewScope(value.activeScope)
    || (value.activeFileId !== null && typeof value.activeFileId !== 'string')
    || typeof value.filterText !== 'string'
    || !Array.isArray(value.comments)
    || typeof value.overallComment !== 'string'
    || typeof value.draftCommentBody !== 'string'
    || !isReviewCommentKind(value.draftCommentKind)
    || typeof value.isOverallNoteOpen !== 'boolean'
    || typeof value.overallNoteDraft !== 'string'
    || typeof value.hideUnchanged !== 'boolean'
    || typeof value.renderSideBySide !== 'boolean'
    || typeof value.wrapLines !== 'boolean') {
    return undefined;
  }

  const reviewedFiles = parseBooleanRecord(value.reviewedFiles);
  const collapsedDirectories = parseBooleanRecord(value.collapsedDirectories);
  const comments = value.comments.map(parseReviewComment);
  const draftComment = parseDraftReviewComment(value.draftComment);
  if (!reviewedFiles
    || !collapsedDirectories
    || comments.some((comment) => !comment)
    || draftComment === undefined) {
    return undefined;
  }

  return {
    snapshotId: value.snapshotId,
    activeScope: value.activeScope,
    activeFileId: value.activeFileId,
    filterText: value.filterText,
    reviewedFiles,
    collapsedDirectories,
    comments: comments as ReviewComment[],
    overallComment: value.overallComment,
    draftComment,
    draftCommentBody: value.draftCommentBody,
    draftCommentKind: value.draftCommentKind,
    isOverallNoteOpen: value.isOverallNoteOpen,
    overallNoteDraft: value.overallNoteDraft,
    hideUnchanged: value.hideUnchanged,
    renderSideBySide: value.renderSideBySide,
    wrapLines: value.wrapLines
  };
};

const parseCheckpoint = (storedCheckpoint: string | null): WorkspaceSessionCheckpoint | undefined => {
  if (storedCheckpoint === null) {
    return undefined;
  }

  let value: unknown;
  try {
    value = JSON.parse(storedCheckpoint);
  } catch {
    return undefined;
  }

  if (!isRecord(value)
    || !isWorkspaceTab(value.activeTab)
    || typeof value.isScratchpadOpen !== 'boolean'
    || !isRecord(value.terminal)
    || !isTerminalPane(value.terminal.activePane)
    || (value.terminal.zoomedPane !== null && !isTerminalPane(value.terminal.zoomedPane))
    || typeof value.terminal.selectedAgentName !== 'string') {
    return undefined;
  }

  const review = parseReviewCheckpoint(value.review);
  if (review === undefined) {
    return undefined;
  }

  return {
    activeTab: value.activeTab,
    isScratchpadOpen: value.isScratchpadOpen,
    terminal: {
      activePane: value.terminal.activePane,
      zoomedPane: value.terminal.zoomedPane,
      selectedAgentName: value.terminal.selectedAgentName
    },
    review
  };
};

const getStoredValue = (key: string) => {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
};

const storeValue = (key: string, value: string) => {
  try {
    window.localStorage.setItem(key, value);
  } catch {
    return;
  }
};

const removeStoredValue = (key: string) => {
  try {
    window.localStorage.removeItem(key);
  } catch {
    return;
  }
};
