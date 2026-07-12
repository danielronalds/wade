import { onBeforeUnmount, readonly, ref } from 'vue';

export type ProjectLink = 'linear-ticket' | 'pull-request' | 'github';

export const useProjectLinkClipboard = () => {
  const copiedProjectLink = ref<ProjectLink | null>(null);
  const clipboardAnnouncement = ref('');
  let copiedStateResetTimeout: ReturnType<typeof setTimeout> | undefined;

  const clearCopiedStateResetTimeout = () => {
    if (copiedStateResetTimeout === undefined) {
      return;
    }

    clearTimeout(copiedStateResetTimeout);
    copiedStateResetTimeout = undefined;
  };

  const copyProjectLink = async (projectLink: ProjectLink, label: string, url: string) => {
    if (url === '') {
      return;
    }

    clearCopiedStateResetTimeout();

    try {
      await navigator.clipboard.writeText(url);
      copiedProjectLink.value = projectLink;
      clipboardAnnouncement.value = `${label} link copied`;
      copiedStateResetTimeout = setTimeout(() => {
        copiedProjectLink.value = null;
        clipboardAnnouncement.value = '';
        copiedStateResetTimeout = undefined;
      }, 1800);
    } catch {
      copiedProjectLink.value = null;
      clipboardAnnouncement.value = `Could not copy ${label.toLowerCase()} link`;
    }
  };

  onBeforeUnmount(clearCopiedStateResetTimeout);

  return {
    clipboardAnnouncement: readonly(clipboardAnnouncement),
    copiedProjectLink: readonly(copiedProjectLink),
    copyProjectLink
  };
};
