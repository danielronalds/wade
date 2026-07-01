export interface ProjectScreenComponent {
  focusActiveTerminal: () => Promise<void>;
  switchToNextTerminal: () => Promise<void>;
}

export interface ReviewScreenComponent extends ProjectScreenComponent {
  startReview: () => Promise<void>;
}
