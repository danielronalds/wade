import { useRouter } from 'vue-router';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';
import { useSettingsStore } from '@/stores/useSettingsStore';

export const useWorktreeNavigation = () => {
  const router = useRouter();
  const { syncWorkspaces } = useWorkspaces();
  const settingsStore = useSettingsStore();

  const reserveWorktreeTab = (): Window | undefined => {
    if (!settingsStore.settings.openWorktreesInNewTabs) {
      return undefined;
    }

    const reservedTab = window.open('about:blank', '_blank');
    if (!reservedTab) {
      return undefined;
    }

    reservedTab.opener = null;
    return reservedTab;
  };

  const closeReservedWorktreeTab = (reservedTab: Window | undefined) => {
    if (reservedTab && !reservedTab.closed) {
      reservedTab.close();
    }
  };

  const openWorktree = async (worktree: { workspaceId: string }, reservedTab?: Window) => {
    await syncWorkspaces();

    const route = router.resolve({ name: 'workspace', params: { workspaceId: worktree.workspaceId } });
    if (reservedTab && !reservedTab.closed) {
      try {
        reservedTab.location.replace(new URL(route.href, window.location.href).toString());
        return;
      } catch {
        closeReservedWorktreeTab(reservedTab);
      }
    }

    await router.push(route);
  };

  return {
    closeReservedWorktreeTab,
    openWorktree,
    reserveWorktreeTab
  };
};
