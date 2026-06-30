import { computed, readonly, type Ref } from 'vue';

export type ProjectMatch = {
  projectName: string;
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

const scoreProject = (projectName: string, rawQuery: string): ProjectMatch | undefined => {
  const query = rawQuery.trim().toLowerCase();
  if (query === '') {
    return { projectName, score: 0 };
  }

  const candidate = projectName.toLowerCase();
  if (candidate === query) {
    return { projectName, score: 100000 - projectName.length };
  }

  if (candidate.startsWith(query)) {
    return { projectName, score: 90000 - projectName.length };
  }

  const substringIndex = candidate.indexOf(query);
  if (substringIndex !== -1) {
    return { projectName, score: 80000 - (substringIndex * 100) - projectName.length };
  }

  const sequentialMatch = findSequentialMatch(candidate, query);
  if (!sequentialMatch) {
    return undefined;
  }

  return {
    projectName,
    score: 70000
      + (sequentialMatch.consecutiveMatches * 25)
      - (sequentialMatch.gapPenalty * 10)
      - (sequentialMatch.firstMatchIndex * 20)
      - projectName.length
  };
};

export const useFuzzyProjects = (projects: Ref<readonly string[]>, query: Ref<string>) => {
  const matchingProjects = computed(() => projects.value
    .map((projectName) => scoreProject(projectName, query.value))
    .filter((match): match is ProjectMatch => Boolean(match))
    .sort((firstMatch, secondMatch) => {
      if (firstMatch.score !== secondMatch.score) {
        return secondMatch.score - firstMatch.score;
      }

      return firstMatch.projectName.localeCompare(secondMatch.projectName);
    }));

  return {
    matchingProjects: readonly(matchingProjects)
  };
};
