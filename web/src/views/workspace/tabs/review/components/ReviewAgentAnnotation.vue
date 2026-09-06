<script setup lang="ts">
import type { ReviewAnnotation } from '@/api/generated/wade';

defineProps<{
  annotation: ReviewAnnotation;
  locationLabel: string;
}>();
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
    <p class="review-agent-annotation-summary">{{ annotation.summary }}</p>
    <p v-if="annotation.rationale" class="review-agent-annotation-rationale">{{ annotation.rationale }}</p>
  </article>
</template>

<style scoped>
.review-agent-annotation {
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
  color: #bd93f9;
  font-size: 11px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
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
</style>
