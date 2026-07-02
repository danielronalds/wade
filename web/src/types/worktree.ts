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

const isStringArray = (value: unknown): value is string[] => Array.isArray(value)
  && value.every((item) => typeof item === 'string');

export const isWorktree = (value: unknown): value is Worktree => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const worktree = value as Partial<Worktree>;

  return typeof worktree.name === 'string'
    && typeof worktree.projectName === 'string'
    && typeof worktree.path === 'string'
    && typeof worktree.branch === 'string'
    && typeof worktree.isBase === 'boolean'
    && typeof worktree.isCurrent === 'boolean'
    && typeof worktree.isRemovable === 'boolean'
    && (worktree.ignoredFileCopyWarnings === undefined || isStringArray(worktree.ignoredFileCopyWarnings));
};

export const isRemoteBranch = (value: unknown): value is RemoteBranch => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const branch = value as Partial<RemoteBranch>;

  return typeof branch.name === 'string'
    && typeof branch.branch === 'string'
    && typeof branch.hasLocalBranch === 'boolean'
    && typeof branch.isCheckedOut === 'boolean'
    && typeof branch.worktreeName === 'string'
    && typeof branch.worktreeProjectName === 'string';
};

export const isRemoteBranchList = (value: unknown): value is RemoteBranchList => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const branchList = value as Partial<RemoteBranchList>;

  return typeof branchList.remote === 'string'
    && Array.isArray(branchList.branches)
    && branchList.branches.every(isRemoteBranch);
};
