<script setup lang="ts">
import { computed } from 'vue';
import { useProjectTerminalTab } from '../composables/useProjectTerminalTab';
import { ProjectTabs } from '../types/projectTabs';
import type { TerminalConnectionStatus } from '../types/terminalConnectionStatus';

const props = defineProps<{
  projectName: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

const isActive = computed(() => props.isActive);
const { focusTerminal, terminalElement } = useProjectTerminalTab({
  projectName: props.projectName,
  terminalName: ProjectTabs.Server,
  isActive,
  onConnectionStatusChange: (status) => emit('connectionStatusChange', status)
});

const focusActiveTerminal = async () => {
  await focusTerminal();
};

const switchToNextTerminal = async () => {
  await focusTerminal();
};

defineExpose({
  focusActiveTerminal,
  switchToNextTerminal
});
</script>

<template>
  <section ref="terminalElement" class="terminal-screen" aria-label="Server shell"></section>
</template>
