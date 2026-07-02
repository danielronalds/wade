import { isSettings, type Settings } from '../types/settings';

const settingsPath = '/api/config';
const reloadConfigPath = '/api/config/reload';

export const fetchSettings = async (): Promise<Settings> => {
  const response = await fetch(settingsPath);
  if (!response.ok) {
    throw new Error(`Settings request failed with ${response.status}`);
  }

  const settings: unknown = await response.json();
  if (!isSettings(settings)) {
    throw new Error('Settings response was invalid');
  }

  return settings;
};

export const saveSettings = async (settings: Settings) => {
  const response = await fetch(settingsPath, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings)
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message.trim() || `Settings save failed with ${response.status}`);
  }
};

export const reloadConfig = async () => {
  const response = await fetch(reloadConfigPath, { method: 'POST' });
  if (!response.ok) {
    throw new Error(`Config reload failed with ${response.status}`);
  }
};
