import { readonly, ref } from 'vue';

type ProjectDetails = {
  name: string;
  gitBranch: string;
  linearTicketUrl: string;
  pullRequestUrl: string;
};

const isProjectDetails = (value: unknown): value is ProjectDetails => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const details = value as Partial<ProjectDetails>;

  return typeof details.name === 'string'
    && typeof details.gitBranch === 'string'
    && typeof details.linearTicketUrl === 'string'
    && typeof details.pullRequestUrl === 'string';
};

export const useProjectDetails = (projectName: string) => {
  const gitBranch = ref('');
  const linearTicketUrl = ref('');
  const pullRequestUrl = ref('');
  const isLoading = ref(false);
  const error = ref('');

  const loadProjectDetails = async () => {
    isLoading.value = true;
    error.value = '';

    try {
      const params = new URLSearchParams({ project: projectName });
      const response = await fetch(`/api/project?${params}`);

      if (!response.ok) {
        throw new Error(`Project details request failed with ${response.status}`);
      }

      const details: unknown = await response.json();

      if (!isProjectDetails(details)) {
        throw new Error('Project details response was invalid');
      }

      gitBranch.value = details.gitBranch;
      linearTicketUrl.value = details.linearTicketUrl;
      pullRequestUrl.value = details.pullRequestUrl;
    } catch (requestError) {
      error.value = requestError instanceof Error ? requestError.message : 'Project details request failed';
      gitBranch.value = '';
      linearTicketUrl.value = '';
      pullRequestUrl.value = '';
    } finally {
      isLoading.value = false;
    }
  };

  return {
    error: readonly(error),
    gitBranch: readonly(gitBranch),
    isLoading: readonly(isLoading),
    linearTicketUrl: readonly(linearTicketUrl),
    pullRequestUrl: readonly(pullRequestUrl),
    loadProjectDetails
  };
};
