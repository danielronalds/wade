import { computed, readonly, type Ref } from 'vue';
import { useFuzzyItems } from '@/composables/useFuzzyItems';

export type ProjectMatch = {
  projectName: string;
  score: number;
};

export const useFuzzyProjects = (projects: Ref<readonly string[]>, query: Ref<string>) => {
  const { matchingItems } = useFuzzyItems(projects, query, (projectName) => projectName);
  const matchingProjects = computed(() => matchingItems.value.map((match) => ({
    projectName: match.item,
    score: match.score
  })));

  return {
    matchingProjects: readonly(matchingProjects)
  };
};
