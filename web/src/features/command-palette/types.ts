import type { Component } from 'vue';

export type PaletteNoticeTone = 'warning' | 'error';

export type PaletteNotice = {
  tone: PaletteNoticeTone;
  title: string;
  messages: readonly string[];
};

export type PaletteResult = {
  id: string;
  label: string;
  secondaryLabel?: string;
  icon?: Component;
  title?: string;
  actionLabel?: string;
  isDisabled: boolean;
  run: () => void;
};
