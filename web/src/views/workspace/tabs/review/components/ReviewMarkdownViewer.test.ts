import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import type { ReviewAnnotation } from '@/api/generated/wade';
import type { ReviewComment } from '@/types/review';
import ReviewMarkdownViewer from './ReviewMarkdownViewer.vue';

const annotation = (id: string, side: ReviewAnnotation['side'], summary: string, startLine = 3): ReviewAnnotation => ({
  author: id === 'first' ? 'Pi' : null,
  createdAt: '2026-01-01T00:00:00Z',
  endLine: startLine + 1,
  fileId: 'file-1',
  id,
  rationale: `Rationale for ${id}`,
  scope: 'working-tree',
  side,
  snapshotId: 'snapshot-1',
  startLine,
  summary
});

const comment: ReviewComment = {
  body: 'Human feedback',
  endLine: 3,
  fileId: 'file-1',
  id: 'comment-1',
  kind: 'feedback',
  scope: 'working-tree',
  side: 'modified',
  startLine: 3
};

describe('ReviewMarkdownViewer agent annotations', () => {
  it('renders modified annotations with editable human comments and omits original annotations', async () => {
    const wrapper = mount(ReviewMarkdownViewer, {
      props: {
        annotations: [
          annotation('first', 'modified', '<img src=x onerror=alert(1)>'),
          annotation('second', 'modified', 'Second note'),
          annotation('original', 'original', 'Original note', 1)
        ],
        comments: [comment],
        contents: {
          originalContent: '# Before\n',
          modifiedContent: '# Heading\n\nParagraph\ncontinued\n'
        },
        isLoading: false
      },
      global: {
        stubs: { MermaidDiagram: true }
      }
    });

    const cards = wrapper.findAll('.review-agent-annotation');
    expect(cards).toHaveLength(2);
    expect(cards[0].text()).toContain('Pi');
    expect(cards[0].text()).toContain('Modified:3-4');
    expect(cards[0].text()).toContain('<img src=x onerror=alert(1)>');
    expect(cards[1].text()).toContain('Agent note');
    expect(cards[0].find('textarea').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('Original note');
    expect(wrapper.find('.review-agent-annotation img').exists()).toBe(false);
    expect(wrapper.findAll('.review-comment-editor')).toHaveLength(1);
    expect(wrapper.find('textarea').element.value).toBe('Human feedback');

    await cards[0].trigger('click');
    expect(wrapper.emitted('addLineComment')).toBeUndefined();

    await wrapper.setProps({ annotations: [] });
    expect(wrapper.findAll('.review-agent-annotation')).toHaveLength(0);
    expect(wrapper.findAll('.review-comment-editor')).toHaveLength(1);
  });
});
