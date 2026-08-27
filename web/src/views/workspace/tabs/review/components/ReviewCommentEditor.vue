<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue';
import type { ReviewComment } from '@/types/review';

const props = withDefaults(
  defineProps<{
    activateOnPointerdown?: boolean;
    autoResize?: boolean;
    comment: ReviewComment;
    locationLabel: string;
  }>(),
  {
    activateOnPointerdown: false,
    autoResize: false
  }
);

const emit = defineEmits<{
  deleteComment: [commentId: string];
  toggleCommentKind: [commentId: string];
  updateCommentBody: [payload: { commentId: string; body: string }];
}>();

const textarea = ref<HTMLTextAreaElement | null>(null);

const commentKindLabel = () => (props.comment.kind === 'question' ? 'Question' : 'Feedback');

const handleButtonAction = (event: Event, action: 'delete' | 'toggle-kind') => {
  event.stopPropagation();

  if (event.type === 'keydown') {
    const keyboardEvent = event as KeyboardEvent;
    if (!props.activateOnPointerdown || (keyboardEvent.key !== 'Enter' && keyboardEvent.key !== ' ')) {
      return;
    }
    event.preventDefault();
  } else if (event.type === 'pointerdown') {
    if (!props.activateOnPointerdown) {
      return;
    }
    event.preventDefault();
  } else if (props.activateOnPointerdown) {
    event.preventDefault();
    return;
  }

  if (action === 'toggle-kind') {
    emit('toggleCommentKind', props.comment.id);
  } else {
    emit('deleteComment', props.comment.id);
  }
};

const updateCommentBody = (event: Event) => {
  const target = event.currentTarget as HTMLTextAreaElement;
  if (props.autoResize) {
    target.style.height = 'auto';
    target.style.height = `${target.scrollHeight}px`;
  }

  emit('updateCommentBody', { commentId: props.comment.id, body: target.value });
};

onMounted(async () => {
  if (props.comment.body.length > 0) {
    return;
  }

  await nextTick();
  textarea.value?.focus();
});
</script>

<template>
  <article
    class="review-comment-editor"
    :data-kind="comment.kind"
    @pointerdown.stop
    @mousedown.stop
    @mouseup.stop
    @click.stop
    @dblclick.stop
    @keydown.stop
  >
    <header>
      <span>{{ commentKindLabel() }} · {{ locationLabel }}</span>
    </header>
    <textarea
      ref="textarea"
      :value="comment.body"
      :data-empty="String(comment.body.length === 0)"
      spellcheck="true"
      placeholder="Write a review comment"
      @input="updateCommentBody"
      @keydown.shift.tab.prevent="emit('toggleCommentKind', comment.id)"
    ></textarea>
    <footer>
      <button
        type="button"
        :data-kind="comment.kind"
        @pointerdown="handleButtonAction($event, 'toggle-kind')"
        @click="handleButtonAction($event, 'toggle-kind')"
        @keydown="handleButtonAction($event, 'toggle-kind')"
      >
        {{ commentKindLabel() }}
      </button>
      <button
        type="button"
        @pointerdown="handleButtonAction($event, 'delete')"
        @click="handleButtonAction($event, 'delete')"
        @keydown="handleButtonAction($event, 'delete')"
      >
        Delete
      </button>
    </footer>
  </article>
</template>

<style scoped>
.review-comment-editor {
  display: grid;
  gap: 8px;
  padding: 8px;
  border: 1px solid rgb(var(--accent-rgb) / 45%);
  background: rgb(0 0 0 / 12%);
}

.review-comment-editor header,
.review-comment-editor footer {
  display: flex;
  align-items: center;
  gap: 8px;
}

.review-comment-editor header {
  justify-content: flex-start;
  color: var(--muted);
  font-size: 11px;
  text-transform: uppercase;
}

.review-comment-editor textarea {
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

.review-comment-editor textarea:focus {
  border-color: var(--text);
}

.review-comment-editor footer {
  justify-content: flex-end;
}

.review-comment-editor button {
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

.review-comment-editor button[data-kind='feedback'] {
  border-color: #ff6e6e;
  background: rgb(255 110 110 / 14%);
  color: #ff6e6e;
}

.review-comment-editor button[data-kind='question'] {
  border-color: #d29922;
  background: rgb(210 153 34 / 14%);
  color: #d29922;
}

.review-comment-editor button:hover,
.review-comment-editor button:focus-visible {
  background: rgb(var(--accent-rgb) / 10%);
}
</style>
