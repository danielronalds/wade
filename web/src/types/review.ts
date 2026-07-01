// NOTE: Vibecoded and not suppppppper reviewed
export type ReviewScope = 'git-diff' | 'last-commit' | 'all-files';

export type ChangeStatus = 'modified' | 'added' | 'deleted' | 'renamed';

export interface ReviewFileComparison {
  status: ChangeStatus;
  oldPath: string | null;
  newPath: string | null;
  displayPath: string;
  hasOriginal: boolean;
  hasModified: boolean;
}

export interface ReviewFile {
  id: string;
  path: string;
  worktreeStatus: ChangeStatus | null;
  hasWorkingTreeFile: boolean;
  inGitDiff: boolean;
  inLastCommit: boolean;
  gitDiff: ReviewFileComparison | null;
  lastCommit: ReviewFileComparison | null;
}

export interface ReviewData {
  repoRoot: string;
  branchName: string;
  files: ReviewFile[];
}

export interface ReviewFileContents {
  originalContent: string;
  modifiedContent: string;
}

export type CommentSide = 'original' | 'modified' | 'file';

export type ReviewCommentKind = 'feedback' | 'question';

export interface ReviewComment {
  id: string;
  fileId: string;
  scope: ReviewScope;
  side: CommentSide;
  kind: ReviewCommentKind;
  startLine: number | null;
  endLine: number | null;
  body: string;
}
