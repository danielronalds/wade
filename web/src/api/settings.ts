import { isSettings, type Settings } from '../types/settings';

type ErrorResponse = {
  message: string;
};

const settingsPath = '/api/config';
const reloadConfigPath = '/api/config/reload';

const isErrorResponse = (value: unknown): value is ErrorResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return typeof (value as Partial<ErrorResponse>).message === 'string';
};

const responseErrorMessage = async (response: Response, fallback: string) => {
  const text = await response.text();
  if (text.trim() === '') {
    return fallback;
  }

  try {
    const body: unknown = JSON.parse(text);
    if (isErrorResponse(body) && body.message.trim() !== '') {
      return body.message;
    }
  } catch {
    return text.trim();
  }

  return fallback;
};

export const fetchSettings = async (): Promise<Settings> => {
  const response = await fetch(settingsPath);
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Settings request failed with ${response.status}`));
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
    throw new Error(await responseErrorMessage(response, `Settings save failed with ${response.status}`));
  }
};

export const reloadConfig = async () => {
  const response = await fetch(reloadConfigPath, { method: 'POST' });
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Config reload failed with ${response.status}`));
  }
};
