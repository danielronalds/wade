import { computed, reactive, toValue, type MaybeRefOrGetter } from 'vue';

export type ReviewSessionState = 'idle' | 'loading' | 'ready' | 'error';

const reviewSessionStates = reactive<Record<string, ReviewSessionState | undefined>>({});

export const isReviewInProgressState = (state: ReviewSessionState) => state === 'loading' || state === 'ready';

export const useReviewSessionState = (projectName: MaybeRefOrGetter<string>) => computed<ReviewSessionState>(() => {
  const resolvedProjectName = toValue(projectName);
  if (resolvedProjectName === '') {
    return 'idle';
  }

  return reviewSessionStates[resolvedProjectName] ?? 'idle';
});

export const setReviewSessionState = (projectName: string, state: ReviewSessionState) => {
  if (projectName === '') {
    return;
  }

  reviewSessionStates[projectName] = state;
};

export const clearReviewSessionState = (projectName: string) => {
  if (projectName === '') {
    return;
  }

  delete reviewSessionStates[projectName];
};
