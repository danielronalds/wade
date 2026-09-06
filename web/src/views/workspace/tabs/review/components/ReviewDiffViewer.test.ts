import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReviewAnnotation } from '@/api/generated/wade';
import type { ReviewComment } from '@/types/review';
import ReviewDiffViewer from './ReviewDiffViewer.vue';

type RecordedZone = {
  afterLineNumber: number;
  heightInPx: number;
  domNode: HTMLElement;
};

type FakeViewZoneAccessor = {
  addZone: (zone: RecordedZone) => string;
  layoutZone: (id: string) => void;
  removeZone: (id: string) => void;
};

class FakeCodeEditor {
  zones = new Map<string, RecordedZone>();
  decorations: Array<Record<string, unknown>> = [];
  layoutZoneCalls: string[] = [];
  private nextZoneId = 0;

  changeViewZones(callback: (accessor: FakeViewZoneAccessor) => void) {
    callback({
      addZone: (zone) => {
        this.nextZoneId += 1;
        const id = `zone-${this.nextZoneId}`;
        this.zones.set(id, zone);
        document.body.append(zone.domNode);
        return id;
      },
      layoutZone: (id) => {
        this.layoutZoneCalls.push(id);
      },
      removeZone: (id) => {
        const zone = this.zones.get(id);
        zone?.domNode.remove();
        this.zones.delete(id);
      }
    });
  }

  deltaDecorations(_oldDecorations: string[], decorations: Array<Record<string, unknown>>) {
    this.decorations = decorations;
    return decorations.map((_, index) => `decoration-${index}`);
  }

  getScrollLeft() {
    return 0;
  }

  getScrollTop() {
    return 0;
  }

  onMouseDown() {
    return { dispose: () => undefined };
  }

  setScrollLeft() {}

  setScrollTop() {}

  updateOptions() {}
}

class FakeResizeObserver {
  static instances: FakeResizeObserver[] = [];

  readonly observedElements = new Set<Element>();

  constructor(private readonly callback: ResizeObserverCallback) {
    FakeResizeObserver.instances.push(this);
  }

  observe(element: Element) {
    this.observedElements.add(element);
  }

  unobserve(element: Element) {
    this.observedElements.delete(element);
  }

  disconnect() {
    this.observedElements.clear();
  }

  trigger() {
    this.callback([], this as unknown as ResizeObserver);
  }
}

class FakeRange {
  constructor(
    readonly startLineNumber: number,
    readonly startColumn: number,
    readonly endLineNumber: number,
    readonly endColumn: number
  ) {}
}

const annotation = (id: string, summary: string, startLine = 1, endLine = 2): ReviewAnnotation => ({
  author: null,
  createdAt: '2026-01-01T00:00:00Z',
  endLine,
  fileId: 'file-1',
  id,
  rationale: `Rationale ${id}`,
  scope: 'working-tree',
  side: 'original',
  snapshotId: 'snapshot-1',
  startLine,
  summary
});

const comment: ReviewComment = {
  body: 'Human feedback',
  endLine: 1,
  fileId: 'file-1',
  id: 'comment-1',
  kind: 'feedback',
  scope: 'working-tree',
  side: 'original',
  startLine: 1
};

describe('ReviewDiffViewer agent annotations', () => {
  const originalEditor = new FakeCodeEditor();
  const modifiedEditor = new FakeCodeEditor();

  beforeEach(() => {
    originalEditor.zones.clear();
    originalEditor.decorations = [];
    originalEditor.layoutZoneCalls = [];
    modifiedEditor.zones.clear();
    modifiedEditor.decorations = [];
    modifiedEditor.layoutZoneCalls = [];
    FakeResizeObserver.instances = [];

    let animationFrameId = 0;
    vi.stubGlobal('ResizeObserver', FakeResizeObserver);
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      animationFrameId += 1;
      queueMicrotask(() => callback(0));
      return animationFrameId;
    });
    vi.stubGlobal('cancelAnimationFrame', () => undefined);

    window.monaco = {
      Range: FakeRange,
      editor: {
        MouseTargetType: { GUTTER_GLYPH_MARGIN: 2, GUTTER_LINE_NUMBERS: 3 },
        createDiffEditor: () => ({
          changeViewZones: () => undefined,
          deltaDecorations: () => [],
          dispose: () => undefined,
          getModifiedEditor: () => modifiedEditor,
          getOriginalEditor: () => originalEditor,
          layout: () => undefined,
          setModel: () => undefined,
          updateOptions: () => undefined
        }),
        createModel: () => ({ dispose: () => undefined }),
        defineTheme: () => undefined,
        setTheme: () => undefined
      }
    };
  });

  afterEach(() => {
    delete window.monaco;
    vi.unstubAllGlobals();
    vi.clearAllTimers();
  });

  it('renders agent and human content together, preserves order and lays out the inclusive range', async () => {
    const wrapper = mount(ReviewDiffViewer, {
      props: {
        annotations: [annotation('first', '<script>alert(1)</script>'), annotation('second', 'Second note')],
        comments: [comment],
        contents: { originalContent: 'before\nsecond\n', modifiedContent: 'after\nsecond\n' },
        filePath: 'src/app.ts',
        hideUnchanged: true,
        isDiff: true,
        isLoading: false,
        renderSideBySide: true,
        scrollKey: 'working-tree:file-1',
        wrapLines: true
      }
    });
    await flushPromises();

    expect(originalEditor.zones.size).toBe(1);
    const zone = [...originalEditor.zones.values()][0];
    const contentElement = zone.domNode.querySelector('.review-inline-zone-content');
    if (!contentElement) {
      throw new Error('Expected a natural-size view-zone content wrapper');
    }
    const text = contentElement.textContent ?? '';
    expect(text.indexOf('<script>alert(1)</script>')).toBeLessThan(text.indexOf('Second note'));
    expect(contentElement.querySelector('script')).toBeNull();
    expect(contentElement.querySelectorAll('.review-agent-annotation')).toHaveLength(2);
    expect(contentElement.querySelectorAll('.review-comment-editor')).toHaveLength(1);
    const textarea = zone.domNode.querySelector('textarea');
    expect(textarea?.value).toBe('Human feedback');
    if (!textarea) {
      throw new Error('Expected an editable human comment');
    }
    textarea.value = 'Updated feedback';
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
    await flushPromises();
    expect(wrapper.emitted('updateCommentBody')).toEqual([[{ body: 'Updated feedback', commentId: 'comment-1' }]]);

    originalEditor.layoutZoneCalls = [];
    Object.defineProperty(zone.domNode, 'scrollHeight', { configurable: true, value: 1316 });
    let contentHeight = 420;
    vi.spyOn(contentElement, 'getBoundingClientRect').mockImplementation(() => ({ height: contentHeight }) as DOMRect);
    const viewZoneObserver = FakeResizeObserver.instances.find((observer) =>
      observer.observedElements.has(contentElement)
    );
    if (!viewZoneObserver) {
      throw new Error('Expected the natural-size wrapper to be observed');
    }
    viewZoneObserver.trigger();
    await flushPromises();
    expect(zone.heightInPx).toBe(420);
    expect(originalEditor.layoutZoneCalls).toHaveLength(1);

    const annotationDecorations = originalEditor.decorations.filter((decoration) =>
      String((decoration.options as { className?: string }).className).includes('review-agent-line')
    );
    expect(annotationDecorations).toHaveLength(2);
    const range = annotationDecorations[0].range as FakeRange;
    expect([range.startLineNumber, range.endLineNumber]).toEqual([1, 2]);

    contentHeight = 149;
    await wrapper.setProps({ annotations: [] });
    await flushPromises();
    expect(zone.heightInPx).toBe(149);

    contentHeight = 240;
    viewZoneObserver.trigger();
    await flushPromises();
    expect(zone.heightInPx).toBe(240);
    expect(originalEditor.layoutZoneCalls).toHaveLength(3);

    wrapper.unmount();
  });

  it('retains a measured human-only zone height while Monaco hides and reveals it', async () => {
    const wrapper = mount(ReviewDiffViewer, {
      props: {
        annotations: [],
        comments: [comment],
        contents: { originalContent: 'before\nsecond\n', modifiedContent: 'after\nsecond\n' },
        filePath: 'src/app.ts',
        hideUnchanged: true,
        isDiff: true,
        isLoading: false,
        renderSideBySide: true,
        scrollKey: 'working-tree:file-1',
        wrapLines: true
      }
    });
    await flushPromises();

    const zone = [...originalEditor.zones.values()][0];
    const contentElement = zone.domNode.querySelector('.review-inline-zone-content');
    if (!contentElement) {
      throw new Error('Expected a human comment view-zone content wrapper');
    }

    let contentHeight = 420;
    vi.spyOn(contentElement, 'getBoundingClientRect').mockImplementation(() => ({ height: contentHeight }) as DOMRect);
    const viewZoneObserver = FakeResizeObserver.instances.find((observer) =>
      observer.observedElements.has(contentElement)
    );
    if (!viewZoneObserver) {
      throw new Error('Expected the human comment wrapper to be observed');
    }

    viewZoneObserver.trigger();
    await flushPromises();
    expect(zone.heightInPx).toBe(420);
    const layoutCallsAfterVisibleMeasurement = originalEditor.layoutZoneCalls.length;

    zone.domNode.style.display = 'none';
    contentHeight = 0;
    viewZoneObserver.trigger();
    await flushPromises();
    expect(zone.heightInPx).toBe(420);
    expect(originalEditor.layoutZoneCalls).toHaveLength(layoutCallsAfterVisibleMeasurement);

    zone.domNode.style.display = '';
    contentHeight = 240;
    viewZoneObserver.trigger();
    await flushPromises();
    expect(zone.heightInPx).toBe(240);
    expect(originalEditor.layoutZoneCalls).toHaveLength(layoutCallsAfterVisibleMeasurement + 1);

    wrapper.unmount();
  });

  it('retains human editor identity, focus, selection and input while annotation zones reconcile', async () => {
    const wrapper = mount(ReviewDiffViewer, {
      props: {
        annotations: [annotation('first', 'First note')],
        comments: [comment],
        contents: { originalContent: 'before\nsecond\nthird\n', modifiedContent: 'after\nsecond\nthird\n' },
        filePath: 'src/app.ts',
        hideUnchanged: true,
        isDiff: true,
        isLoading: false,
        renderSideBySide: true,
        scrollKey: 'working-tree:file-1',
        wrapLines: true
      }
    });
    await flushPromises();

    const retainedZone = [...originalEditor.zones.values()].find((zone) => zone.afterLineNumber === 1);
    const retainedTextarea = retainedZone?.domNode.querySelector('textarea');
    if (!retainedZone || !retainedTextarea) {
      throw new Error('Expected the original human comment editor');
    }
    const zoneAt = (lineNumber: number) =>
      [...originalEditor.zones.values()].find((candidate) => candidate.afterLineNumber === lineNumber);

    retainedTextarea.focus();
    retainedTextarea.setSelectionRange(2, 7);

    await wrapper.setProps({
      annotations: [
        annotation('first', 'First note'),
        annotation('same-anchor', 'Same anchor', 1, 1),
        annotation('other-anchor', 'Other anchor', 3, 3)
      ]
    });
    await flushPromises();

    expect(zoneAt(1)).toBe(retainedZone);
    expect(zoneAt(1)?.domNode).toBe(retainedZone.domNode);
    expect(retainedZone.domNode.querySelector('textarea')).toBe(retainedTextarea);
    expect(document.activeElement).toBe(retainedTextarea);
    expect([retainedTextarea.selectionStart, retainedTextarea.selectionEnd]).toEqual([2, 7]);

    retainedTextarea.value = 'continued human editing';
    retainedTextarea.dispatchEvent(new Event('input', { bubbles: true }));
    await flushPromises();
    expect(wrapper.emitted('updateCommentBody')?.at(-1)).toEqual([
      { body: 'continued human editing', commentId: 'comment-1' }
    ]);
    retainedTextarea.focus();
    retainedTextarea.setSelectionRange(2, 7);

    await wrapper.setProps({ annotations: [annotation('other-anchor', 'Other anchor', 3, 3)] });
    await flushPromises();
    expect(zoneAt(1)).toBe(retainedZone);
    expect(retainedZone.domNode.querySelector('textarea')).toBe(retainedTextarea);

    const otherZone = zoneAt(3);
    const otherContentElement = otherZone?.domNode.querySelector('.review-inline-zone-content');
    if (!otherZone || !otherContentElement) {
      throw new Error('Expected the other annotation zone');
    }

    await wrapper.setProps({ annotations: [] });
    await flushPromises();
    expect(zoneAt(1)).toBe(retainedZone);
    expect(zoneAt(3)).toBeUndefined();
    expect(
      FakeResizeObserver.instances.find((observer) => observer.observedElements.has(otherContentElement))
    ).toBeUndefined();
    expect(retainedZone.domNode.querySelector('textarea')).toBe(retainedTextarea);
    expect(document.activeElement).toBe(retainedTextarea);
    expect([retainedTextarea.selectionStart, retainedTextarea.selectionEnd]).toEqual([2, 7]);

    wrapper.unmount();
  });

  it('exposes original-side annotation notes outside the hidden Monaco subtree', async () => {
    const wrapper = mount(ReviewDiffViewer, {
      props: {
        annotations: [
          annotation('original', 'Original rationale', 3, 4),
          { ...annotation('modified', 'Modified rationale', 7, 7), side: 'modified' }
        ],
        comments: [],
        contents: { originalContent: 'before\nsecond\n', modifiedContent: 'after\nsecond\n' },
        filePath: 'src/app.ts',
        hideUnchanged: true,
        isDiff: true,
        isLoading: false,
        renderSideBySide: true,
        scrollKey: 'working-tree:file-1',
        wrapLines: true
      }
    });
    await flushPromises();

    const accessibleRegion = wrapper.find('.review-diff-accessible-annotations');
    expect(accessibleRegion.attributes('aria-label')).toBe('Agent annotations');
    expect(accessibleRegion.attributes('aria-hidden')).toBeUndefined();
    expect(accessibleRegion.findAll('.review-agent-annotation')).toHaveLength(2);
    expect(accessibleRegion.text()).toContain('Original:3-4');
    expect(accessibleRegion.text()).toContain('Modified:7');

    await wrapper.setProps({ annotations: [] });
    await flushPromises();
    expect(accessibleRegion.findAll('.review-agent-annotation')).toHaveLength(0);

    wrapper.unmount();
  });
});
