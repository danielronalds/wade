<script setup lang="ts">
import { Check, Copy, GitBranch, RefreshCw } from '@lucide/vue';
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { useWorkspaceLinkClipboard } from '@/views/workspace/composables/useWorkspaceLinkClipboard';
import { useSettingsStore } from '@/stores/useSettingsStore';
import { useWorkspaceDetailsStore } from '@/stores/useWorkspaceDetailsStore';
import GitHubIcon from '@/components/icons/GitHubIcon.vue';
import LinearIcon from '@/components/icons/LinearIcon.vue';
import { getWorkspacePresentation, type WorkspacePresentation } from '@/features/workspaces/workspacePresentation';

const props = defineProps<{
  workspaceId: string;
  connectionStatusText: string;
  isConnected: boolean;
}>();

const settingsStore = useSettingsStore();
const workspaceDetailsStore = useWorkspaceDetailsStore();
const { clipboardAnnouncement, copiedWorkspaceLink, copyWorkspaceLink } = useWorkspaceLinkClipboard();

const workspaceDetails = computed(() => workspaceDetailsStore.getWorkspaceDetails(props.workspaceId));
const isWorkspaceDetailsLoading = computed(() => workspaceDetailsStore.isWorkspaceDetailsLoading(props.workspaceId));
const isWaitingForWorkspaceDetails = computed(() => isWorkspaceDetailsLoading.value && !workspaceDetails.value);
const workspaceDetailsError = computed(() => workspaceDetailsStore.getWorkspaceDetailsError(props.workspaceId));
const isWorkspaceDetailsPending = computed(
  () => isWorkspaceDetailsLoading.value || (!workspaceDetails.value && workspaceDetailsError.value === '')
);
const isLinearEnabled = computed(() => settingsStore.settings.linear.enabled);
const githubUrl = computed(() => workspaceDetails.value?.links.repository ?? '');
const linearTicketUrl = computed(() => workspaceDetails.value?.links.issue?.url ?? '');
const pullRequestUrl = computed(() => workspaceDetails.value?.links.pullRequest ?? '');

const workspacePresentation = computed<WorkspacePresentation>(() => {
  if (isWorkspaceDetailsPending.value) {
    return {
      root: 'Loading workspace...',
      branch: '',
      title: 'Loading workspace...'
    };
  }

  if (workspaceDetailsError.value !== '' || !workspaceDetails.value) {
    return {
      root: props.workspaceId,
      branch: '',
      title: props.workspaceId
    };
  }

  return getWorkspacePresentation(workspaceDetails.value);
});

const workspaceDisplayName = computed(() => workspacePresentation.value.root);
const isLinearTicketButtonDisabled = computed(() => isWorkspaceDetailsLoading.value || linearTicketUrl.value === '');
const isPullRequestButtonDisabled = computed(() => pullRequestUrl.value === '');
const isGitHubButtonDisabled = computed(() => githubUrl.value === '');
const linearTicketButtonTitle = computed(() =>
  isWorkspaceDetailsLoading.value
    ? 'Loading Linear ticket'
    : linearTicketUrl.value === ''
      ? 'No Linear ticket found'
      : 'Open Linear ticket'
);
const linearTicketCopyButtonTitle = computed(() =>
  copiedWorkspaceLink.value === 'linear-ticket'
    ? 'Linear ticket link copied'
    : isWorkspaceDetailsLoading.value
      ? 'Loading Linear ticket'
      : linearTicketUrl.value === ''
        ? 'No Linear ticket found'
        : 'Copy Linear ticket link'
);
const pullRequestButtonTitle = computed(() =>
  isWaitingForWorkspaceDetails.value
    ? 'Loading pull request'
    : pullRequestUrl.value === ''
      ? 'No pull request found'
      : 'Open pull request'
);
const pullRequestCopyButtonTitle = computed(() =>
  copiedWorkspaceLink.value === 'pull-request'
    ? 'Pull request link copied'
    : isWaitingForWorkspaceDetails.value
      ? 'Loading pull request'
      : pullRequestUrl.value === ''
        ? 'No pull request found'
        : 'Copy pull request link'
);
const gitHubButtonTitle = computed(() =>
  isWaitingForWorkspaceDetails.value
    ? 'Loading GitHub page'
    : githubUrl.value === ''
      ? 'No GitHub remote found'
      : 'Open GitHub page'
);
const gitHubCopyButtonTitle = computed(() =>
  copiedWorkspaceLink.value === 'github'
    ? 'GitHub link copied'
    : isWaitingForWorkspaceDetails.value
      ? 'Loading GitHub page'
      : githubUrl.value === ''
        ? 'No GitHub remote found'
        : 'Copy GitHub link'
);
const reloadButtonTitle = computed(() =>
  isWorkspaceDetailsLoading.value ? 'Loading workspace details' : 'Reload workspace details'
);
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
  void copyWorkspaceLink('linear-ticket', 'Linear ticket', linearTicketUrl.value);
};

const copyPullRequestUrl = () => {
  void copyWorkspaceLink('pull-request', 'Pull request', pullRequestUrl.value);
};

const copyGitHubUrl = () => {
  void copyWorkspaceLink('github', 'GitHub', githubUrl.value);
};

const reloadWorkspaceDetails = () => {
  void workspaceDetailsStore.loadWorkspaceDetails(props.workspaceId);
};
</script>

<template>
  <header id="workspace-topbar">
    <h1 id="workspace-summary">
      <RouterLink id="brand" :to="{ name: 'home' }">WADE</RouterLink>
      <span id="workspace-presentation" :title="workspacePresentation.title">
        <span id="workspace-name">{{ workspaceDisplayName }}</span>
        <span v-if="workspacePresentation.branch !== ''" id="git-branch">
          <GitBranch :size="14" :stroke-width="1.75" aria-hidden="true" />
          <span>{{ workspacePresentation.branch }}</span>
        </span>
      </span>
    </h1>
    <section id="workspace-actions" aria-label="Workspace actions">
      <button
        class="workspace-reload-action"
        type="button"
        :aria-label="reloadButtonTitle"
        :disabled="isWorkspaceDetailsLoading"
        :title="reloadButtonTitle"
        @click="reloadWorkspaceDetails"
      >
        <RefreshCw :size="14" :stroke-width="1.7" aria-hidden="true" />
      </button>
      <span v-if="isLinearEnabled" class="workspace-action">
        <button
          class="workspace-action-button workspace-action-open"
          type="button"
          :disabled="isLinearTicketButtonDisabled"
          :title="linearTicketButtonTitle"
          @click="openLinearTicket"
        >
          <LinearIcon class="brand-icon" aria-hidden="true" />
          <span>Ticket</span>
        </button>
        <button
          class="workspace-action-button workspace-action-copy"
          type="button"
          :aria-label="linearTicketCopyButtonTitle"
          :disabled="isLinearTicketButtonDisabled"
          :title="linearTicketCopyButtonTitle"
          @click="copyLinearTicketUrl"
        >
          <Check v-if="copiedWorkspaceLink === 'linear-ticket'" class="copy-icon" aria-hidden="true" />
          <Copy v-else class="copy-icon" aria-hidden="true" />
        </button>
      </span>
      <span class="workspace-action">
        <button
          class="workspace-action-button workspace-action-open"
          type="button"
          :disabled="isPullRequestButtonDisabled"
          :title="pullRequestButtonTitle"
          @click="openPullRequest"
        >
          <GitHubIcon class="brand-icon" aria-hidden="true" />
          <span>PR</span>
        </button>
        <button
          class="workspace-action-button workspace-action-copy"
          type="button"
          :aria-label="pullRequestCopyButtonTitle"
          :disabled="isPullRequestButtonDisabled"
          :title="pullRequestCopyButtonTitle"
          @click="copyPullRequestUrl"
        >
          <Check v-if="copiedWorkspaceLink === 'pull-request'" class="copy-icon" aria-hidden="true" />
          <Copy v-else class="copy-icon" aria-hidden="true" />
        </button>
      </span>
      <span class="workspace-action">
        <button
          class="workspace-action-button workspace-action-open"
          type="button"
          :disabled="isGitHubButtonDisabled"
          :title="gitHubButtonTitle"
          @click="openGitHubPage"
        >
          <GitHubIcon class="brand-icon" aria-hidden="true" />
          <span>GitHub</span>
        </button>
        <button
          class="workspace-action-button workspace-action-copy"
          type="button"
          :aria-label="gitHubCopyButtonTitle"
          :disabled="isGitHubButtonDisabled"
          :title="gitHubCopyButtonTitle"
          @click="copyGitHubUrl"
        >
          <Check v-if="copiedWorkspaceLink === 'github'" class="copy-icon" aria-hidden="true" />
          <Copy v-else class="copy-icon" aria-hidden="true" />
        </button>
      </span>
      <span class="visually-hidden" role="status" aria-live="polite">{{ clipboardAnnouncement }}</span>
      <span id="connection-status" role="status" aria-live="polite" :data-connected="String(isConnected)">
        <span aria-hidden="true"></span>
        <span>{{ connectionStatusText }}</span>
      </span>
    </section>
  </header>
</template>

<style scoped>
#workspace-topbar {
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

#workspace-summary {
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

#workspace-presentation {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  color: var(--text);
  font-size: 14px;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

#workspace-name {
  color: var(--text);
}

#git-branch {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 12px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1;
  vertical-align: middle;
}

#git-branch svg {
  flex: 0 0 auto;
}

#workspace-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.workspace-action {
  height: 26px;
  display: flex;
  align-items: stretch;
  overflow: hidden;
  border: 1px solid rgb(var(--accent-rgb) / 45%);
  border-radius: 999px;
  background: transparent;
  color: var(--text);
}

.workspace-action-button {
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

.workspace-action-open {
  gap: 6px;
  padding: 0 7px 0 9px;
}

.workspace-action-copy {
  width: 25px;
  justify-content: center;
  padding: 0;
  border-left: 1px solid rgb(var(--accent-rgb) / 35%);
}

.workspace-reload-action {
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

.workspace-action-button:disabled,
.workspace-reload-action:disabled {
  color: var(--muted);
  cursor: not-allowed;
  opacity: 0.45;
}

.workspace-action-button:not(:disabled):hover,
.workspace-action-button:not(:disabled):focus-visible {
  background: rgb(var(--accent-rgb) / 10%);
}

.workspace-action-button:focus-visible {
  outline: 1px solid var(--text);
  outline-offset: -2px;
}

.workspace-reload-action:not(:disabled):hover,
.workspace-reload-action:not(:disabled):focus-visible {
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

#connection-status[data-connected='true'] span:first-child {
  background: var(--connected);
}
</style>
