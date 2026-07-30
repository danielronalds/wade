import '@xterm/xterm/css/xterm.css';
import '@/assets/styles.css';

import { createPinia } from 'pinia';
import { createApp } from 'vue';
import App from '@/App.vue';
import { router } from '@/router';

if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/service-worker.js').catch(() => undefined);
  });
}

createApp(App).use(createPinia()).use(router).mount('#app');
