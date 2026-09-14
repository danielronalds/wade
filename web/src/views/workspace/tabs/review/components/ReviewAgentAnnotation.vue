<script setup lang="ts">
import { X } from '@lucide/vue';
import { ref } from 'vue';
import type { ReviewAnnotation } from '@/api/generated/wade';
import type { AnnotationDeletionDisplay } from '@/types/review';

const props = withDefaults(
  defineProps<{
    annotation: ReviewAnnotation;
    locationLabel: string;
    deletion?: AnnotationDeletionDisplay;
    activateOnPointerdown?: boolean;
  }>(),
  {
    deletion: () => ({ status: 'available' }),
    activateOnPointerdown: false
  }
);

const emit = defineEmits<{
  deleteAnnotation: [annotationID: string];
}>();

const suppressPointerClick = ref(false);

const isPrimaryPointerdown = (event: Event) => (event as PointerEvent).button === 0;

const handleDelete = (event: Event) => {
  event.stopPropagation();

  if (event.type === 'pointerdown') {
    if (!props.activateOnPointerdown || !isPrimaryPointerdown(event)) {
      return;
    }

    event.preventDefault();
    suppressPointerClick.value = true;
    emit('deleteAnnotation', props.annotation.id);
    return;
  }

  if (suppressPointerClick.value) {
    suppressPointerClick.value = false;
    event.preventDefault();
    return;
  }

  emit('deleteAnnotation', props.annotation.id);
};
</script>

<template>
  <article
    class="review-agent-annotation"
    @pointerdown.stop
    @mousedown.stop
    @mouseup.stop
    @click.stop
    @dblclick.stop
    @keydown.stop
  >
    <header>
      <span>{{ annotation.author?.trim() || 'Agent note' }}</span>
      <span>{{ locationLabel }}</span>
    </header>
    <button
      type="button"
      class="review-agent-annotation-delete"
      aria-label="Delete agent annotation"
      title="Delete agent annotation"
      @pointerdown="handleDelete"
      @click="handleDelete"
    >
      <X :size="14" :stroke-width="2" aria-hidden="true" />
    </button>
    <p class="review-agent-annotation-summary">{{ annotation.summary }}</p>
    <p v-if="annotation.rationale" class="review-agent-annotation-rationale">{{ annotation.rationale }}</p>
    <p v-if="deletion.status === 'failed'" class="review-agent-annotation-error" role="alert">
      {{ deletion.message }}
    </p>
  </article>
</template>

<style scoped>
.review-agent-annotation {
  position: relative;
  display: grid;
  gap: 7px;
  padding: 10px;
  border: 1px solid rgb(189 147 249 / 68%);
  background: rgb(189 147 249 / 8%);
}

.review-agent-annotation header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-right: 24px;
  color: #bd93f9;
  font-size: 11px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.review-agent-annotation-delete {
  position: absolute;
  top: 6px;
  right: 6px;
  display: grid;
  width: 22px;
  height: 22px;
  padding: 0;
  place-items: center;
  border: 1px solid transparent;
  background: transparent;
  color: #858b96;
  cursor: pointer;
}

.review-agent-annotation-delete:hover,
.review-agent-annotation-delete:focus-visible {
  border-color: #858b96;
  background: rgb(133 139 150 / 16%);
  color: var(--text);
}

.review-agent-annotation-delete:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}

.review-agent-annotation p {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.review-agent-annotation-summary {
  color: var(--text);
  font-weight: 500;
}

.review-agent-annotation-rationale {
  color: var(--muted);
}

.review-agent-annotation-error {
  color: #ff6e6e;
}
</style>
