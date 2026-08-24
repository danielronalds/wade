export interface WorkspaceScreenComponent {
  focusActiveTerminal: () => Promise<void>;
  focusFirstPane?: () => Promise<void>;
  switchToNextTerminal: () => Promise<void>;
  toggleActivePaneZoom?: () => Promise<void>;
}

export interface ReviewScreenComponent extends WorkspaceScreenComponent {
  cancelReview: () => Promise<void>;
  finishReview: () => Promise<void>;
  startReview: () => Promise<void>;
}
