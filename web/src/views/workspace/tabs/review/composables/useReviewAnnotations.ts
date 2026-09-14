import { ref } from 'vue';
import {
  deleteReviewAnnotation,
  getSubscribeReviewAnnotationEventsUrl,
  listReviewAnnotations,
  type ReviewAnnotation
} from '@/api/generated/wade';
import { WadeHTTPError } from '@/api/httpClient';
import type { AnnotationSyncStatus } from '@/types/review';

type AnnotationDeletionRecord = {
  annotation: ReviewAnnotation;
  order: number;
  lifecycleToken: number;
  deletionStartedAfterLoadSequence: number;
  lastAuthoritativePresenceSequence: number;
  reconcileAfterLoadSequence: number;
  authoritativeAbsenceConfirmed: boolean;
  status: 'pending' | 'succeeded' | 'failed';
};

export const useReviewAnnotations = () => {
  const annotations = ref<ReviewAnnotation[]>([]);
  const deletionErrors = ref<ReadonlyMap<string, string>>(new Map());
  const status = ref<AnnotationSyncStatus>('loading');
  const errorMessage = ref('');

  let activeSnapshotId = '';
  let activeLifecycleToken = 0;
  let currentRevision: number | null = null;
  let eventSource: EventSource | null = null;
  let collectionLoadRun = 0;
  let collectionRequestSequence = 0;
  let collectionLoadPending = false;
  let streamConnected = false;
  let nextAnnotationOrder = 0;
  const annotationOrder = new Map<string, number>();
  const annotationPresenceSequence = new Map<string, number>();
  const deletionRecords = new Map<string, AnnotationDeletionRecord>();

  const isCurrentLifecycle = (snapshotId: string, lifecycleToken: number) =>
    snapshotId === activeSnapshotId && lifecycleToken === activeLifecycleToken;

  const updateDeletionErrors = (annotationID: string, message?: string) => {
    const nextErrors = new Map(deletionErrors.value);
    if (message) {
      nextErrors.set(annotationID, message);
    } else {
      nextErrors.delete(annotationID);
    }
    deletionErrors.value = nextErrors;
  };

  const isNotFoundError = (error: unknown) => {
    if (error instanceof WadeHTTPError) {
      return error.status === 404;
    }

    return (
      typeof error === 'object' && error !== null && 'status' in error && (error as { status?: unknown }).status === 404
    );
  };

  const getDeletionErrorMessage = (error: unknown) => {
    if (error instanceof Error && error.message.trim() !== '') {
      return error.message;
    }

    if (typeof error === 'object' && error !== null && 'message' in error) {
      const message = (error as { message?: unknown }).message;
      if (typeof message === 'string' && message.trim() !== '') {
        return message;
      }
    }

    return 'Could not delete agent annotation';
  };

  const rememberAnnotationOrder = (items: ReviewAnnotation[]) => {
    for (const annotation of items) {
      if (annotationOrder.has(annotation.id)) {
        continue;
      }

      annotationOrder.set(annotation.id, nextAnnotationOrder);
      nextAnnotationOrder += 1;
    }
  };

  const getAnnotationOrder = (annotationID: string) => {
    const existingOrder = annotationOrder.get(annotationID);
    if (existingOrder !== undefined) {
      return existingOrder;
    }

    const order = nextAnnotationOrder;
    nextAnnotationOrder += 1;
    annotationOrder.set(annotationID, order);
    return order;
  };

  const insertAnnotationInStableOrder = (items: ReviewAnnotation[], record: AnnotationDeletionRecord) => {
    const insertionIndex = items.findIndex((annotation) => getAnnotationOrder(annotation.id) > record.order);
    if (insertionIndex < 0) {
      return [...items, record.annotation];
    }

    return [...items.slice(0, insertionIndex), record.annotation, ...items.slice(insertionIndex)];
  };

  const reconcileCollection = (items: ReviewAnnotation[], loadSequence: number) => {
    rememberAnnotationOrder(items);
    for (const annotation of items) {
      const record = deletionRecords.get(annotation.id);
      if (!record || (record.lifecycleToken === activeLifecycleToken && record.status === 'failed')) {
        annotationPresenceSequence.set(annotation.id, loadSequence);
      }
    }

    const itemIDs = new Set(items.map((annotation) => annotation.id));
    let reconciledItems = items.filter((annotation) => {
      const record = deletionRecords.get(annotation.id);
      if (!record || record.lifecycleToken !== activeLifecycleToken) {
        return true;
      }

      return record.status !== 'pending' && record.status !== 'succeeded';
    });

    for (const [annotationID, record] of deletionRecords) {
      if (record.lifecycleToken !== activeLifecycleToken) {
        continue;
      }

      const hasAnnotation = itemIDs.has(annotationID);
      const absenceConfirmedByCollection =
        !hasAnnotation &&
        (loadSequence >= record.deletionStartedAfterLoadSequence ||
          loadSequence > record.lastAuthoritativePresenceSequence);
      if (record.status === 'pending') {
        if (absenceConfirmedByCollection) {
          record.authoritativeAbsenceConfirmed = true;
        }
        continue;
      }

      if (record.status === 'succeeded') {
        if (absenceConfirmedByCollection) {
          deletionRecords.delete(annotationID);
          continue;
        }

        if (!hasAnnotation && loadSequence >= record.reconcileAfterLoadSequence) {
          deletionRecords.delete(annotationID);
        }
        continue;
      }

      if (record.status === 'failed') {
        if (absenceConfirmedByCollection) {
          deletionRecords.delete(annotationID);
          updateDeletionErrors(annotationID);
          continue;
        }

        if (loadSequence < record.reconcileAfterLoadSequence) {
          if (!hasAnnotation) {
            reconciledItems = insertAnnotationInStableOrder(reconciledItems, record);
          }
          continue;
        }

        deletionRecords.delete(annotationID);
        if (!hasAnnotation) {
          updateDeletionErrors(annotationID);
        }
      }
    }

    return reconciledItems;
  };

  const completeSuccessfulDeletion = (annotationID: string, record: AnnotationDeletionRecord) => {
    record.status = 'succeeded';
    record.reconcileAfterLoadSequence = collectionRequestSequence + 1;
    if (record.authoritativeAbsenceConfirmed && !collectionLoadPending) {
      deletionRecords.delete(annotationID);
    }
  };

  const loadCollection = async (snapshotId: string, lifecycleToken: number) => {
    collectionLoadRun += 1;
    const loadRun = collectionLoadRun;
    const loadSequence = ++collectionRequestSequence;
    collectionLoadPending = true;

    try {
      const collection = await listReviewAnnotations(snapshotId);
      if (!isCurrentLifecycle(snapshotId, lifecycleToken) || loadRun !== collectionLoadRun) {
        return;
      }

      annotations.value = reconcileCollection(collection.items, loadSequence);
      errorMessage.value = '';
      status.value = streamConnected ? 'ready' : 'disconnected';
    } catch (error) {
      if (!isCurrentLifecycle(snapshotId, lifecycleToken) || loadRun !== collectionLoadRun) {
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
    activeLifecycleToken += 1;
    collectionLoadRun += 1;
    collectionLoadPending = false;
    streamConnected = false;
    eventSource?.close();
    eventSource = null;
    activeSnapshotId = '';
    currentRevision = null;
    annotationOrder.clear();
    annotationPresenceSequence.clear();
    nextAnnotationOrder = 0;
    deletionRecords.clear();
    annotations.value = [];
    deletionErrors.value = new Map();
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
    const lifecycleToken = activeLifecycleToken;
    status.value = 'loading';

    try {
      const source = new EventSource(getSubscribeReviewAnnotationEventsUrl(snapshotId));
      eventSource = source;
      source.addEventListener('open', () => {
        if (source !== eventSource || !isCurrentLifecycle(snapshotId, lifecycleToken)) {
          return;
        }

        streamConnected = true;
      });
      source.addEventListener('annotations-changed', (event) => {
        if (
          source !== eventSource ||
          !isCurrentLifecycle(snapshotId, lifecycleToken) ||
          !(event instanceof MessageEvent)
        ) {
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
            void loadCollection(snapshotId, lifecycleToken);
            return;
          }
          status.value = 'ready';
          return;
        }

        currentRevision = revision;
        void loadCollection(snapshotId, lifecycleToken);
      });
      source.onerror = () => {
        if (source !== eventSource || !isCurrentLifecycle(snapshotId, lifecycleToken)) {
          return;
        }
        streamConnected = false;
        if (status.value !== 'error') {
          status.value = 'disconnected';
        }
      };
    } catch (error) {
      if (!isCurrentLifecycle(snapshotId, lifecycleToken)) {
        return;
      }

      streamConnected = false;
      errorMessage.value = error instanceof Error ? error.message : 'Could not connect to agent annotation updates';
      status.value = 'error';
    }
  };

  const deleteAnnotation = async (annotationID: string) => {
    const snapshotId = activeSnapshotId;
    const lifecycleToken = activeLifecycleToken;
    if (!snapshotId) {
      return;
    }

    const annotationIndex = annotations.value.findIndex(
      (annotation) => annotation.id === annotationID && annotation.snapshotId === snapshotId
    );
    if (annotationIndex < 0) {
      return;
    }

    const existingDeletion = deletionRecords.get(annotationID);
    if (existingDeletion?.status === 'pending' || existingDeletion?.status === 'succeeded') {
      return;
    }

    const record: AnnotationDeletionRecord = {
      annotation: annotations.value[annotationIndex],
      order: getAnnotationOrder(annotationID),
      lifecycleToken,
      deletionStartedAfterLoadSequence: collectionRequestSequence + 1,
      lastAuthoritativePresenceSequence: annotationPresenceSequence.get(annotationID) ?? collectionRequestSequence,
      reconcileAfterLoadSequence: Number.POSITIVE_INFINITY,
      authoritativeAbsenceConfirmed: false,
      status: 'pending'
    };
    deletionRecords.set(annotationID, record);
    updateDeletionErrors(annotationID);
    annotations.value = annotations.value.filter((annotation) => annotation.id !== annotationID);

    try {
      await deleteReviewAnnotation(snapshotId, annotationID);
    } catch (error) {
      if (!isCurrentLifecycle(snapshotId, lifecycleToken) || deletionRecords.get(annotationID) !== record) {
        return;
      }

      if (isNotFoundError(error) || record.authoritativeAbsenceConfirmed) {
        completeSuccessfulDeletion(annotationID, record);
        return;
      }

      record.status = 'failed';
      record.reconcileAfterLoadSequence = collectionRequestSequence + 1;
      annotations.value = insertAnnotationInStableOrder(annotations.value, record);
      updateDeletionErrors(annotationID, getDeletionErrorMessage(error));
      return;
    }

    if (!isCurrentLifecycle(snapshotId, lifecycleToken) || deletionRecords.get(annotationID) !== record) {
      return;
    }

    completeSuccessfulDeletion(annotationID, record);
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
    void loadCollection(activeSnapshotId, activeLifecycleToken);
  };

  return {
    annotations,
    deletionErrors,
    deleteAnnotation,
    errorMessage,
    retry,
    status,
    stop,
    synchronise
  };
};
