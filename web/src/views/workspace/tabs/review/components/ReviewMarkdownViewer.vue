<!-- NOTE: Vibecoded and unreviewed -->
<script setup lang="ts">
import MarkdownIt from 'markdown-it';
import { computed, nextTick, ref, watch } from 'vue';
import type { CommentSide, ReviewComment, ReviewFileContents } from '@/types/review';
import MermaidDiagram from './MermaidDiagram.vue';

type InlineCommentSide = Exclude<CommentSide, 'file'>;

type MarkdownBlock = {
  id: string;
  startLine: number;
  endLine: number;
  html: string;
  mermaidSource: string | null;
  isList: boolean;
};

const props = defineProps<{
  comments: ReviewComment[];
  contents: ReviewFileContents | null;
  isLoading: boolean;
}>();

const emit = defineEmits<{
  addLineComment: [payload: { side: InlineCommentSide; lineNumber: number; endLine?: number }];
  deleteComment: [commentId: string];
  toggleCommentKind: [commentId: string];
  updateCommentBody: [payload: { commentId: string; body: string }];
}>();

const viewerElement = ref<HTMLElement | null>(null);
const markdown = new MarkdownIt({
  breaks: false,
  html: false,
  linkify: true,
  typographer: false
});
const defaultLinkRenderer = markdown.renderer.rules.link_open;
markdown.renderer.rules.link_open = (tokens, index, options, environment, renderer) => {
  tokens[index].attrSet('target', '_blank');
  tokens[index].attrSet('rel', 'noopener noreferrer');

  return defaultLinkRenderer
    ? defaultLinkRenderer(tokens, index, options, environment, renderer)
    : renderer.renderToken(tokens, index, options);
};
const defaultImageRenderer = markdown.renderer.rules.image;
markdown.renderer.rules.image = (tokens, index, options, environment, renderer) => {
  tokens[index].attrSet('loading', 'lazy');
  tokens[index].attrSet('referrerpolicy', 'no-referrer');

  return defaultImageRenderer
    ? defaultImageRenderer(tokens, index, options, environment, renderer)
    : renderer.renderToken(tokens, index, options);
};

const renderMarkdownBlocks = (source: string) => {
  const environment = {};
  const tokens = markdown.parse(source, environment);
  const sourceLines = source.split('\n');
  const blocks: MarkdownBlock[] = [];
  let blockTokens: typeof tokens = [];
  let nestingDepth = 0;

  const appendBlock = () => {
    if (blockTokens.length === 0) {
      return;
    }

    const sourceMaps = blockTokens
      .map((token) => token.map)
      .filter((sourceMap): sourceMap is [number, number] => sourceMap != null);
    const startLine = sourceMaps.length > 0 ? Math.min(...sourceMaps.map((sourceMap) => sourceMap[0])) + 1 : 1;
    let endLine = sourceMaps.length > 0 ? Math.max(...sourceMaps.map((sourceMap) => sourceMap[1])) : startLine;
    while (endLine > startLine && sourceLines[endLine - 1]?.trim() === '') {
      endLine -= 1;
    }

    const firstTokenType = blockTokens[0]?.type;
    const mermaidToken = blockTokens.length === 1 ? blockTokens[0] : null;
    const mermaidSource =
      mermaidToken?.type === 'fence' && mermaidToken.info.trim().split(/\s+/)[0]?.toLowerCase() === 'mermaid'
        ? mermaidToken.content
        : null;
    blocks.push({
      id: `${startLine}:${endLine}:${blocks.length}`,
      startLine,
      endLine: Math.max(startLine, endLine),
      html: markdown.renderer.render(blockTokens, markdown.options, environment),
      mermaidSource,
      isList: firstTokenType === 'bullet_list_open' || firstTokenType === 'ordered_list_open'
    });
    blockTokens = [];
  };

  for (const token of tokens) {
    blockTokens.push(token);
    nestingDepth += token.nesting;
    if (nestingDepth === 0) {
      appendBlock();
    }
  }
  appendBlock();

  return blocks;
};

const blocks = computed(() => renderMarkdownBlocks(props.contents?.modifiedContent ?? ''));

const placeholderText = computed(() => {
  if (props.isLoading) {
    return 'Loading file contents';
  }
  if (!props.contents) {
    return 'Select a file to start reviewing';
  }
  return '';
});

const commentsForBlock = (block: MarkdownBlock) =>
  props.comments.filter((comment) => {
    if (comment.side !== 'modified' || comment.startLine == null) {
      return false;
    }

    const lineNumber = comment.startLine;
    const containingBlock = blocks.value.find(
      (candidate) => lineNumber >= candidate.startLine && lineNumber <= candidate.endLine
    );
    if (containingBlock) {
      return containingBlock.id === block.id;
    }

    const nearestBlock = blocks.value.reduce<MarkdownBlock | null>((nearest, candidate) => {
      if (!nearest) {
        return candidate;
      }

      const candidateDistance = Math.min(
        Math.abs(lineNumber - candidate.startLine),
        Math.abs(lineNumber - candidate.endLine)
      );
      const nearestDistance = Math.min(
        Math.abs(lineNumber - nearest.startLine),
        Math.abs(lineNumber - nearest.endLine)
      );
      return candidateDistance < nearestDistance ? candidate : nearest;
    }, null);

    return nearestBlock?.id === block.id;
  });

const commentKindLabel = (kind: ReviewComment['kind']) => (kind === 'question' ? 'Question' : 'Feedback');

const commentLineRange = (comment: ReviewComment) =>
  comment.endLine != null && comment.endLine !== comment.startLine
    ? `${comment.startLine}-${comment.endLine}`
    : `${comment.startLine}`;

const addBlockComment = (block: MarkdownBlock) => {
  emit('addLineComment', {
    side: 'modified',
    lineNumber: block.startLine,
    endLine: block.isList ? block.endLine : block.startLine
  });
};

const handleBlockClick = (block: MarkdownBlock, event: MouseEvent) => {
  if (!(event.target instanceof Element)) {
    return;
  }

  const clickedInteractiveContent = event.target.closest('a, button, input, textarea, select');
  const clickedExistingComment = event.target.closest('.markdown-inline-comments');
  const selection = window.getSelection();
  if (clickedInteractiveContent || clickedExistingComment || (selection && !selection.isCollapsed)) {
    return;
  }

  addBlockComment(block);
};

const updateCommentBody = (commentId: string, event: Event) => {
  const textarea = event.target as HTMLTextAreaElement;
  textarea.style.height = 'auto';
  textarea.style.height = `${textarea.scrollHeight}px`;
  emit('updateCommentBody', { commentId, body: textarea.value });
};

const commentSignature = computed(() => props.comments.map((comment) => comment.id).join('|'));

watch(commentSignature, async () => {
  await nextTick();
  const emptyTextarea = viewerElement.value?.querySelector<HTMLTextAreaElement>(
    '.markdown-inline-comment textarea[data-empty="true"]'
  );
  emptyTextarea?.focus();
});
</script>

<template>
  <section ref="viewerElement" class="markdown-review-viewer" aria-label="Rendered Markdown review">
    <div v-if="placeholderText" class="markdown-placeholder">{{ placeholderText }}</div>
    <section v-else class="markdown-document-body">
      <p v-if="blocks.length === 0" class="markdown-empty-document">This document is empty.</p>
      <article
        v-for="block in blocks"
        :key="block.id"
        class="markdown-review-block"
        tabindex="0"
        @click="handleBlockClick(block, $event)"
        @keydown.enter.self.prevent="addBlockComment(block)"
      >
        <section class="markdown-rendered-content">
          <MermaidDiagram v-if="block.mermaidSource !== null" :source="block.mermaidSource ?? ''" />
          <section v-else v-html="block.html"></section>
        </section>
        <section
          v-if="commentsForBlock(block).length > 0"
          class="markdown-inline-comments"
          aria-label="Comments on Markdown content"
        >
          <article v-for="comment in commentsForBlock(block)" :key="comment.id" class="markdown-inline-comment">
            <header>
              <span>{{ commentKindLabel(comment.kind) }} · Modified:{{ commentLineRange(comment) }}</span>
            </header>
            <textarea
              :value="comment.body"
              :data-empty="String(comment.body.length === 0)"
              spellcheck="true"
              placeholder="Write a review comment"
              @input="updateCommentBody(comment.id, $event)"
              @keydown.shift.tab.prevent.stop="emit('toggleCommentKind', comment.id)"
            ></textarea>
            <footer>
              <button type="button" :data-kind="comment.kind" @click="emit('toggleCommentKind', comment.id)">
                {{ commentKindLabel(comment.kind) }}
              </button>
              <button type="button" @click="emit('deleteComment', comment.id)">Delete</button>
            </footer>
          </article>
        </section>
      </article>
    </section>
  </section>
</template>

<style scoped>
.markdown-review-viewer {
  position: relative;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--window);
}

.markdown-document-body {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 28px clamp(22px, 5vw, 64px) 64px;
  scrollbar-width: thin;
}

.markdown-review-block {
  position: relative;
  width: min(900px, 100%);
  margin: 0 auto 2px;
  padding: 6px 12px;
  border: 1px solid transparent;
  outline: none;
  cursor: pointer;
  transition: background-color 100ms ease;
}

.markdown-review-block:hover {
  background: rgb(var(--accent-rgb) / 5%);
}

.markdown-review-block:focus-visible {
  border-color: rgb(var(--accent-rgb) / 18%);
  background: rgb(var(--accent-rgb) / 5%);
}

.markdown-review-block:focus-within {
  background: rgb(var(--accent-rgb) / 3%);
}

.markdown-rendered-content {
  color: var(--text);
  overflow-wrap: anywhere;
}

.markdown-rendered-content :deep(h1),
.markdown-rendered-content :deep(h2),
.markdown-rendered-content :deep(h3),
.markdown-rendered-content :deep(h4),
.markdown-rendered-content :deep(h5),
.markdown-rendered-content :deep(h6) {
  margin: 1.4em 0 0.6em;
  color: var(--text);
  font-weight: 500;
  line-height: 1.25;
}

.markdown-rendered-content :deep(h1) {
  padding-bottom: 0.35em;
  border-bottom: 1px solid rgb(var(--accent-rgb) / 35%);
  font-size: clamp(24px, 3vw, 32px);
}

.markdown-rendered-content :deep(h2) {
  padding-bottom: 0.3em;
  border-bottom: 1px solid rgb(var(--accent-rgb) / 22%);
  font-size: 22px;
}

.markdown-rendered-content :deep(h3) {
  font-size: 18px;
}

.markdown-rendered-content :deep(h4),
.markdown-rendered-content :deep(h5),
.markdown-rendered-content :deep(h6) {
  font-size: 14px;
}

.markdown-rendered-content :deep(p),
.markdown-rendered-content :deep(ul),
.markdown-rendered-content :deep(ol),
.markdown-rendered-content :deep(blockquote),
.markdown-rendered-content :deep(pre),
.markdown-rendered-content :deep(table) {
  margin: 0.8em 0;
}

.markdown-rendered-content :deep(p),
.markdown-rendered-content :deep(li) {
  font-size: 13px;
  line-height: 1.75;
}

.markdown-rendered-content :deep(ul),
.markdown-rendered-content :deep(ol) {
  padding-left: 2em;
}

.markdown-rendered-content :deep(li + li) {
  margin-top: 0.35em;
}

.markdown-rendered-content :deep(a) {
  color: var(--accent);
  text-decoration-thickness: 1px;
  text-underline-offset: 3px;
}

.markdown-rendered-content :deep(a:hover),
.markdown-rendered-content :deep(a:focus-visible) {
  background: rgb(var(--accent-rgb) / 10%);
}

.markdown-rendered-content :deep(blockquote) {
  padding: 2px 0 2px 16px;
  border-left: 3px solid rgb(var(--accent-rgb) / 45%);
  color: var(--muted);
}

.markdown-rendered-content :deep(blockquote > :first-child) {
  margin-top: 0;
}

.markdown-rendered-content :deep(blockquote > :last-child) {
  margin-bottom: 0;
}

.markdown-rendered-content :deep(code) {
  padding: 0.12em 0.35em;
  border: 1px solid rgb(var(--accent-rgb) / 24%);
  background: rgb(0 0 0 / 22%);
  color: var(--text);
  font: inherit;
  font-size: 0.92em;
}

.markdown-rendered-content :deep(pre) {
  padding: 14px 16px;
  border: 1px solid rgb(var(--accent-rgb) / 32%);
  background: rgb(0 0 0 / 24%);
  overflow: auto;
  line-height: 1.55;
  scrollbar-width: thin;
}

.markdown-rendered-content :deep(pre code) {
  padding: 0;
  border: 0;
  background: transparent;
  font-size: 12px;
}

.markdown-rendered-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.markdown-rendered-content :deep(th),
.markdown-rendered-content :deep(td) {
  padding: 8px 10px;
  border: 1px solid rgb(var(--accent-rgb) / 35%);
  text-align: left;
}

.markdown-rendered-content :deep(th) {
  background: rgb(var(--accent-rgb) / 8%);
  font-weight: 500;
}

.markdown-rendered-content :deep(hr) {
  height: 1px;
  margin: 1.8em 0;
  border: 0;
  background: rgb(var(--accent-rgb) / 35%);
}

.markdown-rendered-content :deep(img) {
  max-width: 100%;
  height: auto;
  border: 1px solid rgb(var(--accent-rgb) / 35%);
}

.markdown-rendered-content :deep(strong) {
  font-weight: 600;
}

.markdown-rendered-content :deep(del) {
  color: var(--muted);
}

.markdown-inline-comments {
  display: grid;
  gap: 8px;
  margin: 12px 0 16px;
}

.markdown-inline-comment {
  display: grid;
  gap: 8px;
  padding: 8px;
  border: 1px solid rgb(var(--accent-rgb) / 45%);
  background: rgb(0 0 0 / 12%);
}

.markdown-inline-comment header,
.markdown-inline-comment footer {
  display: flex;
  align-items: center;
  gap: 8px;
}

.markdown-inline-comment header {
  justify-content: flex-start;
  color: var(--muted);
  font-size: 11px;
  text-transform: uppercase;
}

.markdown-inline-comment textarea {
  width: 100%;
  min-height: 78px;
  resize: vertical;
  padding: 8px;
  border: 1px solid var(--text);
  border-radius: 0;
  outline: none;
  background: rgb(0 0 0 / 18%);
  color: var(--text);
  font: inherit;
  font-size: 12px;
  line-height: 1.45;
}

.markdown-inline-comment textarea:focus {
  border-color: var(--text);
}

.markdown-inline-comment footer {
  justify-content: flex-end;
}

.markdown-inline-comment button {
  height: 24px;
  padding: 0 8px;
  border: 1px solid var(--text);
  border-radius: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
}

.markdown-inline-comment button[data-kind='feedback'] {
  border-color: #ff6e6e;
  background: rgb(255 110 110 / 14%);
  color: #ff6e6e;
}

.markdown-inline-comment button[data-kind='question'] {
  border-color: #d29922;
  background: rgb(210 153 34 / 14%);
  color: #d29922;
}

.markdown-inline-comment button:hover,
.markdown-inline-comment button:focus-visible {
  background: rgb(var(--accent-rgb) / 10%);
}

.markdown-empty-document,
.markdown-placeholder {
  color: var(--muted);
  font-size: 12px;
  text-align: center;
}

.markdown-empty-document {
  margin: 48px 0;
}

.markdown-placeholder {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
}

@media (max-width: 900px) {
  .markdown-document-body {
    padding-inline: 16px;
  }
}
</style>
