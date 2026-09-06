import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReviewData } from '@/types/review';
import { useWorkspaceSessionStore } from './useWorkspaceSessionStore';

const api = vi.hoisted(() => ({
  getReviewSnapshot: vi.fn(),
  listWorkspaceTerminals: vi.fn()
}));

vi.mock('@/api/generated/wade', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/generated/wade')>()),
  getReviewSnapshot: api.getReviewSnapshot,
  listWorkspaceTerminals: api.listWorkspaceTerminals
}));

const reviewData: ReviewData = {
  branch: { name: 'feature', ref: 'refs/heads/feature', remote: null },
  createdAt: '2026-01-01T00:00:00Z',
  files: [],
  id: 'snapshot-1',
  pullRequest: null,
  workspaceId: 'workspace-restore'
};

const legacyCheckpoint = {
  activeTab: 'review',
  isScratchpadOpen: true,
  terminal: {
    activePane: 'misc',
    zoomedPane: 'agent',
    selectedAgentName: 'Claude'
  },
  review: {
    snapshotId: 'snapshot-1',
    activeScope: 'last-commit',
    activeFileId: 'file-1',
    filterText: 'app',
    reviewedFiles: { 'file-1': true },
    collapsedDirectories: { src: true },
    comments: [
      {
        body: 'Human comment',
        endLine: 3,
        fileId: 'file-1',
        id: 'comment-1',
        kind: 'question',
        scope: 'last-commit',
        side: 'original',
        startLine: 2
      }
    ],
    overallComment: 'Human overall note',
    draftComment: {
      endLine: 3,
      fileId: 'file-1',
      filePath: 'src/app.ts',
      scope: 'last-commit',
      side: 'modified',
      startLine: 2
    },
    draftCommentBody: 'Draft human comment',
    draftCommentKind: 'feedback',
    isOverallNoteOpen: true,
    overallNoteDraft: 'Draft overall note',
    hideUnchanged: false,
    renderSideBySide: false,
    wrapLines: false,
    annotations: [{ summary: 'agent-only-marker' }]
  }
};

const storageKey = 'wade:workspace-session:workspace-restore';

const expectRestoredHumanState = (store: ReturnType<typeof useWorkspaceSessionStore>) => {
  expect({
    activeTab: store.activeTab,
    isScratchpadOpen: store.isScratchpadOpen,
    terminalActivePane: store.terminalActivePane,
    terminalZoomedPane: store.terminalZoomedPane,
    selectedAgentName: store.selectedAgentName,
    activeScope: store.reviewActiveScope,
    activeFileId: store.reviewActiveFileId,
    filterText: store.reviewFilterText,
    reviewedFiles: store.reviewReviewedFiles,
    collapsedDirectories: store.reviewCollapsedDirectories,
    comments: store.reviewComments,
    overallComment: store.reviewOverallComment,
    draftComment: store.reviewDraftComment,
    draftCommentBody: store.reviewDraftCommentBody,
    draftCommentKind: store.reviewDraftCommentKind,
    isOverallNoteOpen: store.isReviewOverallNoteOpen,
    overallNoteDraft: store.reviewOverallNoteDraft,
    hideUnchanged: store.reviewHideUnchanged,
    renderSideBySide: store.reviewRenderSideBySide,
    wrapLines: store.reviewWrapLines
  }).toEqual({
    activeTab: 'review',
    isScratchpadOpen: true,
    terminalActivePane: 'misc',
    terminalZoomedPane: 'agent',
    selectedAgentName: 'Claude',
    activeScope: 'last-commit',
    activeFileId: 'file-1',
    filterText: 'app',
    reviewedFiles: { 'file-1': true },
    collapsedDirectories: { src: true },
    comments: legacyCheckpoint.review.comments,
    overallComment: 'Human overall note',
    draftComment: legacyCheckpoint.review.draftComment,
    draftCommentBody: 'Draft human comment',
    draftCommentKind: 'feedback',
    isOverallNoteOpen: true,
    overallNoteDraft: 'Draft overall note',
    hideUnchanged: false,
    renderSideBySide: false,
    wrapLines: false
  });
};

describe('useWorkspaceSessionStore review checkpoint boundary', () => {
  beforeEach(() => {
    localStorage.clear();
    api.getReviewSnapshot.mockReset();
    api.listWorkspaceTerminals.mockReset();
    api.listWorkspaceTerminals.mockResolvedValue({ items: [{ status: 'running' }] });
    api.getReviewSnapshot.mockResolvedValue(reviewData);
  });

  it('restores legacy human state, defaults annotation visibility, and persists only the presentation preference', async () => {
    localStorage.setItem(storageKey, JSON.stringify(legacyCheckpoint));

    const firstPinia = createPinia();
    setActivePinia(firstPinia);
    const firstStore = useWorkspaceSessionStore();
    firstStore.activateWorkspaceSession('workspace-restore');
    await firstStore.prepareWorkspaceSession('workspace-restore');

    expectRestoredHumanState(firstStore);
    expect(firstStore.reviewShowAgentAnnotations).toBe(true);
    expect(firstStore.reviewData?.id).toBe('snapshot-1');

    const firstSerialisedCheckpoint = JSON.parse(localStorage.getItem(storageKey) ?? '{}');
    expect(firstSerialisedCheckpoint.review).not.toHaveProperty('annotations');
    expect(JSON.stringify(firstSerialisedCheckpoint)).not.toContain('agent-only-marker');

    firstStore.reviewShowAgentAnnotations = false;
    await vi.waitFor(() => {
      const checkpoint = JSON.parse(localStorage.getItem(storageKey) ?? '{}');
      expect(checkpoint.review.showAgentAnnotations).toBe(false);
    });

    const secondPinia = createPinia();
    setActivePinia(secondPinia);
    const secondStore = useWorkspaceSessionStore();
    secondStore.activateWorkspaceSession('workspace-restore');
    await secondStore.prepareWorkspaceSession('workspace-restore');

    expectRestoredHumanState(secondStore);
    expect(secondStore.reviewShowAgentAnnotations).toBe(false);
    const secondSerialisedCheckpoint = JSON.parse(localStorage.getItem(storageKey) ?? '{}');
    expect(secondSerialisedCheckpoint.review).not.toHaveProperty('annotations');
    expect(JSON.stringify(secondSerialisedCheckpoint)).not.toContain('agent-only-marker');
  });
});
