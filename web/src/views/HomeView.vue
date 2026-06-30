<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import { useRecentProjects } from '../composables/useRecentProjects';

const { recentProjects } = useRecentProjects();

onMounted(() => {
  document.title = 'WADE';
});
</script>

<template>
  <section id="home-view" aria-labelledby="home-title">
    <h1 id="home-title">WADE</h1>
    <nav aria-label="Recent projects">
      <ul id="recent-projects">
        <li v-for="project in recentProjects" :key="project">
          <RouterLink :to="{ name: 'project', params: { projectName: project } }">
            {{ project }}
          </RouterLink>
        </li>
      </ul>
    </nav>
    <p id="empty-projects" :hidden="recentProjects.length > 0">No projects opened yet</p>
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

nav {
  justify-self: start;
}

#recent-projects,
#empty-projects {
  margin: 0;
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
