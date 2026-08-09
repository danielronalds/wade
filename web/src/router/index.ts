import { createRouter, createWebHistory } from 'vue-router';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';
import HomeView from '@/views/home/HomeView.vue';
import SettingsView from '@/views/settings/SettingsView.vue';
import WorkspaceView from '@/views/workspace/WorkspaceView.vue';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsView
    },
    {
      path: '/workspaces/:workspaceId',
      name: 'workspace',
      component: WorkspaceView,
      props: (route) => ({
        workspaceId: String(route.params.workspaceId)
      })
    }
  ]
});

router.beforeResolve(async (to) => {
  if (to.name !== 'workspace') {
    return;
  }

  const workspaceId = String(to.params.workspaceId ?? '');
  await useWorkspaceSessionStore().prepareWorkspaceSession(workspaceId);
});
