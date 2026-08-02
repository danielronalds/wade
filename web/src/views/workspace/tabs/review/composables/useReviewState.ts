import { computed, reactive, toValue, type MaybeRefOrGetter } from 'vue';

export type ReviewState = 'idle' | 'loading' | 'ready' | 'error';

const reviewStates = reactive<Record<string, ReviewState | undefined>>({});

export const isReviewInProgressState = (state: ReviewState) => state === 'loading' || state === 'ready';

export const useReviewState = (workspaceId: MaybeRefOrGetter<string>) => computed<ReviewState>(() => {
  const resolvedWorkspaceId = toValue(workspaceId);
  if (resolvedWorkspaceId === '') {
    return 'idle';
  }

  return reviewStates[resolvedWorkspaceId] ?? 'idle';
});

export const setReviewState = (workspaceId: string, state: ReviewState) => {
  if (workspaceId === '') {
    return;
  }

  reviewStates[workspaceId] = state;
};

export const clearReviewState = (workspaceId: string) => {
  if (workspaceId === '') {
    return;
  }

  delete reviewStates[workspaceId];
};
