<script setup lang="ts">
import { RouterView } from 'vue-router';
</script>

<template>
  <main>
    <RouterView v-slot="{ Component, route }">
      <component :is="Component" :key="route.fullPath" />
    </RouterView>
  </main>
</template>

<style scoped>
main {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--window);
}
</style>
