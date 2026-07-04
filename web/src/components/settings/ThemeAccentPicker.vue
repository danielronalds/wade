<script setup lang="ts">
import { themeAccentColorOptions, type ThemeAccentColor } from '../../theme';

const themeAccentColor = defineModel<ThemeAccentColor>('themeAccentColor', { required: true });
</script>

<template>
  <section id="theme-section" aria-labelledby="theme-title">
    <header class="settings-section-header">
      <section>
        <h2 id="theme-title">Theme</h2>
        <p>Choose the accent colour used across WADE.</p>
      </section>
    </header>

    <fieldset id="theme-accent-picker" aria-label="Accent colour">
      <legend>Accent colour</legend>
      <label
        v-for="option in themeAccentColorOptions"
        :key="option.value"
        class="theme-accent-option"
        :title="option.label"
      >
        <input
          class="theme-accent-input"
          type="radio"
          name="theme-accent-color"
          v-model="themeAccentColor"
          :value="option.value"
          :aria-label="option.label"
        >
        <span class="theme-accent-swatch" :style="{ '--theme-accent-option-color': option.color }">
          <span aria-hidden="true"></span>
        </span>
        <span class="theme-accent-label">{{ option.label }}</span>
      </label>
    </fieldset>
  </section>
</template>

<style scoped>
#theme-section {
  width: min(860px, 100%);
  display: grid;
  gap: 18px;
}

.settings-section-header {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: 18px;
  padding-bottom: 13px;
  border-bottom: 1px solid rgb(var(--accent-rgb) / 18%);
}

.settings-section-header h2,
.settings-section-header p {
  margin: 0;
}

.settings-section-header h2 {
  font-size: 18px;
}

.settings-section-header p {
  margin-top: 7px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}

#theme-accent-picker {
  min-width: 0;
  display: flex;
  align-items: start;
  justify-content: start;
  gap: 14px;
  margin: 0;
  padding: 0;
  border: 0;
}

#theme-accent-picker legend {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
  border: 0;
}

.theme-accent-option {
  position: relative;
  display: grid;
  justify-items: center;
  gap: 8px;
  color: var(--muted);
  font-size: 12px;
  cursor: pointer;
}

.theme-accent-input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.theme-accent-swatch {
  width: 34px;
  height: 34px;
  display: block;
  padding: 4px;
  border: 1px solid #f8f8f2;
  background: transparent;
}

.theme-accent-swatch span {
  width: 100%;
  height: 100%;
  display: block;
  background: var(--theme-accent-option-color);
}

.theme-accent-input:checked + .theme-accent-swatch,
.theme-accent-input:focus-visible + .theme-accent-swatch {
  outline: 1px solid var(--text);
  outline-offset: 3px;
}

.theme-accent-input:checked ~ .theme-accent-label {
  color: var(--text);
}

@media (max-width: 720px) {
  .settings-section-header {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
