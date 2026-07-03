<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import { useProjects } from '../composables/useProjects';
import { useRecentProjects } from '../composables/useRecentProjects';

const { syncProjects } = useProjects();
const { recentProjects, removeUnavailableRecentProjects } = useRecentProjects();

const syncRecentProjects = async () => {
  const availableProjects = await syncProjects();
  if (!availableProjects) {
    return;
  }

  removeUnavailableRecentProjects(availableProjects);
};

onMounted(() => {
  document.title = 'WADE';
  void syncRecentProjects();
});
</script>

<template>
  <section id="home-view" aria-labelledby="home-title">
    <h1 id="home-title">WADE</h1>
    <p id="home-subtitle">Web-based Agentic Development Environment</p>
    <nav v-if="recentProjects.length > 0" id="recent-projects-nav" aria-labelledby="recent-projects-title">
      <h2 id="recent-projects-title">Recent Projects</h2>
      <ul id="recent-projects">
        <li v-for="project in recentProjects" :key="project">
          <RouterLink :to="{ name: 'project', params: { projectName: project } }">
            {{ project }}
          </RouterLink>
        </li>
      </ul>
    </nav>
    <p v-else id="empty-projects">Press Ctrl + P to get started</p>
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

#recent-projects-nav {
  display: grid;
  gap: 10px;
  justify-self: start;
  margin-top: 6px;
}

#recent-projects-title,
#recent-projects,
#empty-projects {
  margin: 0;
}

#recent-projects-title {
  color: var(--text);
  font-size: 14px;
  font-weight: 400;
  line-height: 1;
  text-align: left;
}

#recent-projects {
  text-align: left;
}

#recent-projects a {
  color: var(--text);
}

#empty-projects {
  text-align: center;
}
</style>
