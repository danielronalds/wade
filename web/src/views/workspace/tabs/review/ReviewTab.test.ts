import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReviewData } from '@/types/review';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';
import ReviewTab from './ReviewTab.vue';

const api = vi.hoisted(() => ({
  deleteReviewSnapshot: vi.fn(),
  getReviewSnapshotFileContents: vi.fn()
}));

const terminal = vi.hoisted(() => ({
  sendWorkspaceTerminalInput: vi.fn()
}));

const annotationSynchronisation = vi.hoisted(() => ({
  annotations: {
    value: [
      {
        author: 'Agent author',
        createdAt: '2026-01-01T00:00:00Z',
        endLine: 1,
        fileId: 'file-1',
        id: 'annotation-1',
        rationale: 'Agent rationale',
        scope: 'working-tree',
        side: 'modified',
        snapshotId: 'snapshot-1',
        startLine: 1,
        summary: 'Agent summary'
      }
    ]
  },
  errorMessage: { value: '' },
  retry: vi.fn(),
  status: { value: 'ready' },
  stop: vi.fn(),
  synchronise: vi.fn()
}));

vi.mock('@/api/generated/wade', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/generated/wade')>()),
  deleteReviewSnapshot: api.deleteReviewSnapshot,
  getReviewSnapshotFileContents: api.getReviewSnapshotFileContents,
  sendWorkspaceTerminalInput: terminal.sendWorkspaceTerminalInput
}));

vi.mock('@/views/workspace/tabs/review/composables/useReviewAnnotations', () => ({
  useReviewAnnotations: () => annotationSynchronisation
}));

const reviewData: ReviewData = {
  branch: { name: 'feature', ref: 'refs/heads/feature', remote: null },
  createdAt: '2026-01-01T00:00:00Z',
  files: [
    {
      gitDiff: {
        displayPath: 'src/app.ts',
        hasModified: true,
        hasOriginal: true,
        newPath: 'src/app.ts',
        oldPath: 'src/app.ts',
        status: 'modified'
      },
      hasWorkingTreeFile: true,
      id: 'file-1',
      inGitDiff: true,
      inLastCommit: false,
      inPullRequest: false,
      lastCommit: null,
      path: 'src/app.ts',
      pullRequest: null,
      worktreeStatus: 'modified'
    }
  ],
  id: 'snapshot-1',
  pullRequest: null,
  workspaceId: 'workspace-1'
};

const ReviewDiffViewerStub = defineComponent({
  name: 'ReviewDiffViewer',
  props: {
    annotations: { type: Array, required: true }
  },
  template: `<div class="review-diff-viewer-stub" :data-annotation-count="annotations.length" :data-annotation-ids="annotations.map((annotation) => annotation.id).join(',')">
    <span v-for="annotation in annotations" :key="annotation.id" class="review-diff-annotation-id">{{ annotation.id }}</span>
  </div>`
});

const multiFileReviewData: ReviewData = {
  ...reviewData,
  files: [
    {
      ...reviewData.files[0],
      inLastCommit: true,
      lastCommit: {
        displayPath: 'src/app.ts',
        hasModified: true,
        hasOriginal: true,
        newPath: 'src/app.ts',
        oldPath: 'src/app.ts',
        status: 'modified'
      }
    },
    {
      gitDiff: {
        displayPath: 'src/other.ts',
        hasModified: true,
        hasOriginal: true,
        newPath: 'src/other.ts',
        oldPath: 'src/other.ts',
        status: 'modified'
      },
      hasWorkingTreeFile: true,
      id: 'file-2',
      inGitDiff: true,
      inLastCommit: true,
      inPullRequest: false,
      lastCommit: {
        displayPath: 'src/other.ts',
        hasModified: true,
        hasOriginal: true,
        newPath: 'src/other.ts',
        oldPath: 'src/other.ts',
        status: 'modified'
      },
      path: 'src/other.ts',
      pullRequest: null,
      worktreeStatus: 'modified'
    }
  ]
};

const annotationsAcrossFilesAndScopes = [
  { ...annotationSynchronisation.annotations.value[0], id: 'working-file-1', fileId: 'file-1', scope: 'working-tree' },
  { ...annotationSynchronisation.annotations.value[0], id: 'working-file-2', fileId: 'file-2', scope: 'working-tree' },
  { ...annotationSynchronisation.annotations.value[0], id: 'last-file-1', fileId: 'file-1', scope: 'last-commit' },
  { ...annotationSynchronisation.annotations.value[0], id: 'last-file-2', fileId: 'file-2', scope: 'last-commit' }
];

const mountReadyReview = (workspaceId = 'workspace-1', data = reviewData) => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const workspaceSessionStore = useWorkspaceSessionStore();
  workspaceSessionStore.activateWorkspaceSession(workspaceId);
  workspaceSessionStore.initialiseReview(workspaceId, data, 'working-tree');

  const wrapper = mount(ReviewTab, {
    props: { workspaceId, isActive: true },
    global: {
      plugins: [pinia],
      stubs: {
        ReviewCommentEditor: true,
        ReviewDiffViewer: ReviewDiffViewerStub,
        ReviewMarkdownViewer: true
      }
    }
  });

  return { wrapper, workspaceSessionStore };
};

const getCancelReview = (wrapper: ReturnType<typeof mount>) =>
  (wrapper.vm as unknown as { cancelReview: () => Promise<void> }).cancelReview;

describe('ReviewTab agent annotations', () => {
  beforeEach(() => {
    api.deleteReviewSnapshot.mockReset();
    api.deleteReviewSnapshot.mockResolvedValue(undefined);
    api.getReviewSnapshotFileContents.mockReset();
    api.getReviewSnapshotFileContents.mockResolvedValue({ originalContent: 'before\n', modifiedContent: 'after\n' });
    terminal.sendWorkspaceTerminalInput.mockReset();
    terminal.sendWorkspaceTerminalInput.mockResolvedValue(undefined);
    annotationSynchronisation.annotations.value = [
      {
        author: 'Agent author',
        createdAt: '2026-01-01T00:00:00Z',
        endLine: 1,
        fileId: 'file-1',
        id: 'annotation-1',
        rationale: 'Agent rationale',
        scope: 'working-tree',
        side: 'modified',
        snapshotId: 'snapshot-1',
        startLine: 1,
        summary: 'Agent summary'
      }
    ];
    annotationSynchronisation.retry.mockClear();
    annotationSynchronisation.stop.mockClear();
    annotationSynchronisation.synchronise.mockClear();
  });

  it('toggles annotation cards and counts without enabling review submission', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const workspaceSessionStore = useWorkspaceSessionStore();
    workspaceSessionStore.activateWorkspaceSession('workspace-1');
    workspaceSessionStore.initialiseReview('workspace-1', reviewData, 'working-tree');

    const wrapper = mount(ReviewTab, {
      props: { workspaceId: 'workspace-1', isActive: true },
      global: {
        plugins: [pinia],
        stubs: {
          ReviewCommentEditor: true,
          ReviewDiffViewer: ReviewDiffViewerStub,
          ReviewMarkdownViewer: true
        }
      }
    });
    await flushPromises();

    expect(annotationSynchronisation.synchronise).toHaveBeenCalledWith('snapshot-1');
    expect(wrapper.find('.review-annotation-count').text()).toBe('A1');
    expect(wrapper.find('.review-diff-viewer-stub').attributes('data-annotation-count')).toBe('1');
    expect(wrapper.find('.review-finish-button').attributes('disabled')).toBeDefined();

    await wrapper.get('[aria-label="Hide agent annotations"]').trigger('click');

    expect(wrapper.find('.review-annotation-count').exists()).toBe(false);
    expect(wrapper.find('.review-diff-viewer-stub').attributes('data-annotation-count')).toBe('0');
    expect(workspaceSessionStore.reviewShowAgentAnnotations).toBe(false);

    wrapper.unmount();
    expect(annotationSynchronisation.stop).toHaveBeenCalled();
  });

  it('sends only human review content through the terminal boundary and deletes after stopping annotation sync', async () => {
    const { wrapper, workspaceSessionStore } = mountReadyReview();
    workspaceSessionStore.reviewComments = [
      {
        body: 'Human feedback',
        endLine: 3,
        fileId: 'file-1',
        id: 'human-feedback',
        kind: 'feedback',
        scope: 'working-tree',
        side: 'modified',
        startLine: 2
      },
      {
        body: 'Human question',
        endLine: null,
        fileId: 'file-1',
        id: 'human-question',
        kind: 'question',
        scope: 'working-tree',
        side: 'file',
        startLine: null
      }
    ];
    workspaceSessionStore.reviewOverallComment = 'Human overall note';
    workspaceSessionStore.selectedAgentName = 'Claude';
    await flushPromises();
    const events: string[] = [];
    annotationSynchronisation.stop.mockImplementation(() => events.push('stop'));
    api.deleteReviewSnapshot.mockImplementation(async () => {
      events.push('delete');
    });

    await wrapper.get('.review-finish-button').trigger('click');
    await vi.waitFor(() => expect(terminal.sendWorkspaceTerminalInput).toHaveBeenCalledOnce());
    await vi.waitFor(() => expect(api.deleteReviewSnapshot).toHaveBeenCalledWith('snapshot-1'));

    const expectedPrompt = `Please address the following review comments.

Instructions:
- Feedback/improvement items request code changes. Action them. If an item is unclear, ask a clarifying question before changing code.
- Question items are reviewer questions. Answer them briefly and clearly. Do not make code changes for question items.

Reviewer note:
Human overall note

Feedback/improvements:
1. [working tree] src/app.ts:2-3 (new)
   Human feedback

Questions:
1. [working tree] src/app.ts
   Human question`;
    expect(terminal.sendWorkspaceTerminalInput).toHaveBeenCalledWith('workspace-1', 'agent:claude', {
      mode: 'bracketed-paste',
      text: expectedPrompt
    });
    expect(expectedPrompt).not.toContain('Agent summary');
    expect(expectedPrompt).not.toContain('Agent rationale');
    expect(expectedPrompt).not.toContain('Agent author');
    expect(events.indexOf('stop')).toBeGreaterThanOrEqual(0);
    expect(events.indexOf('stop')).toBeLessThan(events.indexOf('delete'));

    wrapper.unmount();
  });

  it('stops annotation sync before deleting on cancellation through the UI', async () => {
    const { wrapper } = mountReadyReview();
    const events: string[] = [];
    annotationSynchronisation.stop.mockImplementation(() => events.push('stop'));
    api.deleteReviewSnapshot.mockImplementation(async () => {
      events.push('delete');
    });

    await wrapper.get('[aria-label="Cancel review"]').trigger('click');
    await vi.waitFor(() => expect(api.deleteReviewSnapshot).toHaveBeenCalledWith('snapshot-1'));

    expect(events.indexOf('stop')).toBeGreaterThanOrEqual(0);
    expect(events.indexOf('stop')).toBeLessThan(events.indexOf('delete'));
    wrapper.unmount();
  });

  it('filters annotations by file and captured scope without changing human state', async () => {
    annotationSynchronisation.annotations.value = annotationsAcrossFilesAndScopes;
    const { wrapper, workspaceSessionStore } = mountReadyReview('workspace-1', multiFileReviewData);
    const humanComments = [
      {
        body: 'Keep this human comment',
        endLine: 1,
        fileId: 'file-1',
        id: 'human-comment',
        kind: 'feedback' as const,
        scope: 'working-tree' as const,
        side: 'original' as const,
        startLine: 1
      }
    ];
    workspaceSessionStore.reviewComments = humanComments;
    await flushPromises();

    const annotationIds = () => wrapper.find('.review-diff-viewer-stub').attributes('data-annotation-ids');
    const fileButton = (path: string) => wrapper.findAll('.review-file-row').find((row) => row.text().includes(path));
    const scopeButton = (label: string) =>
      wrapper.findAll('.review-scopes button').find((button) => button.text().includes(label));

    expect(annotationIds()).toBe('working-file-1');
    expect(wrapper.findAll('.review-annotation-count').map((count) => count.text())).toEqual(['A1', 'A1']);

    await fileButton('other.ts')?.trigger('click');
    await flushPromises();
    expect(annotationIds()).toBe('working-file-2');

    await scopeButton('Current files')?.trigger('click');
    await flushPromises();
    expect(annotationIds()).toBe('');
    expect(wrapper.findAll('.review-annotation-count')).toHaveLength(0);

    await scopeButton('Last commit')?.trigger('click');
    await flushPromises();
    expect(annotationIds()).toBe('last-file-2');
    expect(wrapper.findAll('.review-annotation-count').map((count) => count.text())).toEqual(['A1', 'A1']);

    await fileButton('app.ts')?.trigger('click');
    await flushPromises();
    expect(annotationIds()).toBe('last-file-1');

    await wrapper.get('[aria-label="Hide agent annotations"]').trigger('click');
    expect(annotationIds()).toBe('');
    expect(wrapper.findAll('.review-annotation-count')).toHaveLength(0);
    expect(workspaceSessionStore.reviewComments).toEqual(humanComments);

    wrapper.unmount();
  });

  it('does not reconnect annotations when cancellation rejects after unmount', async () => {
    const { wrapper } = mountReadyReview();
    let rejectDelete: ((error: unknown) => void) | undefined;
    api.deleteReviewSnapshot.mockImplementation(
      () =>
        new Promise<void>((_resolve, reject) => {
          rejectDelete = reject;
        })
    );
    annotationSynchronisation.synchronise.mockClear();

    const cancellation = getCancelReview(wrapper)();
    wrapper.unmount();
    rejectDelete?.(new Error('delete failed'));
    await cancellation;

    expect(annotationSynchronisation.synchronise).not.toHaveBeenCalled();
  });

  it('clears the captured review when cancellation resolves after unmount', async () => {
    const { wrapper, workspaceSessionStore } = mountReadyReview();
    let resolveDelete: (() => void) | undefined;
    api.deleteReviewSnapshot.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveDelete = resolve;
        })
    );

    const cancellation = getCancelReview(wrapper)();
    wrapper.unmount();
    resolveDelete?.();
    await cancellation;

    expect(workspaceSessionStore.getReviewState('workspace-1')).toBe('idle');
  });

  it('clears only the captured review when cancellation resolves after the workspace changes', async () => {
    const { wrapper, workspaceSessionStore } = mountReadyReview();
    let resolveDelete: (() => void) | undefined;
    api.deleteReviewSnapshot.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveDelete = resolve;
        })
    );

    const cancellation = getCancelReview(wrapper)();
    const nextReviewData = { ...reviewData, id: 'snapshot-2', workspaceId: 'workspace-2' };
    workspaceSessionStore.activateWorkspaceSession('workspace-2');
    workspaceSessionStore.initialiseReview('workspace-2', nextReviewData, 'working-tree');
    resolveDelete?.();
    await cancellation;

    expect(workspaceSessionStore.getReviewState('workspace-1')).toBe('idle');
    expect(workspaceSessionStore.getReviewState('workspace-2')).toBe('ready');
    wrapper.unmount();
  });

  it('preserves a replacement review when cancellation resolves after it starts', async () => {
    const { wrapper, workspaceSessionStore } = mountReadyReview();
    let resolveDelete: (() => void) | undefined;
    api.deleteReviewSnapshot.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveDelete = resolve;
        })
    );

    const cancellation = getCancelReview(wrapper)();
    workspaceSessionStore.initialiseReview('workspace-1', { ...reviewData, id: 'snapshot-2' }, 'working-tree');
    resolveDelete?.();
    await cancellation;

    expect(workspaceSessionStore.getReviewState('workspace-1')).toBe('ready');
    expect(workspaceSessionStore.reviewData?.id).toBe('snapshot-2');
    wrapper.unmount();
  });

  it('does not reconnect annotations when cancellation rejects after the workspace changes', async () => {
    const { wrapper, workspaceSessionStore } = mountReadyReview();
    let rejectDelete: ((error: unknown) => void) | undefined;
    api.deleteReviewSnapshot.mockImplementation(
      () =>
        new Promise<void>((_resolve, reject) => {
          rejectDelete = reject;
        })
    );
    annotationSynchronisation.synchronise.mockClear();

    const cancellation = getCancelReview(wrapper)();
    const nextReviewData = { ...reviewData, id: 'snapshot-2', workspaceId: 'workspace-2' };
    workspaceSessionStore.activateWorkspaceSession('workspace-2');
    workspaceSessionStore.initialiseReview('workspace-2', nextReviewData, 'working-tree');
    rejectDelete?.(new Error('delete failed'));
    await cancellation;

    expect(annotationSynchronisation.synchronise).not.toHaveBeenCalledWith('snapshot-1');
    wrapper.unmount();
  });

  it('does not reconnect annotations when cancellation rejects after the snapshot changes', async () => {
    const { wrapper, workspaceSessionStore } = mountReadyReview();
    let rejectDelete: ((error: unknown) => void) | undefined;
    api.deleteReviewSnapshot.mockImplementation(
      () =>
        new Promise<void>((_resolve, reject) => {
          rejectDelete = reject;
        })
    );
    annotationSynchronisation.synchronise.mockClear();

    const cancellation = getCancelReview(wrapper)();
    const nextReviewData = { ...reviewData, id: 'snapshot-2' };
    workspaceSessionStore.initialiseReview('workspace-1', nextReviewData, 'working-tree');
    rejectDelete?.(new Error('delete failed'));
    await cancellation;

    expect(annotationSynchronisation.synchronise).not.toHaveBeenCalledWith('snapshot-1');
    wrapper.unmount();
  });

  it('reconnects annotations after a cancellation failure in the same review', async () => {
    const { wrapper, workspaceSessionStore } = mountReadyReview();
    api.deleteReviewSnapshot.mockRejectedValueOnce(new Error('delete failed'));
    annotationSynchronisation.synchronise.mockClear();

    await getCancelReview(wrapper)();

    expect(annotationSynchronisation.synchronise).toHaveBeenCalledWith('snapshot-1');
    expect(workspaceSessionStore.getReviewState('workspace-1')).toBe('ready');
    wrapper.unmount();
  });
});
