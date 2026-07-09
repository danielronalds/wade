import { createSharedComposable } from '@vueuse/core';
import { computed, readonly, ref } from 'vue';
import { listActiveProjectSessions } from '@/api/generated/wade';

const normaliseActiveSessions = (sessions: unknown): string[] => {
  if (!Array.isArray(sessions)) {
    return [];
  }

  return Array.from(new Set(
    sessions.filter((session): session is string => typeof session === 'string' && session.length > 0)
  )).sort((firstSession, secondSession) => firstSession.localeCompare(secondSession));
};

export const useActiveSessions = createSharedComposable(() => {
  const storedActiveSessions = ref<string[]>([]);
  const isSyncing = ref(false);
  const error = ref('');

  const activeSessions = computed(() => normaliseActiveSessions(storedActiveSessions.value));

  let syncRequest: Promise<string[] | undefined> | undefined;

  const syncActiveSessions = () => {
    if (syncRequest) {
      return syncRequest;
    }

    isSyncing.value = true;
    error.value = '';

    syncRequest = (async () => {
      try {
        const { sessions } = await listActiveProjectSessions();
        storedActiveSessions.value = normaliseActiveSessions(sessions);

        return activeSessions.value;
      } catch (requestError) {
        error.value = requestError instanceof Error ? requestError.message : 'Sessions request failed';
        return undefined;
      } finally {
        isSyncing.value = false;
        syncRequest = undefined;
      }
    })();

    return syncRequest;
  };

  return {
    activeSessions: readonly(activeSessions),
    error: readonly(error),
    isSyncing: readonly(isSyncing),
    syncActiveSessions
  };
});
