import { onBeforeUnmount, readonly, ref } from 'vue';

export type WorkspaceLink = 'linear-ticket' | 'pull-request' | 'github';

export const useWorkspaceLinkClipboard = () => {
  const copiedWorkspaceLink = ref<WorkspaceLink | null>(null);
  const clipboardAnnouncement = ref('');
  let copiedStateResetTimeout: ReturnType<typeof setTimeout> | undefined;

  const clearCopiedStateResetTimeout = () => {
    if (copiedStateResetTimeout === undefined) {
      return;
    }

    clearTimeout(copiedStateResetTimeout);
    copiedStateResetTimeout = undefined;
  };

  const copyWorkspaceLink = async (workspaceLink: WorkspaceLink, label: string, url: string) => {
    if (url === '') {
      return;
    }

    clearCopiedStateResetTimeout();

    try {
      await navigator.clipboard.writeText(url);
      copiedWorkspaceLink.value = workspaceLink;
      clipboardAnnouncement.value = `${label} link copied`;
      copiedStateResetTimeout = setTimeout(() => {
        copiedWorkspaceLink.value = null;
        clipboardAnnouncement.value = '';
        copiedStateResetTimeout = undefined;
      }, 1800);
    } catch {
      copiedWorkspaceLink.value = null;
      clipboardAnnouncement.value = `Could not copy ${label.toLowerCase()} link`;
    }
  };

  onBeforeUnmount(clearCopiedStateResetTimeout);

  return {
    clipboardAnnouncement: readonly(clipboardAnnouncement),
    copiedWorkspaceLink: readonly(copiedWorkspaceLink),
    copyWorkspaceLink
  };
};
