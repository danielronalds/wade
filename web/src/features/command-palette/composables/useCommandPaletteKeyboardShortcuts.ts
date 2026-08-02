import { onBeforeUnmount, onMounted } from 'vue';

export const useCommandPaletteKeyboardShortcuts = ({
  openActiveWorkspacePicker,
  openCommandPalette,
  openWorkspacePicker
}: CommandPaletteKeyboardShortcutsOptions) => {
  const handleKeydown = (event: KeyboardEvent) => {
    if (isCtrlShortcut(event, 'p')) {
      stopEvent(event);
      openWorkspacePicker();
      return;
    }

    if (isCtrlShortcut(event, 's')) {
      stopEvent(event);
      openActiveWorkspacePicker();
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
  openActiveWorkspacePicker: () => void;
  openCommandPalette: () => void;
  openWorkspacePicker: () => void;
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
