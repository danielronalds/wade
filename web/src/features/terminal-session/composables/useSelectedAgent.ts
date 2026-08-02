// NOTE: Vibecoded and not suppppppper reviewed
const selectedAgentStorageKey = (workspaceId: string) => `wade:selected-agent:v3:${workspaceId}`;
const legacySelectedAgentStorageKey = (workspaceId: string) => `wade:selected-agent:v2:${workspaceId}`;

export const loadSelectedAgentName = (workspaceId: string) => {
  const selectedAgentName = window.localStorage.getItem(selectedAgentStorageKey(workspaceId));
  if (selectedAgentName !== null) {
    return selectedAgentName;
  }

  const legacySelectedAgentName = window.localStorage.getItem(legacySelectedAgentStorageKey(workspaceId)) ?? '';
  if (legacySelectedAgentName !== '') {
    window.localStorage.setItem(selectedAgentStorageKey(workspaceId), legacySelectedAgentName);
    window.localStorage.removeItem(legacySelectedAgentStorageKey(workspaceId));
  }

  return legacySelectedAgentName;
};

export const storeSelectedAgentName = (workspaceId: string, agentName: string) => {
  window.localStorage.setItem(selectedAgentStorageKey(workspaceId), agentName);
  window.localStorage.removeItem(legacySelectedAgentStorageKey(workspaceId));
};
