export interface ProjectScreenComponent {
  focusActiveTerminal: () => Promise<void>;
  focusFirstPane?: () => Promise<void>;
  switchToNextTerminal: () => Promise<void>;
}

export interface ReviewScreenComponent extends ProjectScreenComponent {
  startReview: () => Promise<void>;
}
