import { computed, ref, type Ref } from 'vue';
import type { PaletteNotice } from '../types';

type PaletteRequestStateOptions = {
  errorTitle: string;
  warningTitle?: string;
  warningMessages?: Ref<readonly string[]>;
};

const errorMessage = (error: unknown, fallback: string) => error instanceof Error
  ? error.message
  : fallback;

export const usePaletteRequestState = ({
  errorTitle,
  warningTitle,
  warningMessages
}: PaletteRequestStateOptions) => {
  const query = ref('');
  const loadError = ref('');
  const actionError = ref('');
  const isLoading = ref(false);
  const isActing = ref(false);

  const notice = computed<PaletteNotice | undefined>(() => {
    const error = actionError.value || loadError.value;
    if (error !== '') {
      return {
        tone: 'error',
        title: errorTitle,
        messages: [error]
      };
    }

    const warnings = warningMessages?.value ?? [];
    if (warningTitle && warnings.length > 0) {
      return {
        tone: 'warning',
        title: warningTitle,
        messages: warnings
      };
    }

    return undefined;
  });

  const clearErrors = () => {
    loadError.value = '';
    actionError.value = '';
  };

  const clearActionError = () => {
    actionError.value = '';
  };

  const setLoadError = (error: unknown, fallback: string) => {
    loadError.value = errorMessage(error, fallback);
  };

  const setActionError = (error: unknown, fallback: string) => {
    actionError.value = errorMessage(error, fallback);
  };

  const updateQuery = (nextQuery: string) => {
    query.value = nextQuery;
    clearActionError();
  };

  return {
    actionError,
    clearActionError,
    clearErrors,
    isActing,
    isLoading,
    loadError,
    notice,
    query,
    setActionError,
    setLoadError,
    updateQuery
  };
};
