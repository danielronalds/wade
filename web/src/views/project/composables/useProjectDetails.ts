import { readonly, ref } from 'vue';
import { getProjectDetails } from '@/api/generated/wade';

export type ProjectDetails = {
  name: string;
  gitBranch: string;
  linearTicketUrl: string;
  pullRequestUrl: string;
  githubUrl: string;
};

export const useProjectDetails = (projectName: string) => {
  const gitBranch = ref('');
  const linearTicketUrl = ref('');
  const pullRequestUrl = ref('');
  const githubUrl = ref('');
  const isLoading = ref(false);
  const error = ref('');

  const loadProjectDetails = async () => {
    isLoading.value = true;
    error.value = '';

    try {
      const details = await getProjectDetails({ project: projectName });

      gitBranch.value = details.gitBranch;
      linearTicketUrl.value = details.linearTicketUrl;
      pullRequestUrl.value = details.pullRequestUrl;
      githubUrl.value = details.githubUrl;
    } catch (requestError) {
      error.value = requestError instanceof Error ? requestError.message : 'Project details request failed';
      gitBranch.value = '';
      linearTicketUrl.value = '';
      pullRequestUrl.value = '';
      githubUrl.value = '';
    } finally {
      isLoading.value = false;
    }
  };

  return {
    error: readonly(error),
    githubUrl: readonly(githubUrl),
    gitBranch: readonly(gitBranch),
    isLoading: readonly(isLoading),
    linearTicketUrl: readonly(linearTicketUrl),
    pullRequestUrl: readonly(pullRequestUrl),
    loadProjectDetails
  };
};
