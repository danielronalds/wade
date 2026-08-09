import { sendWorkspaceTerminalInput } from '@/api/generated/wade';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';

export const pasteIntoAgentTerminal = async (workspaceId: string, prompt: string) => {
  const agentName = useWorkspaceSessionStore().getSelectedAgentName(workspaceId);
  if (agentName === '') {
    throw new Error('No Agent terminal is selected');
  }

  await sendWorkspaceTerminalInput(workspaceId, `agent:${agentName.toLowerCase()}`, {
    mode: 'bracketed-paste',
    text: prompt
  });
};
