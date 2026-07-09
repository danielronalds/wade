import { createRouter, createWebHistory } from 'vue-router';
import HomeView from '@/views/home/HomeView.vue';
import ProjectView from '@/views/project/ProjectView.vue';
import SettingsView from '@/views/settings/SettingsView.vue';

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
      path: '/:projectName',
      name: 'project',
      component: ProjectView,
      props: (route) => ({
        projectName: String(route.params.projectName)
      })
    }
  ]
});
