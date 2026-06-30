<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useFuzzyProjects } from '../composables/useFuzzyProjects';
import { useProjects } from '../composables/useProjects';

const resultId = (index: number) => `command-palette-result-${index}`;

const router = useRouter();
const { isSyncing, projects, syncProjects } = useProjects();

const isOpen = ref(false);
const query = ref('');
const selectedIndex = ref(0);
const searchInput = ref<HTMLInputElement | null>(null);
let previouslyFocusedElement: HTMLElement | null = null;

const { matchingProjects } = useFuzzyProjects(projects, query);
const selectedProject = computed(() => matchingProjects.value[selectedIndex.value]);
const activeDescendant = computed(() => selectedProject.value ? resultId(selectedIndex.value) : undefined);
const projectSummary = computed(() => {
  const label = projects.value.length === 1 ? 'project' : 'projects';
  const summary = `${projects.value.length} ${label}`;

  return isSyncing.value ? `Syncing ${summary}` : summary;
});

const statusMessage = computed(() => {
  if (projects.value.length === 0 && isSyncing.value) {
    return 'Loading projects';
  }

  if (projects.value.length === 0) {
    return 'No projects found';
  }

  return 'No matching projects';
});

const restorePreviousFocus = () => {
  const element = previouslyFocusedElement;
  previouslyFocusedElement = null;

  if (!element || !document.contains(element)) {
    return;
  }

  element.focus();
};

const openPalette = () => {
  if (!isOpen.value) {
    previouslyFocusedElement = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    query.value = '';
    selectedIndex.value = 0;
  }

  isOpen.value = true;
  void syncProjects();
  void nextTick(() => {
    searchInput.value?.focus();
    searchInput.value?.select();
  });
};

const closePalette = (restoreFocus = true) => {
  if (!isOpen.value) {
    return;
  }

  isOpen.value = false;
  query.value = '';
  selectedIndex.value = 0;

  if (restoreFocus) {
    void nextTick(restorePreviousFocus);
    return;
  }

  previouslyFocusedElement = null;
};

const scrollSelectedProjectIntoView = () => {
  void nextTick(() => {
    document.getElementById(resultId(selectedIndex.value))?.scrollIntoView({ block: 'nearest' });
  });
};

const moveSelection = (offset: number) => {
  const projectCount = matchingProjects.value.length;
  if (projectCount === 0) {
    return;
  }

  selectedIndex.value = (selectedIndex.value + offset + projectCount) % projectCount;
  scrollSelectedProjectIntoView();
};

const openProject = async (projectName: string) => {
  const currentRoute = router.currentRoute.value;
  const currentProjectName = String(currentRoute.params.projectName ?? '');
  if (currentRoute.name === 'project' && currentProjectName === projectName) {
    closePalette();
    return;
  }

  closePalette(false);
  await router.push({ name: 'project', params: { projectName } });
};

const openSelectedProject = () => {
  if (!selectedProject.value) {
    return;
  }

  void openProject(selectedProject.value.projectName);
};

const isPaletteShortcut = (event: KeyboardEvent) => event.ctrlKey
  && !event.altKey
  && !event.metaKey
  && event.key.toLowerCase() === 's';

const handleGlobalKeydown = (event: KeyboardEvent) => {
  if (isPaletteShortcut(event)) {
    event.preventDefault();
    event.stopImmediatePropagation();
    openPalette();
    return;
  }

  if (!isOpen.value) {
    return;
  }

  switch (event.key) {
    case 'Escape':
      event.preventDefault();
      event.stopImmediatePropagation();
      closePalette();
      break;
    case 'ArrowDown':
      event.preventDefault();
      event.stopImmediatePropagation();
      moveSelection(1);
      break;
    case 'ArrowUp':
      event.preventDefault();
      event.stopImmediatePropagation();
      moveSelection(-1);
      break;
    case 'Enter':
      event.preventDefault();
      event.stopImmediatePropagation();
      openSelectedProject();
      break;
  }
};

watch(query, () => {
  selectedIndex.value = 0;
});

watch(matchingProjects, () => {
  if (selectedIndex.value < matchingProjects.value.length) {
    return;
  }

  selectedIndex.value = Math.max(0, matchingProjects.value.length - 1);
});

onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown, true);
  void syncProjects();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeydown, true);
});
</script>

<template>
  <Teleport to="body">
    <section
      v-if="isOpen"
      id="command-palette-backdrop"
      aria-label="Command palette backdrop"
      @mousedown.self="closePalette()"
    >
      <section
        id="command-palette"
        role="dialog"
        aria-modal="true"
        aria-labelledby="command-palette-title"
      >
        <header id="command-palette-header">
          <h2 id="command-palette-title">Open project</h2>
          <p>{{ projectSummary }}</p>
        </header>
        <form id="command-palette-search-form" role="search" @submit.prevent="openSelectedProject">
          <label for="command-palette-search">Project search</label>
          <input
            id="command-palette-search"
            ref="searchInput"
            v-model="query"
            type="search"
            autocomplete="off"
            spellcheck="false"
            role="combobox"
            aria-autocomplete="list"
            aria-controls="command-palette-results"
            :aria-activedescendant="activeDescendant"
            :aria-expanded="isOpen"
            placeholder="Search projects"
          >
        </form>
        <nav id="command-palette-projects" aria-label="Projects WADE can see">
          <ul v-if="matchingProjects.length > 0" id="command-palette-results">
            <li v-for="(match, index) in matchingProjects" :key="match.projectName">
              <button
                :id="resultId(index)"
                class="command-palette-project"
                type="button"
                :data-selected="String(index === selectedIndex)"
                @mouseenter="selectedIndex = index"
                @click="openProject(match.projectName)"
              >
                <span>{{ match.projectName }}</span>
                <span>Open project</span>
              </button>
            </li>
          </ul>
          <p v-else id="command-palette-empty">{{ statusMessage }}</p>
        </nav>
      </section>
    </section>
  </Teleport>
</template>

<style scoped>
#command-palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: grid;
  align-items: start;
  justify-items: center;
  padding: min(18vh, 140px) 16px 16px;
  background: rgb(0 0 0 / 28%);
  backdrop-filter: blur(2px);
}

#command-palette {
  width: min(680px, 100%);
  overflow: hidden;
  border: 1px solid var(--text);
  border-radius: 0;
  background: rgb(23 24 28 / 94%);
  box-shadow: 0 24px 80px rgb(0 0 0 / 38%);
  color: var(--text);
}

#command-palette-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 11px 14px;
  color: var(--muted);
  font-size: 12px;
  line-height: 1;
}

#command-palette-header {
  border-bottom: 1px solid rgb(248 248 242 / 14%);
}

#command-palette-header h2,
#command-palette-header p {
  margin: 0;
}

#command-palette-header h2 {
  color: var(--text);
  font-size: 13px;
  font-weight: 700;
}

#command-palette-search-form {
  border-bottom: 1px solid rgb(248 248 242 / 14%);
}

#command-palette-search-form label {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

#command-palette-search {
  width: 100%;
  height: 58px;
  padding: 0 18px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 18px;
}

#command-palette-search::placeholder {
  color: var(--muted);
}

#command-palette-search::-webkit-search-cancel-button {
  display: none;
}

#command-palette-projects {
  max-height: min(52vh, 420px);
  overflow: auto;
  padding: 6px;
  scrollbar-width: thin;
}

#command-palette-results {
  display: grid;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.command-palette-project {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 11px 12px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.command-palette-project[data-selected="true"] {
  background: rgb(248 248 242 / 10%);
}

.command-palette-project:hover,
.command-palette-project:focus-visible {
  background: rgb(248 248 242 / 12%);
}

.command-palette-project span:first-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-palette-project span:last-child {
  flex: 0 0 auto;
  color: var(--muted);
  font-size: 12px;
}

#command-palette-empty {
  margin: 0;
  padding: 28px 12px;
  color: var(--muted);
  text-align: center;
}

</style>
