import { responseErrorMessage } from '@/api/http';
import {
  isRemoteBranchList,
  isWorktree,
  type RemoteBranchList,
  type Worktree
} from '@/types/worktree';

type WorktreesResponse = {
  worktrees: Worktree[];
};

type WorktreeResponse = {
  worktree: Worktree;
};

const worktreesPath = '/api/worktrees';
const remoteBranchesPath = '/api/worktrees/remote-branches';

const isWorktreesResponse = (value: unknown): value is WorktreesResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const response = value as Partial<WorktreesResponse>;

  return Array.isArray(response.worktrees)
    && response.worktrees.every(isWorktree);
};

const isWorktreeResponse = (value: unknown): value is WorktreeResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return isWorktree((value as Partial<WorktreeResponse>).worktree);
};

export const listWorktrees = async (project: string): Promise<Worktree[]> => {
  const params = new URLSearchParams({ project });
  const response = await fetch(`${worktreesPath}?${params}`);
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Worktrees request failed with ${response.status}`));
  }

  const worktrees: unknown = await response.json();
  if (!isWorktreesResponse(worktrees)) {
    throw new Error('Worktrees response was invalid');
  }

  return worktrees.worktrees;
};

export const listRemoteBranches = async (project: string): Promise<RemoteBranchList> => {
  const params = new URLSearchParams({ project });
  const response = await fetch(`${remoteBranchesPath}?${params}`);
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Remote branches request failed with ${response.status}`));
  }

  const branches: unknown = await response.json();
  if (!isRemoteBranchList(branches)) {
    throw new Error('Remote branches response was invalid');
  }

  return branches;
};

export const createWorktree = async (project: string, branch: string): Promise<Worktree> => {
  const response = await fetch(worktreesPath, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ project, branch })
  });
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Worktree creation failed with ${response.status}`));
  }

  const worktree: unknown = await response.json();
  if (!isWorktreeResponse(worktree)) {
    throw new Error('Worktree response was invalid');
  }

  return worktree.worktree;
};

export const removeWorktree = async (project: string, worktree: string): Promise<Worktree> => {
  const response = await fetch(worktreesPath, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ project, worktree })
  });
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Worktree removal failed with ${response.status}`));
  }

  const removed: unknown = await response.json();
  if (!isWorktreeResponse(removed)) {
    throw new Error('Worktree response was invalid');
  }

  return removed.worktree;
};
