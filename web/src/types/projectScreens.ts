export interface ProjectScreenComponent {
  focusActiveTerminal: () => Promise<void>;
  focusFirstPane?: () => Promise<void>;
  switchToNextTerminal: () => Promise<void>;
  toggleActivePaneZoom?: () => Promise<void>;
}

export interface ReviewScreenComponent extends ProjectScreenComponent {
  cancelReview: () => Promise<void>;
  startReview: () => Promise<void>;
}
