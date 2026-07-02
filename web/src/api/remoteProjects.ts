// NOTE: Vibecoded and not suppppppper reviewed
import {
  isClonedProject,
  isRemoteProject,
  type ClonedProject,
  type RemoteProject
} from '../types/remoteProject';

type RemoteProjectsResponse = {
  projects: RemoteProject[];
};

type CloneRemoteProjectResponse = {
  project: ClonedProject;
};

type ErrorResponse = {
  message: string;
};

const remoteProjectsPath = '/api/remote-projects';
const cloneRemoteProjectPath = '/api/remote-projects/clone';

const isErrorResponse = (value: unknown): value is ErrorResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return typeof (value as Partial<ErrorResponse>).message === 'string';
};

const isRemoteProjectsResponse = (value: unknown): value is RemoteProjectsResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const response = value as Partial<RemoteProjectsResponse>;

  return Array.isArray(response.projects)
    && response.projects.every(isRemoteProject);
};

const isCloneRemoteProjectResponse = (value: unknown): value is CloneRemoteProjectResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return isClonedProject((value as Partial<CloneRemoteProjectResponse>).project);
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

export const listRemoteProjects = async (): Promise<RemoteProject[]> => {
  const response = await fetch(remoteProjectsPath);
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Remote projects request failed with ${response.status}`));
  }

  const projects: unknown = await response.json();
  if (!isRemoteProjectsResponse(projects)) {
    throw new Error('Remote projects response was invalid');
  }

  return projects.projects;
};

export const cloneRemoteProject = async (nameWithOwner: string, directoryIndex: number): Promise<ClonedProject> => {
  const response = await fetch(cloneRemoteProjectPath, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ nameWithOwner, directoryIndex })
  });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Remote project clone failed with ${response.status}`));
  }

  const project: unknown = await response.json();
  if (!isCloneRemoteProjectResponse(project)) {
    throw new Error('Remote project clone response was invalid');
  }

  return project.project;
};
