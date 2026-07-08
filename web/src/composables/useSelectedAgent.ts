// NOTE: Vibecoded and not suppppppper reviewed
const selectedAgentStorageKey = (projectName: string) => `wade:selected-agent:v2:${projectName}`;

export const loadSelectedAgentName = (projectName: string) => window.localStorage.getItem(selectedAgentStorageKey(projectName)) ?? '';

export const storeSelectedAgentName = (projectName: string, agentName: string) => {
  window.localStorage.setItem(selectedAgentStorageKey(projectName), agentName);
};
