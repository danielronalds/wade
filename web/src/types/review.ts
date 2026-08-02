// NOTE: Vibecoded and not suppppppper reviewed
import type {
  GetReviewSnapshotFileContentsScope,
  ReviewChangeStatus,
  ReviewFile,
  ReviewFileComparison,
  ReviewFileContents,
  ReviewSnapshot,
  ReviewSnapshotPullRequest
} from '@/api/generated/wade';

export type ReviewScope = GetReviewSnapshotFileContentsScope;
export type ChangeStatus = ReviewChangeStatus;
export type ReviewData = ReviewSnapshot;
export type ReviewPullRequest = ReviewSnapshotPullRequest;
export type { ReviewFile, ReviewFileComparison, ReviewFileContents };

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
