// NOTE: Vibecoded and not suppppppper reviewed
import { sendWorkspaceTerminalInput } from '@/api/generated/wade';
import { loadSelectedAgentName } from '@/features/terminal-session/composables/useSelectedAgent';

export const pasteIntoAgentTerminal = async (workspaceId: string, prompt: string) => {
  const agentName = loadSelectedAgentName(workspaceId);
  if (agentName === '') {
    throw new Error('No Agent terminal is selected');
  }

  await sendWorkspaceTerminalInput(workspaceId, `agent:${agentName.toLowerCase()}`, {
    mode: 'bracketed-paste',
    text: prompt
  });
};
