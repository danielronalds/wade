export type PaletteResult = {
  id: string;
  label: string;
  actionLabel: string;
  isDisabled: boolean;
  run: () => void;
};
