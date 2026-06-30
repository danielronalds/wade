<script setup lang="ts">
import { computed } from 'vue';
import { useProjectTerminalTab } from '../composables/useProjectTerminalTab';
import { ProjectTabs } from '../projectTabs';
import type { TerminalConnectionStatus } from '../terminalConnectionStatus';

const props = defineProps<{
  projectName: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

const isActive = computed(() => props.isActive);
const { terminalElement } = useProjectTerminalTab({
  projectName: props.projectName,
  terminalName: ProjectTabs.Terminal,
  isActive,
  onConnectionStatusChange: (status) => emit('connectionStatusChange', status)
});
</script>

<template>
  <section ref="terminalElement" class="terminal-screen" aria-label="Interactive shell"></section>
</template>
