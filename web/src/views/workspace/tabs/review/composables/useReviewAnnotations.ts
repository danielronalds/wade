import { ref } from 'vue';
import {
  getSubscribeReviewAnnotationEventsUrl,
  listReviewAnnotations,
  type ReviewAnnotation
} from '@/api/generated/wade';
import type { AnnotationSyncStatus } from '@/types/review';

export const useReviewAnnotations = () => {
  const annotations = ref<ReviewAnnotation[]>([]);
  const status = ref<AnnotationSyncStatus>('loading');
  const errorMessage = ref('');

  let activeSnapshotId = '';
  let currentRevision: number | null = null;
  let eventSource: EventSource | null = null;
  let collectionLoadRun = 0;
  let collectionLoadPending = false;
  let streamConnected = false;

  const loadCollection = async (snapshotId: string) => {
    collectionLoadRun += 1;
    const loadRun = collectionLoadRun;
    collectionLoadPending = true;

    try {
      const collection = await listReviewAnnotations(snapshotId);
      if (snapshotId !== activeSnapshotId || loadRun !== collectionLoadRun) {
        return;
      }

      annotations.value = collection.items;
      errorMessage.value = '';
      status.value = streamConnected ? 'ready' : 'disconnected';
    } catch (error) {
      if (snapshotId !== activeSnapshotId || loadRun !== collectionLoadRun) {
        return;
      }

      errorMessage.value = error instanceof Error ? error.message : 'Could not load agent annotations';
      status.value = 'error';
    } finally {
      if (loadRun === collectionLoadRun) {
        collectionLoadPending = false;
      }
    }
  };

  const stop = () => {
    collectionLoadRun += 1;
    collectionLoadPending = false;
    streamConnected = false;
    eventSource?.close();
    eventSource = null;
    activeSnapshotId = '';
    currentRevision = null;
    annotations.value = [];
    errorMessage.value = '';
    status.value = 'loading';
  };

  const synchronise = (snapshotId: string) => {
    if (snapshotId === activeSnapshotId && eventSource) {
      return;
    }

    stop();
    if (snapshotId === '') {
      return;
    }

    activeSnapshotId = snapshotId;
    status.value = 'loading';

    try {
      const source = new EventSource(getSubscribeReviewAnnotationEventsUrl(snapshotId));
      eventSource = source;
      source.addEventListener('open', () => {
        if (source !== eventSource || snapshotId !== activeSnapshotId) {
          return;
        }

        streamConnected = true;
      });
      source.addEventListener('annotations-changed', (event) => {
        if (source !== eventSource || snapshotId !== activeSnapshotId || !(event instanceof MessageEvent)) {
          return;
        }

        let revision: unknown;
        try {
          revision = (JSON.parse(event.data) as { revision?: unknown }).revision;
        } catch {
          return;
        }
        if (typeof revision !== 'number' || !Number.isSafeInteger(revision) || revision < 0) {
          return;
        }

        streamConnected = true;
        if (revision === currentRevision) {
          if (collectionLoadPending) {
            return;
          }
          if (status.value === 'error') {
            void loadCollection(snapshotId);
            return;
          }
          status.value = 'ready';
          return;
        }

        currentRevision = revision;
        void loadCollection(snapshotId);
      });
      source.onerror = () => {
        if (source !== eventSource || snapshotId !== activeSnapshotId) {
          return;
        }
        streamConnected = false;
        if (status.value !== 'error') {
          status.value = 'disconnected';
        }
      };
    } catch (error) {
      streamConnected = false;
      errorMessage.value = error instanceof Error ? error.message : 'Could not connect to agent annotation updates';
      status.value = 'error';
    }
  };

  const retry = () => {
    if (activeSnapshotId === '' || collectionLoadPending) {
      return;
    }

    if (!eventSource) {
      const snapshotId = activeSnapshotId;
      activeSnapshotId = '';
      synchronise(snapshotId);
      return;
    }

    errorMessage.value = '';
    status.value = 'loading';
    void loadCollection(activeSnapshotId);
  };

  return {
    annotations,
    errorMessage,
    retry,
    status,
    stop,
    synchronise
  };
};
