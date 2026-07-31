import { defineStore } from 'pinia';
import { reactive } from 'vue';
import {
  getProjectDetails as requestProjectDetails,
  type HandlersProjectResponse
} from '@/api/generated/wade';

type ProjectDetails = Readonly<HandlersProjectResponse>;

type ProjectDetailsState = {
  details?: ProjectDetails;
  error: string;
  isLoading: boolean;
};

const errorMessage = (error: unknown) => error instanceof Error
  ? error.message
  : 'Project details request failed';

export const useProjectDetailsStore = defineStore('project-details', () => {
  const projectStates = reactive(new Map<string, ProjectDetailsState>());
  const loadRequests = new Map<string, Promise<ProjectDetails | undefined>>();

  const ensureProjectState = (projectName: string) => {
    const existingState = projectStates.get(projectName);
    if (existingState) {
      return existingState;
    }

    projectStates.set(projectName, {
      error: '',
      isLoading: false
    });

    return projectStates.get(projectName)!;
  };

  const getProjectDetails = (projectName: string) => projectStates.get(projectName)?.details;
  const getProjectDetailsError = (projectName: string) => projectStates.get(projectName)?.error ?? '';
  const isProjectDetailsLoading = (projectName: string) => projectStates.get(projectName)?.isLoading ?? false;

  const loadProjectDetails = (projectName: string) => {
    const activeRequest = loadRequests.get(projectName);
    if (activeRequest) {
      return activeRequest;
    }

    const projectState = ensureProjectState(projectName);
    projectState.error = '';
    projectState.isLoading = true;

    const loadRequest = (async () => {
      try {
        const details = await requestProjectDetails({ project: projectName });
        projectState.details = details;

        return details;
      } catch (requestError) {
        projectState.error = errorMessage(requestError);
        return undefined;
      } finally {
        projectState.isLoading = false;
        loadRequests.delete(projectName);
      }
    })();

    loadRequests.set(projectName, loadRequest);
    return loadRequest;
  };

  return {
    getProjectDetails,
    getProjectDetailsError,
    isProjectDetailsLoading,
    loadProjectDetails
  };
});
