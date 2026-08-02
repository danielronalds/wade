<script setup lang="ts">
import { ref } from 'vue';
import TerminalPane from '@/views/workspace/components/terminal-pane/TerminalPane.vue';
import { WorkspaceTabs } from '@/types/workspaceTabs';
import type { TerminalConnectionStatus } from '@/types/terminalConnectionStatus';

const props = defineProps<{
  workspaceId: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

type TerminalPaneComponent = {
  focusTerminal: () => Promise<void>;
};

const serverLabel = 'Server';
const serverPane = ref<TerminalPaneComponent | null>(null);

const focusActiveTerminal = async () => {
  await serverPane.value?.focusTerminal();
};

const switchToNextTerminal = async () => {
  await focusActiveTerminal();
};

defineExpose({
  focusActiveTerminal,
  switchToNextTerminal
});
</script>

<template>
  <section class="server-tab" aria-label="Server terminal pane">
    <TerminalPane
      ref="serverPane"
      :workspace-id="workspaceId"
      :terminal-id="WorkspaceTabs.Server"
      :label="serverLabel"
      :is-active="isActive"
      @connection-status-change="emit('connectionStatusChange', $event)"
    />
  </section>
</template>

<style scoped>
.server-tab {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--window);
}

.server-tab :deep(.terminal-pane) {
  width: 100%;
  height: 100%;
}
</style>
