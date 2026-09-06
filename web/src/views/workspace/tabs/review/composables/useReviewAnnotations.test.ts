import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReviewAnnotation } from '@/api/generated/wade';
import { useReviewAnnotations } from './useReviewAnnotations';

const api = vi.hoisted(() => ({
  listReviewAnnotations: vi.fn()
}));

vi.mock('@/api/generated/wade', () => ({
  getSubscribeReviewAnnotationEventsUrl: (snapshotId: string) => `/snapshots/${snapshotId}/events`,
  listReviewAnnotations: api.listReviewAnnotations
}));

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  static rejectNextConnection = false;

  readonly url: string;
  closed = false;
  onerror: ((event: Event) => void) | null = null;
  private listeners = new Map<string, EventListener[]>();

  constructor(url: string | URL) {
    if (FakeEventSource.rejectNextConnection) {
      FakeEventSource.rejectNextConnection = false;
      throw new Error('connection failed');
    }

    this.url = String(url);
    FakeEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    if (typeof listener !== 'function') {
      return;
    }
    const listeners = this.listeners.get(type) ?? [];
    listeners.push(listener);
    this.listeners.set(type, listeners);
  }

  close() {
    this.closed = true;
  }

  emitRevision(revision: number) {
    const event = new MessageEvent('annotations-changed', { data: JSON.stringify({ revision }) });
    for (const listener of this.listeners.get('annotations-changed') ?? []) {
      listener(event);
    }
  }

  fail() {
    this.onerror?.(new Event('error'));
  }
}

const annotation = (id: string): ReviewAnnotation => ({
  author: null,
  createdAt: '2026-01-01T00:00:00Z',
  endLine: 1,
  fileId: 'file-1',
  id,
  rationale: null,
  scope: 'working-tree',
  side: 'modified',
  snapshotId: 'snapshot-1',
  startLine: 1,
  summary: `Annotation ${id}`
});

describe('useReviewAnnotations', () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    FakeEventSource.rejectNextConnection = false;
    api.listReviewAnnotations.mockReset();
    vi.stubGlobal('EventSource', FakeEventSource);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('loads the authoritative collection for initial and changed revisions without duplicating reads', async () => {
    api.listReviewAnnotations
      .mockResolvedValueOnce({ items: [annotation('one')] })
      .mockResolvedValueOnce({ items: [annotation('one'), annotation('two')] });
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const source = FakeEventSource.instances[0];
    source.emitRevision(0);

    await vi.waitFor(() => expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['one']));
    expect(synchronisation.status.value).toBe('ready');

    source.emitRevision(1);
    await vi.waitFor(() => expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['one', 'two']));
    source.emitRevision(1);

    expect(api.listReviewAnnotations).toHaveBeenCalledTimes(2);
    expect(api.listReviewAnnotations).toHaveBeenLastCalledWith('snapshot-1');
  });

  it('closes the previous stream and ignores events from an old snapshot', async () => {
    api.listReviewAnnotations.mockResolvedValue({ items: [annotation('new')] });
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const oldSource = FakeEventSource.instances[0];
    synchronisation.synchronise('snapshot-2');
    const newSource = FakeEventSource.instances[1];

    expect(oldSource.closed).toBe(true);
    oldSource.emitRevision(1);
    newSource.emitRevision(0);

    await vi.waitFor(() => expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['new']));
    expect(api.listReviewAnnotations).toHaveBeenCalledTimes(1);
    expect(api.listReviewAnnotations).toHaveBeenCalledWith('snapshot-2');
  });

  it('ignores a late collection response from the previous snapshot', async () => {
    let resolveOldCollection: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    api.listReviewAnnotations
      .mockImplementationOnce(
        () =>
          new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
            resolveOldCollection = resolve;
          })
      )
      .mockResolvedValueOnce({ items: [annotation('new')] });
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    FakeEventSource.instances[0].emitRevision(0);
    synchronisation.synchronise('snapshot-2');
    FakeEventSource.instances[1].emitRevision(0);

    await vi.waitFor(() => expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['new']));
    resolveOldCollection?.({ items: [annotation('old')] });
    await Promise.resolve();

    expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['new']);
  });

  it('keeps the newer same-snapshot revision when an older pending request settles later', async () => {
    let resolveOlder: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    let resolveNewer: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    api.listReviewAnnotations
      .mockImplementationOnce(
        () =>
          new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
            resolveOlder = resolve;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
            resolveNewer = resolve;
          })
      );
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const source = FakeEventSource.instances[0];
    source.emitRevision(0);
    source.emitRevision(1);

    resolveNewer?.({ items: [annotation('newer')] });
    await vi.waitFor(() => expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['newer']));

    resolveOlder?.({ items: [annotation('older')] });
    await Promise.resolve();
    await Promise.resolve();

    expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['newer']);
    expect(synchronisation.status.value).toBe('ready');
    expect(synchronisation.errorMessage.value).toBe('');
  });

  it('keeps the newer same-snapshot revision when an older pending request rejects later', async () => {
    let rejectOlder: ((reason?: unknown) => void) | undefined;
    let resolveNewer: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    let olderRequest: Promise<{ items: ReviewAnnotation[] }> | undefined;
    api.listReviewAnnotations
      .mockImplementationOnce(
        () =>
          (olderRequest = new Promise<{ items: ReviewAnnotation[] }>((_, reject) => {
            rejectOlder = reject;
          }))
      )
      .mockImplementationOnce(
        () =>
          new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
            resolveNewer = resolve;
          })
      );
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const source = FakeEventSource.instances[0];
    source.emitRevision(0);
    source.emitRevision(1);
    expect(api.listReviewAnnotations).toHaveBeenCalledTimes(2);

    const olderRequestSettled = new Promise<void>((resolve) => {
      olderRequest?.catch(() => resolve());
    });
    resolveNewer?.({ items: [annotation('newer')] });
    await vi.waitFor(() => {
      expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['newer']);
      expect(synchronisation.status.value).toBe('ready');
    });

    rejectOlder?.(new Error('stale failure'));
    await olderRequestSettled;

    expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['newer']);
    expect(synchronisation.status.value).toBe('ready');
    expect(synchronisation.errorMessage.value).toBe('');
  });

  it('ignores stale events and responses when reopening the same snapshot while loading', async () => {
    let resolveOldRequest: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    let resolveReopenedRequest: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    api.listReviewAnnotations
      .mockImplementationOnce(
        () =>
          new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
            resolveOldRequest = resolve;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
            resolveReopenedRequest = resolve;
          })
      );
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const oldSource = FakeEventSource.instances[0];
    oldSource.emitRevision(0);
    synchronisation.stop();
    synchronisation.synchronise('snapshot-1');
    const reopenedSource = FakeEventSource.instances[1];
    reopenedSource.emitRevision(0);
    oldSource.emitRevision(1);

    expect(api.listReviewAnnotations).toHaveBeenCalledTimes(2);
    resolveReopenedRequest?.({ items: [annotation('reopened')] });
    await vi.waitFor(() => expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['reopened']));

    resolveOldRequest?.({ items: [annotation('stale')] });
    await Promise.resolve();
    await Promise.resolve();

    expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['reopened']);
    expect(synchronisation.status.value).toBe('ready');
  });

  it('keeps a pending collection disconnected after stream failure and repairs it on reconnect', async () => {
    let resolveCollection: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    api.listReviewAnnotations.mockImplementationOnce(
      () =>
        new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
          resolveCollection = resolve;
        })
    );
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const source = FakeEventSource.instances[0];
    source.emitRevision(0);
    source.fail();
    expect(synchronisation.status.value).toBe('disconnected');

    resolveCollection?.({ items: [annotation('one')] });
    await vi.waitFor(() => expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['one']));
    expect(synchronisation.status.value).toBe('disconnected');

    source.emitRevision(0);
    expect(synchronisation.status.value).toBe('ready');
    expect(api.listReviewAnnotations).toHaveBeenCalledTimes(1);
  });

  it('does not start a second collection read for duplicate revisions while the first is pending', async () => {
    let resolveCollection: ((value: { items: ReviewAnnotation[] }) => void) | undefined;
    api.listReviewAnnotations.mockImplementationOnce(
      () =>
        new Promise<{ items: ReviewAnnotation[] }>((resolve) => {
          resolveCollection = resolve;
        })
    );
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const source = FakeEventSource.instances[0];
    source.emitRevision(0);
    source.emitRevision(0);

    expect(api.listReviewAnnotations).toHaveBeenCalledTimes(1);
    expect(synchronisation.status.value).toBe('loading');

    resolveCollection?.({ items: [annotation('one')] });
    await vi.waitFor(() => expect(synchronisation.status.value).toBe('ready'));
    expect(api.listReviewAnnotations).toHaveBeenCalledTimes(1);
  });

  it('retains loaded annotations while the event stream reconnects', async () => {
    api.listReviewAnnotations.mockResolvedValue({ items: [annotation('one')] });
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    const source = FakeEventSource.instances[0];
    source.emitRevision(0);
    await vi.waitFor(() => expect(synchronisation.status.value).toBe('ready'));

    source.fail();

    expect(synchronisation.status.value).toBe('disconnected');
    expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['one']);
    source.emitRevision(0);
    expect(synchronisation.status.value).toBe('ready');
  });

  it('reconnects the event stream when connection setup is retried', async () => {
    api.listReviewAnnotations.mockResolvedValue({ items: [annotation('reconnected')] });
    FakeEventSource.rejectNextConnection = true;
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    expect(synchronisation.status.value).toBe('error');
    expect(synchronisation.errorMessage.value).toBe('connection failed');

    synchronisation.retry();
    expect(FakeEventSource.instances).toHaveLength(1);
    FakeEventSource.instances[0].emitRevision(0);

    await vi.waitFor(() => expect(synchronisation.status.value).toBe('ready'));
    expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['reconnected']);
  });

  it('allows a failed collection read to be retried', async () => {
    api.listReviewAnnotations
      .mockRejectedValueOnce(new Error('temporary failure'))
      .mockResolvedValueOnce({ items: [annotation('recovered')] });
    const synchronisation = useReviewAnnotations();

    synchronisation.synchronise('snapshot-1');
    FakeEventSource.instances[0].emitRevision(0);
    await vi.waitFor(() => expect(synchronisation.status.value).toBe('error'));
    expect(synchronisation.errorMessage.value).toBe('temporary failure');

    synchronisation.retry();

    await vi.waitFor(() => expect(synchronisation.status.value).toBe('ready'));
    expect(synchronisation.annotations.value.map(({ id }) => id)).toEqual(['recovered']);
  });
});
