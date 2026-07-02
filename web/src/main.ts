import '@xterm/xterm/css/xterm.css';
import './styles.css';

import { createApp } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import App from './App.vue';
import HomeView from './views/HomeView.vue';
import ProjectView from './views/ProjectView.vue';
import SettingsView from './views/SettingsView.vue';

const router = createRouter({
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

createApp(App).use(router).mount('#app');
