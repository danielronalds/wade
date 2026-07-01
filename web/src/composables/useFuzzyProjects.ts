import { computed, readonly, type Ref } from 'vue';

export type ProjectMatch = {
  projectName: string;
  score: number;
};

export type FuzzyMatch<T> = {
  item: T;
  label: string;
  score: number;
};

type SequentialMatch = {
  consecutiveMatches: number;
  firstMatchIndex: number;
  gapPenalty: number;
};

const findSequentialMatch = (candidate: string, query: string): SequentialMatch | undefined => {
  let queryIndex = 0;
  let firstMatchIndex = -1;
  let lastMatchIndex = -1;
  let consecutiveMatches = 0;
  let gapPenalty = 0;

  for (let candidateIndex = 0; candidateIndex < candidate.length; candidateIndex += 1) {
    if (candidate[candidateIndex] !== query[queryIndex]) {
      continue;
    }

    if (firstMatchIndex === -1) {
      firstMatchIndex = candidateIndex;
    }

    if (lastMatchIndex !== -1) {
      const gap = candidateIndex - lastMatchIndex - 1;
      gapPenalty += gap;

      if (gap === 0) {
        consecutiveMatches += 1;
      }
    }

    lastMatchIndex = candidateIndex;
    queryIndex += 1;

    if (queryIndex === query.length) {
      return {
        consecutiveMatches,
        firstMatchIndex,
        gapPenalty
      };
    }
  }

  return undefined;
};

const scoreLabel = <T>(item: T, label: string, rawQuery: string): FuzzyMatch<T> | undefined => {
  const query = rawQuery.trim().toLowerCase();
  if (query === '') {
    return { item, label, score: 0 };
  }

  const candidate = label.toLowerCase();
  if (candidate === query) {
    return { item, label, score: 100000 - label.length };
  }

  if (candidate.startsWith(query)) {
    return { item, label, score: 90000 - label.length };
  }

  const substringIndex = candidate.indexOf(query);
  if (substringIndex !== -1) {
    return { item, label, score: 80000 - (substringIndex * 100) - label.length };
  }

  const sequentialMatch = findSequentialMatch(candidate, query);
  if (!sequentialMatch) {
    return undefined;
  }

  return {
    item,
    label,
    score: 70000
      + (sequentialMatch.consecutiveMatches * 25)
      - (sequentialMatch.gapPenalty * 10)
      - (sequentialMatch.firstMatchIndex * 20)
      - label.length
  };
};

export const useFuzzyItems = <T>(
  items: Ref<readonly T[]>,
  query: Ref<string>,
  getLabel: (item: T) => string
) => {
  const matchingItems = computed(() => items.value
    .map((item) => scoreLabel(item, getLabel(item), query.value))
    .filter((match): match is FuzzyMatch<T> => Boolean(match))
    .sort((firstMatch, secondMatch) => {
      if (firstMatch.score !== secondMatch.score) {
        return secondMatch.score - firstMatch.score;
      }

      return firstMatch.label.localeCompare(secondMatch.label);
    }));

  return {
    matchingItems: readonly(matchingItems)
  };
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
