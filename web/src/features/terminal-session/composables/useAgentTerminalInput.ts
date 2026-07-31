// NOTE: Vibecoded and not suppppppper reviewed
import { loadSelectedAgentName } from '@/features/terminal-session/composables/useSelectedAgent';
import { createTerminalWebSocket } from '@/features/terminal-session/createTerminalWebSocket';

const encoder = new TextEncoder();
const agentTerminalName = 'agent';
const bracketedPasteStart = '\x1b[200~';
const bracketedPasteEnd = '\x1b[201~';

export const pasteIntoAgentTerminal = (projectName: string, prompt: string) => new Promise<void>((resolve, reject) => {
  const agentName = loadSelectedAgentName(projectName);
  const socket = createTerminalWebSocket({
    project: projectName,
    terminal: agentTerminalName,
    agent: agentName || undefined
  });

  let hasSentPrompt = false;

  const cleanup = () => {
    socket.removeEventListener('open', handleOpen);
    socket.removeEventListener('error', handleError);
    socket.removeEventListener('close', handleClose);
  };

  const settle = (callback: () => void) => {
    cleanup();
    callback();
  };

  function handleOpen() {
    hasSentPrompt = true;
    socket.send(encoder.encode(`${bracketedPasteStart}${prompt}${bracketedPasteEnd}`));
    window.setTimeout(() => {
      socket.close();
      settle(resolve);
    }, 100);
  }

  function handleError() {
    settle(() => reject(new Error('Could not connect to the Agent terminal')));
  }

  function handleClose() {
    if (hasSentPrompt) {
      return;
    }

    settle(() => reject(new Error('Agent terminal connection closed before the prompt was sent')));
  }

  socket.addEventListener('open', handleOpen);
  socket.addEventListener('error', handleError);
  socket.addEventListener('close', handleClose);
});
