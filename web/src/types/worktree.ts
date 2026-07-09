export type Worktree = {
  name: string;
  projectName: string;
  path: string;
  branch: string;
  isBase: boolean;
  isCurrent: boolean;
  isRemovable: boolean;
  ignoredFileCopyWarnings?: readonly string[];
};

export type RemoteBranch = {
  name: string;
  branch: string;
  hasLocalBranch: boolean;
  isCheckedOut: boolean;
  worktreeName: string;
  worktreeProjectName: string;
};

export type RemoteBranchList = {
  remote: string;
  branches: RemoteBranch[];
};
