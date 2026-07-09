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
