<script setup lang="ts">
import { nextTick, ref } from 'vue';
import type { TerminalConnectionStatus } from '../types/terminalConnectionStatus';
import TerminalPane from './TerminalPane.vue';

const props = defineProps<{
  projectName: string;
  isOpen: boolean;
  isActive: boolean;
}>();

const emit = defineEmits<{
  close: [];
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

type TerminalPaneComponent = {
  focusTerminal: () => Promise<void>;
};

const scratchpadTerminalName = 'scratchpad';
const scratchpadPane = ref<TerminalPaneComponent | null>(null);

const focusActiveTerminal = async () => {
  if (!props.isActive) {
    return;
  }

  await nextTick();
  await scratchpadPane.value?.focusTerminal();
};

const switchToNextTerminal = async () => {
  await focusActiveTerminal();
};

const activateScratchpad = () => {
  void focusActiveTerminal();
};

defineExpose({
  focusActiveTerminal,
  switchToNextTerminal
});
</script>

<template>
  <Teleport to="body">
    <section
      v-show="isOpen"
      id="scratchpad-terminal-backdrop"
      aria-label="Scratchpad terminal backdrop"
    >
      <section
        id="scratchpad-terminal"
        role="dialog"
        aria-modal="true"
        aria-label="Scratchpad terminal"
      >
        <TerminalPane
          ref="scratchpadPane"
          :project-name="projectName"
          :terminal-name="scratchpadTerminalName"
          label="Scratchpad"
          :is-active="isActive"
          show-close-icon
          @activate="activateScratchpad"
          @close="emit('close')"
          @connection-status-change="emit('connectionStatusChange', $event)"
        />
      </section>
    </section>
  </Teleport>
</template>

<style scoped>
#scratchpad-terminal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: grid;
  place-items: center;
  padding: 16px;
  background: rgb(0 0 0 / 28%);
  backdrop-filter: blur(2px);
}

#scratchpad-terminal {
  width: 75vw;
  height: 75vh;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--text);
  background: rgb(23 24 28 / 94%);
  box-shadow: 0 24px 80px rgb(0 0 0 / 38%);
}

#scratchpad-terminal :deep(.terminal-pane) {
  width: 100%;
  height: 100%;
}
</style>
