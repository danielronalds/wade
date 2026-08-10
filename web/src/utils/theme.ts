export const themeAccentColors = ['white', 'purple', 'orange'] as const;

export type ThemeAccentColor = (typeof themeAccentColors)[number];

export const defaultThemeAccentColor: ThemeAccentColor = 'white';

const storedThemeAccentColorKey = 'wade-theme-accent-color';
const themeAccentColorChangedEvent = 'theme-accent-color-changed';

const themeAccentColorValues: Record<ThemeAccentColor, string> = {
  white: '#f8f8f2',
  orange: '#ffb86c',
  purple: '#bd93f9'
};

export const themeAccentColorOptions = themeAccentColors.map((value) => ({
  value,
  label: value[0].toUpperCase() + value.slice(1),
  color: themeAccentColorValues[value]
}));

export const isThemeAccentColor = (value: unknown): value is ThemeAccentColor =>
  typeof value === 'string' && themeAccentColors.includes(value as ThemeAccentColor);

export const normaliseThemeAccentColor = (value: unknown): ThemeAccentColor =>
  isThemeAccentColor(value) ? value : defaultThemeAccentColor;

export const storedThemeAccentColor = () =>
  normaliseThemeAccentColor(window.localStorage.getItem(storedThemeAccentColorKey));

export const applyThemeAccentColor = (color: unknown) => {
  const themeAccentColor = normaliseThemeAccentColor(color);
  document.documentElement.dataset.themeAccentColor = themeAccentColor;
  window.localStorage.setItem(storedThemeAccentColorKey, themeAccentColor);
  window.dispatchEvent(new CustomEvent(themeAccentColorChangedEvent, { detail: themeAccentColor }));
};
