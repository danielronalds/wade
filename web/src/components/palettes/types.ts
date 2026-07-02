export type PaletteNoticeTone = 'warning' | 'error';

export type PaletteNotice = {
  tone: PaletteNoticeTone;
  title: string;
  messages: readonly string[];
};

export type PaletteResult = {
  id: string;
  label: string;
  actionLabel: string;
  isDisabled: boolean;
  run: () => void;
};
