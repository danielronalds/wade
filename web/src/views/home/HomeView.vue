<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import { useRecentWorkspaces } from '@/features/workspaces/composables/useRecentWorkspaces';
import { useWorkspaces } from '@/features/workspaces/composables/useWorkspaces';

const { syncWorkspaces } = useWorkspaces();
const { recentWorkspaceIds, removeUnavailableRecentWorkspaces } = useRecentWorkspaces();

const syncRecentWorkspaces = async () => {
  const availableWorkspaces = await syncWorkspaces();
  if (!availableWorkspaces) {
    return;
  }

  removeUnavailableRecentWorkspaces(availableWorkspaces);
};

onMounted(() => {
  document.title = 'WADE';
  void syncRecentWorkspaces();
});
</script>

<template>
  <section id="home-view" aria-labelledby="home-title">
    <h1 id="home-title">WADE</h1>
    <p id="home-subtitle">Web-based Agentic Development Environment</p>
    <nav v-if="recentWorkspaceIds.length > 0" id="recent-workspaces-nav" aria-labelledby="recent-workspaces-title">
      <h2 id="recent-workspaces-title">Recent Workspaces</h2>
      <ul id="recent-workspaces">
        <li v-for="workspaceId in recentWorkspaceIds" :key="workspaceId">
          <RouterLink :to="{ name: 'workspace', params: { workspaceId } }">
            {{ workspaceId }}
          </RouterLink>
        </li>
      </ul>
    </nav>
    <p v-else id="empty-workspaces">Press Ctrl + P to get started</p>
  </section>
</template>

<style scoped>
#home-view {
  width: 100vw;
  height: 100vh;
  display: grid;
  gap: 24px;
  place-content: center;
  place-items: center;
  padding: 24px;
  color: var(--text);
}

#home-title {
  margin: 0;
  font-size: clamp(48px, 12vw, 120px);
  line-height: 1;
}

#home-subtitle {
  margin: -12px 0 0;
  color: var(--muted);
  font-size: clamp(13px, 2vw, 17px);
  text-align: center;
}

#recent-workspaces-nav {
  display: grid;
  gap: 10px;
  justify-self: start;
  margin-top: 6px;
}

#recent-workspaces-title,
#recent-workspaces,
#empty-workspaces {
  margin: 0;
}

#recent-workspaces-title {
  color: var(--text);
  font-size: 14px;
  font-weight: 400;
  line-height: 1;
  text-align: left;
}

#recent-workspaces {
  text-align: left;
}

#recent-workspaces a {
  color: var(--text);
}

#empty-workspaces {
  text-align: center;
}
</style>
