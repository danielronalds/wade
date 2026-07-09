import { onBeforeUnmount, onMounted } from 'vue';

export const useCommandPaletteKeyboardShortcuts = ({
  openActiveSessionPicker,
  openCommandPalette,
  openProjectPicker
}: CommandPaletteKeyboardShortcutsOptions) => {
  const handleKeydown = (event: KeyboardEvent) => {
    if (isCtrlShortcut(event, 'p')) {
      stopEvent(event);
      openProjectPicker();
      return;
    }

    if (isCtrlShortcut(event, 's')) {
      stopEvent(event);
      openActiveSessionPicker();
      return;
    }

    if (isCtrlShortcut(event, 'k')) {
      stopEvent(event);
      openCommandPalette();
    }
  };

  onMounted(() => {
    window.addEventListener('keydown', handleKeydown, true);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('keydown', handleKeydown, true);
  });
};

type CommandPaletteKeyboardShortcutsOptions = {
  openActiveSessionPicker: () => void;
  openCommandPalette: () => void;
  openProjectPicker: () => void;
};

const isCtrlShortcut = (event: KeyboardEvent, key: string) => event.ctrlKey
  && !event.altKey
  && !event.metaKey
  && !event.shiftKey
  && event.key.toLowerCase() === key;

const stopEvent = (event: KeyboardEvent) => {
  event.preventDefault();
  event.stopImmediatePropagation();
};
