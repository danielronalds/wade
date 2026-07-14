import { useRouter } from 'vue-router';
import { useProjects } from '@/features/projects/composables/useProjects';
import { useSettingsStore } from '@/stores/useSettingsStore';
import type { Worktree } from '@/types/worktree';

export const useWorktreeNavigation = () => {
  const router = useRouter();
  const { syncProjects } = useProjects();
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

  const openWorktree = async (worktree: Worktree, reservedTab?: Window) => {
    await syncProjects();

    const route = router.resolve({ name: 'project', params: { projectName: worktree.projectName } });
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
