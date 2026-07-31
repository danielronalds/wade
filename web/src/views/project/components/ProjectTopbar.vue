<script setup lang="ts">
import { Check, Copy, GitBranch, RefreshCw } from '@lucide/vue';
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { useProjectLinkClipboard } from '@/views/project/composables/useProjectLinkClipboard';
import { useProjectDetailsStore } from '@/stores/useProjectDetailsStore';
import GitHubIcon from '@/components/icons/GitHubIcon.vue';
import LinearIcon from '@/components/icons/LinearIcon.vue';

const props = defineProps<{
  projectName: string;
  connectionStatusText: string;
  isConnected: boolean;
}>();

const projectDetailsStore = useProjectDetailsStore();
const {
  clipboardAnnouncement,
  copiedProjectLink,
  copyProjectLink
} = useProjectLinkClipboard();

const projectDetails = computed(() => projectDetailsStore.getProjectDetails(props.projectName));
const isProjectDetailsLoading = computed(() => projectDetailsStore.isProjectDetailsLoading(props.projectName));
const isWaitingForProjectDetails = computed(() => isProjectDetailsLoading.value && !projectDetails.value);
const gitBranch = computed(() => projectDetails.value?.gitBranch ?? '');
const githubUrl = computed(() => projectDetails.value?.githubUrl ?? '');
const linearTicketUrl = computed(() => projectDetails.value?.linearTicketUrl ?? '');
const pullRequestUrl = computed(() => projectDetails.value?.pullRequestUrl ?? '');

const projectDisplayName = computed(() => props.projectName.split('-feature')[0] || props.projectName);
const isLinearTicketButtonDisabled = computed(() => linearTicketUrl.value === '');
const isPullRequestButtonDisabled = computed(() => pullRequestUrl.value === '');
const isGitHubButtonDisabled = computed(() => githubUrl.value === '');
const linearTicketButtonTitle = computed(() => isWaitingForProjectDetails.value
  ? 'Loading Linear ticket'
  : linearTicketUrl.value === '' ? 'No Linear ticket found' : 'Open Linear ticket');
const linearTicketCopyButtonTitle = computed(() => copiedProjectLink.value === 'linear-ticket'
  ? 'Linear ticket link copied'
  : isWaitingForProjectDetails.value
    ? 'Loading Linear ticket'
    : linearTicketUrl.value === '' ? 'No Linear ticket found' : 'Copy Linear ticket link');
const pullRequestButtonTitle = computed(() => isWaitingForProjectDetails.value
  ? 'Loading pull request'
  : pullRequestUrl.value === '' ? 'No pull request found' : 'Open pull request');
const pullRequestCopyButtonTitle = computed(() => copiedProjectLink.value === 'pull-request'
  ? 'Pull request link copied'
  : isWaitingForProjectDetails.value
    ? 'Loading pull request'
    : pullRequestUrl.value === '' ? 'No pull request found' : 'Copy pull request link');
const gitHubButtonTitle = computed(() => isWaitingForProjectDetails.value
  ? 'Loading GitHub page'
  : githubUrl.value === '' ? 'No GitHub remote found' : 'Open GitHub page');
const gitHubCopyButtonTitle = computed(() => copiedProjectLink.value === 'github'
  ? 'GitHub link copied'
  : isWaitingForProjectDetails.value
    ? 'Loading GitHub page'
    : githubUrl.value === '' ? 'No GitHub remote found' : 'Copy GitHub link');
const reloadButtonTitle = computed(() => isProjectDetailsLoading.value
  ? 'Loading project details'
  : 'Reload project details');
const gitBranchLabel = computed(() => {
  if (isWaitingForProjectDetails.value) {
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

const copyLinearTicketUrl = () => {
  void copyProjectLink('linear-ticket', 'Linear ticket', linearTicketUrl.value);
};

const copyPullRequestUrl = () => {
  void copyProjectLink('pull-request', 'Pull request', pullRequestUrl.value);
};

const copyGitHubUrl = () => {
  void copyProjectLink('github', 'GitHub', githubUrl.value);
};

const reloadProjectDetails = () => {
  void projectDetailsStore.loadProjectDetails(props.projectName);
};
</script>

<template>
  <header id="project-topbar">
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
      <span class="project-action">
        <button
          class="project-action-button project-action-open"
          type="button"
          :disabled="isLinearTicketButtonDisabled"
          :title="linearTicketButtonTitle"
          @click="openLinearTicket"
        >
          <LinearIcon class="brand-icon" aria-hidden="true" />
          <span>Ticket</span>
        </button>
        <button
          class="project-action-button project-action-copy"
          type="button"
          :aria-label="linearTicketCopyButtonTitle"
          :disabled="isLinearTicketButtonDisabled"
          :title="linearTicketCopyButtonTitle"
          @click="copyLinearTicketUrl"
        >
          <Check v-if="copiedProjectLink === 'linear-ticket'" class="copy-icon" aria-hidden="true" />
          <Copy v-else class="copy-icon" aria-hidden="true" />
        </button>
      </span>
      <span class="project-action">
        <button
          class="project-action-button project-action-open"
          type="button"
          :disabled="isPullRequestButtonDisabled"
          :title="pullRequestButtonTitle"
          @click="openPullRequest"
        >
          <GitHubIcon class="brand-icon" aria-hidden="true" />
          <span>PR</span>
        </button>
        <button
          class="project-action-button project-action-copy"
          type="button"
          :aria-label="pullRequestCopyButtonTitle"
          :disabled="isPullRequestButtonDisabled"
          :title="pullRequestCopyButtonTitle"
          @click="copyPullRequestUrl"
        >
          <Check v-if="copiedProjectLink === 'pull-request'" class="copy-icon" aria-hidden="true" />
          <Copy v-else class="copy-icon" aria-hidden="true" />
        </button>
      </span>
      <span class="project-action">
        <button
          class="project-action-button project-action-open"
          type="button"
          :disabled="isGitHubButtonDisabled"
          :title="gitHubButtonTitle"
          @click="openGitHubPage"
        >
          <GitHubIcon class="brand-icon" aria-hidden="true" />
          <span>GitHub</span>
        </button>
        <button
          class="project-action-button project-action-copy"
          type="button"
          :aria-label="gitHubCopyButtonTitle"
          :disabled="isGitHubButtonDisabled"
          :title="gitHubCopyButtonTitle"
          @click="copyGitHubUrl"
        >
          <Check v-if="copiedProjectLink === 'github'" class="copy-icon" aria-hidden="true" />
          <Copy v-else class="copy-icon" aria-hidden="true" />
        </button>
      </span>
      <span class="visually-hidden" role="status" aria-live="polite">{{ clipboardAnnouncement }}</span>
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
#project-topbar {
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
  align-items: stretch;
  overflow: hidden;
  border: 1px solid rgb(var(--accent-rgb) / 45%);
  border-radius: 999px;
  background: transparent;
  color: var(--text);
}

.project-action-button {
  display: flex;
  align-items: center;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}

.project-action-open {
  gap: 6px;
  padding: 0 7px 0 9px;
}

.project-action-copy {
  width: 25px;
  justify-content: center;
  padding: 0;
  border-left: 1px solid rgb(var(--accent-rgb) / 35%);
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

.project-action-button:disabled,
.project-reload-action:disabled {
  color: var(--muted);
  cursor: not-allowed;
  opacity: 0.45;
}

.project-action-button:not(:disabled):hover,
.project-action-button:not(:disabled):focus-visible {
  background: rgb(var(--accent-rgb) / 10%);
}

.project-action-button:focus-visible {
  outline: 1px solid var(--text);
  outline-offset: -2px;
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

.copy-icon {
  width: 12px;
  height: 12px;
}

.visually-hidden {
  width: 1px;
  height: 1px;
  position: absolute;
  overflow: hidden;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
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
