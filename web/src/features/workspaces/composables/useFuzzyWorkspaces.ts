import { computed, readonly, type Ref } from 'vue';
import type { WorkspaceSummary } from '@/api/generated/wade';
import { useFuzzyItems } from '@/composables/useFuzzyItems';

export const useFuzzyWorkspaces = (workspaces: Ref<readonly WorkspaceSummary[]>, query: Ref<string>) => {
  const { matchingItems } = useFuzzyItems(workspaces, query, (workspace) => workspace.name);
  const matchingWorkspaces = computed(() =>
    matchingItems.value.map((match) => ({
      score: match.score,
      workspace: match.item
    }))
  );

  return {
    matchingWorkspaces: readonly(matchingWorkspaces)
  };
};
