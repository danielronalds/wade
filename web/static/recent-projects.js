const recentProjectsStorageKey = 'web-terminal:recent-projects';
const recentProjectsLimit = 5;

export const readRecentProjects = () => {
  try {
    const storedProjects = JSON.parse(localStorage.getItem(recentProjectsStorageKey) || '[]');
    if (!Array.isArray(storedProjects)) {
      return [];
    }

    return storedProjects
      .filter((project) => typeof project === 'string' && project.length > 0)
      .slice(0, recentProjectsLimit);
  } catch {
    return [];
  }
};

export const recordRecentProject = (projectName) => {
  const projects = readRecentProjects();
  const nextProjects = [projectName, ...projects.filter((project) => project !== projectName)];
  localStorage.setItem(recentProjectsStorageKey, JSON.stringify(nextProjects.slice(0, recentProjectsLimit)));
};
