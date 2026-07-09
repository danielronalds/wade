import { responseErrorMessage } from '@/api/http';

type SessionsResponse = {
  sessions: string[];
};

const sessionPath = '/api/session';
const sessionsPath = '/api/sessions';

const isSessionsResponse = (value: unknown): value is SessionsResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const response = value as Partial<SessionsResponse>;

  return Array.isArray(response.sessions)
    && response.sessions.every((session) => typeof session === 'string');
};

export const listActiveProjectSessions = async (): Promise<string[]> => {
  const response = await fetch(sessionsPath);

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Sessions request failed with ${response.status}`));
  }

  const body: unknown = await response.json();
  if (!isSessionsResponse(body)) {
    throw new Error('Sessions response was invalid');
  }

  return Array.from(new Set(body.sessions.filter((session) => session.length > 0)))
    .sort((firstSession, secondSession) => firstSession.localeCompare(secondSession));
};

export const closeProjectSession = async (project: string): Promise<void> => {
  const response = await fetch(`${sessionPath}/${encodeURIComponent(project)}`, { method: 'DELETE' });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Session close failed with ${response.status}`));
  }
};
