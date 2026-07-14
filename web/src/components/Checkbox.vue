<script setup lang="ts">
const checked = defineModel<boolean>({ required: true });

defineProps<{
  id?: string;
  disabled?: boolean;
}>();
</script>

<template>
  <label class="checkbox" :data-disabled="String(disabled ?? false)">
    <input
      :id="id"
      v-model="checked"
      type="checkbox"
      :disabled="disabled"
    >
    <span><slot /></span>
  </label>
</template>

<style scoped>
.checkbox {
  width: fit-content;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  font-size: 13px;
  cursor: pointer;
}

.checkbox[data-disabled="true"] {
  cursor: not-allowed;
  opacity: 0.45;
}

.checkbox input {
  width: 15px;
  height: 15px;
  appearance: none;
  display: grid;
  place-items: center;
  margin: 0;
  border: 1px solid currentColor;
  border-radius: 0;
  background: transparent;
  color: inherit;
}

.checkbox:hover input,
.checkbox:focus-within input,
.checkbox input:checked {
  color: var(--text);
}

.checkbox input:checked::before {
  width: 9px;
  height: 9px;
  background: currentColor;
  content: '';
}

.checkbox input:focus-visible {
  outline: 1px solid var(--text);
  outline-offset: 2px;
}
</style>
