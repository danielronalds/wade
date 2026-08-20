import type { Workspace, WorkspaceSummary } from '@/api/generated/wade';

type WorkspacePresentationInput = Pick<Workspace | WorkspaceSummary, 'name' | 'repositoryId' | 'branch'>;

export type WorkspacePresentation = {
  root: string;
  branch: string;
  title: string;
};

export const getWorkspaceDisplayRoot = (workspace: WorkspacePresentationInput) =>
  workspace.repositoryId ?? workspace.name;

export const getWorkspaceBranchDisplay = (workspace: WorkspacePresentationInput) => {
  if (!workspace.repositoryId || !workspace.branch) {
    return '';
  }

  if (workspace.branch.isDetached) {
    return 'detached HEAD';
  }

  return workspace.branch.name;
};

export const getWorkspaceSearchCandidates = (workspace: WorkspacePresentationInput) => [
  workspace.name,
  ...(workspace.repositoryId ? [workspace.repositoryId] : []),
  ...(workspace.repositoryId && workspace.branch?.name ? [workspace.branch.name] : [])
];

export const getWorkspacePresentation = (workspace: WorkspacePresentationInput): WorkspacePresentation => {
  const root = getWorkspaceDisplayRoot(workspace);
  const branch = getWorkspaceBranchDisplay(workspace);

  return {
    root,
    branch,
    title: branch === '' ? root : `${root} ${branch}`
  };
};
