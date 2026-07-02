export type RemoteProject = {
  name: string;
  nameWithOwner: string;
  url: string;
  sshUrl: string;
  isLocal: boolean;
  localName: string;
};

export type ClonedProject = {
  name: string;
  path: string;
};

export const isRemoteProject = (value: unknown): value is RemoteProject => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const project = value as Partial<RemoteProject>;

  return typeof project.name === 'string'
    && typeof project.nameWithOwner === 'string'
    && typeof project.url === 'string'
    && typeof project.sshUrl === 'string'
    && typeof project.isLocal === 'boolean'
    && typeof project.localName === 'string';
};

export const isClonedProject = (value: unknown): value is ClonedProject => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const project = value as Partial<ClonedProject>;

  return typeof project.name === 'string'
    && typeof project.path === 'string';
};
