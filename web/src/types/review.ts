// NOTE: Vibecoded and not suppppppper reviewed
import { GetReviewSnapshotFileContentsScope } from '@/api/generated/wade';
import type {
  ReviewChangeStatus,
  ReviewFile,
  ReviewFileComparison,
  ReviewFileContents,
  ReviewSnapshot,
  ReviewSnapshotPullRequest
} from '@/api/generated/wade';

export const reviewScopes = Object.values(GetReviewSnapshotFileContentsScope);
export type ReviewScope = GetReviewSnapshotFileContentsScope;
export type ChangeStatus = ReviewChangeStatus;
export type ReviewData = ReviewSnapshot;
export type ReviewPullRequest = ReviewSnapshotPullRequest;
export type { ReviewFile, ReviewFileComparison, ReviewFileContents };

export const commentSides = ['original', 'modified', 'file'] as const;
export type CommentSide = typeof commentSides[number];

export const reviewCommentKinds = ['feedback', 'question'] as const;
export type ReviewCommentKind = typeof reviewCommentKinds[number];

export type ReviewState = 'idle' | 'loading' | 'ready' | 'error';

export interface DraftReviewComment {
  fileId: string;
  filePath: string;
  scope: ReviewScope;
  side: CommentSide;
  startLine: number | null;
  endLine: number | null;
}

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

export interface ReviewCheckpoint {
  snapshotId: string;
  activeScope: ReviewScope;
  activeFileId: string | null;
  filterText: string;
  reviewedFiles: Record<string, boolean>;
  collapsedDirectories: Record<string, boolean>;
  comments: ReviewComment[];
  overallComment: string;
  draftComment: DraftReviewComment | null;
  draftCommentBody: string;
  draftCommentKind: ReviewCommentKind;
  isOverallNoteOpen: boolean;
  overallNoteDraft: string;
  hideUnchanged: boolean;
  renderSideBySide: boolean;
  wrapLines: boolean;
}

export const isReviewInProgressState = (state: ReviewState) => state === 'loading' || state === 'ready';
