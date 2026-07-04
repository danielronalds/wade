<script setup lang="ts">
import { GitBranch, RefreshCw } from '@lucide/vue';
import { computed, onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import { useProjectDetails } from '../composables/useProjectDetails';
import GitHubIcon from '../icons/GitHubIcon.vue';
import LinearIcon from '../icons/LinearIcon.vue';

const props = defineProps<{
  projectName: string;
  connectionStatusText: string;
  isConnected: boolean;
}>();

const {
  gitBranch,
  isLoading: isProjectDetailsLoading,
  githubUrl,
  linearTicketUrl,
  loadProjectDetails,
  pullRequestUrl
} = useProjectDetails(props.projectName);

const projectDisplayName = computed(() => props.projectName.split('-feature')[0] || props.projectName);
const isLinearTicketButtonDisabled = computed(() => isProjectDetailsLoading.value || linearTicketUrl.value === '');
const isPullRequestButtonDisabled = computed(() => isProjectDetailsLoading.value || pullRequestUrl.value === '');
const isGitHubButtonDisabled = computed(() => isProjectDetailsLoading.value || githubUrl.value === '');
const linearTicketButtonTitle = computed(() => isProjectDetailsLoading.value
  ? 'Loading Linear ticket'
  : linearTicketUrl.value === '' ? 'No Linear ticket found' : 'Open Linear ticket');
const pullRequestButtonTitle = computed(() => isProjectDetailsLoading.value
  ? 'Loading pull request'
  : pullRequestUrl.value === '' ? 'No pull request found' : 'Open pull request');
const gitHubButtonTitle = computed(() => isProjectDetailsLoading.value
  ? 'Loading GitHub page'
  : githubUrl.value === '' ? 'No GitHub remote found' : 'Open GitHub page');
const reloadButtonTitle = computed(() => isProjectDetailsLoading.value
  ? 'Loading project details'
  : 'Reload project details');
const gitBranchLabel = computed(() => {
  if (isProjectDetailsLoading.value) {
    return 'Loading branch';
  }

  return gitBranch.value || 'No branch';
});

const openExternalUrl = (url: string) => {
  if (url === '') {
    return;
  }

  window.open(url, '_blank', 'noopener,noreferrer');
};

const openLinearTicket = () => {
  openExternalUrl(linearTicketUrl.value);
};

const openPullRequest = () => {
  openExternalUrl(pullRequestUrl.value);
};

const openGitHubPage = () => {
  openExternalUrl(githubUrl.value);
};

const reloadProjectDetails = () => {
  void loadProjectDetails();
};

onMounted(() => {
  void loadProjectDetails();
});
</script>

<template>
  <header id="terminal-topbar">
    <h1 id="project-summary">
      <RouterLink id="brand" :to="{ name: 'home' }">WADE</RouterLink>
      <span id="project-name" :title="projectName">{{ projectDisplayName }}</span>
      <span id="git-branch" :title="gitBranchLabel">
        <GitBranch :size="14" :stroke-width="1.75" aria-hidden="true" />
        <span>{{ gitBranchLabel }}</span>
      </span>
    </h1>
    <section id="project-actions" aria-label="Project actions">
      <button
        class="project-reload-action"
        type="button"
        :aria-label="reloadButtonTitle"
        :disabled="isProjectDetailsLoading"
        :title="reloadButtonTitle"
        @click="reloadProjectDetails"
      >
        <RefreshCw :size="14" :stroke-width="1.7" aria-hidden="true" />
      </button>
      <button
        class="project-action"
        type="button"
        :disabled="isLinearTicketButtonDisabled"
        :title="linearTicketButtonTitle"
        @click="openLinearTicket"
      >
        <LinearIcon class="brand-icon" aria-hidden="true" />
        <span>Ticket</span>
      </button>
      <button
        class="project-action"
        type="button"
        :disabled="isPullRequestButtonDisabled"
        :title="pullRequestButtonTitle"
        @click="openPullRequest"
      >
        <GitHubIcon class="brand-icon" aria-hidden="true" />
        <span>PR</span>
      </button>
      <button
        class="project-action"
        type="button"
        :disabled="isGitHubButtonDisabled"
        :title="gitHubButtonTitle"
        @click="openGitHubPage"
      >
        <GitHubIcon class="brand-icon" aria-hidden="true" />
        <span>GitHub</span>
      </button>
      <span
        id="connection-status"
        role="status"
        aria-live="polite"
        :data-connected="String(isConnected)"
      >
        <span aria-hidden="true"></span>
        <span>{{ connectionStatusText }}</span>
      </span>
    </section>
  </header>
</template>

<style scoped>
#terminal-topbar {
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 0 16px;
  border-bottom: 1px solid var(--text);
  background: var(--window);
  color: var(--text);
  user-select: none;
}

#project-summary {
  min-width: 0;
  display: flex;
  margin: 0;
  align-items: center;
  gap: 14px;
  font-weight: 400;
}

#brand {
  margin: 0;
  color: var(--text);
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
  text-decoration: none;
}

#brand:hover,
#brand:focus-visible {
  text-decoration: underline;
}

#project-name {
  flex: 0 1 auto;
  margin: 0;
  overflow: hidden;
  color: var(--muted);
  font-size: 14px;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

#git-branch {
  min-width: 0;
  max-width: 45vw;
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  overflow: hidden;
  color: var(--text);
  font-size: 13px;
  line-height: 1;
  white-space: nowrap;
}

#git-branch svg {
  flex: 0 0 auto;
}

#git-branch span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

#project-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.project-action {
  height: 26px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 9px;
  border: 1px solid rgb(var(--accent-rgb) / 45%);
  border-radius: 999px;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}

.project-reload-action {
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.project-action:disabled,
.project-reload-action:disabled {
  color: var(--muted);
  cursor: not-allowed;
  opacity: 0.45;
}

.project-action:not(:disabled):hover,
.project-action:not(:disabled):focus-visible {
  background: rgb(var(--accent-rgb) / 10%);
}

.project-reload-action:not(:disabled):hover,
.project-reload-action:not(:disabled):focus-visible {
  color: var(--text);
}

.brand-icon {
  width: 13px;
  height: 13px;
  flex: 0 0 auto;
  fill: currentColor;
}

#connection-status {
  height: 26px;
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 9px;
  border: 1px solid rgb(var(--accent-rgb) / 45%);
  border-radius: 999px;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 12px;
  letter-spacing: 0.01em;
  line-height: 1;
}

#connection-status span:first-child {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--disconnected);
}

#connection-status[data-connected="true"] span:first-child {
  background: var(--connected);
}
</style>
