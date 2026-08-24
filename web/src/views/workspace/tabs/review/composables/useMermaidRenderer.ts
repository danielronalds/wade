import type { Mermaid, MermaidConfig } from 'mermaid';

let mermaidModulePromise: Promise<Mermaid> | null = null;
let mermaidRenderQueue = Promise.resolve();
let nextMermaidDiagramId = 0;

const loadMermaid = () => {
  mermaidModulePromise ??= import('mermaid').then(({ default: mermaid }) => mermaid);
  return mermaidModulePromise;
};

const getThemeVariable = (name: string, fallback: string) => {
  if (typeof document === 'undefined') {
    return fallback;
  }

  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
};

const createMermaidConfig = (): MermaidConfig => {
  const windowColour = getThemeVariable('--window', '#17181c');
  const accentColour = getThemeVariable('--accent', '#f8f8f2');
  const textColour = getThemeVariable('--text', accentColour);

  return {
    darkMode: true,
    fontFamily: 'monospace',
    htmlLabels: false,
    securityLevel: 'strict',
    startOnLoad: false,
    suppressErrorRendering: true,
    theme: 'base',
    themeVariables: {
      actorBkg: windowColour,
      actorBorder: accentColour,
      actorLineColor: accentColour,
      actorTextColor: textColour,
      background: windowColour,
      clusterBkg: windowColour,
      clusterBorder: accentColour,
      classText: textColour,
      defaultLinkColor: accentColour,
      edgeLabelBackground: windowColour,
      lineColor: accentColour,
      labelBoxBkgColor: windowColour,
      labelBoxBorderColor: accentColour,
      labelTextColor: textColour,
      mainBkg: windowColour,
      nodeBorder: accentColour,
      nodeTextColor: textColour,
      primaryBorderColor: accentColour,
      primaryColor: windowColour,
      primaryTextColor: textColour,
      relationLabelBackground: windowColour,
      relationLabelColor: textColour,
      secondaryBorderColor: accentColour,
      secondaryColor: windowColour,
      secondaryTextColor: textColour,
      signalColor: accentColour,
      signalTextColor: textColour,
      tertiaryBorderColor: accentColour,
      tertiaryColor: windowColour,
      tertiaryTextColor: textColour,
      textColor: textColour,
      titleColor: textColour
    }
  };
};

const renderMermaidDiagram = async (source: string) => {
  const mermaid = await loadMermaid();
  const renderPromise = mermaidRenderQueue.then(async () => {
    const diagramId = `wade-mermaid-${nextMermaidDiagramId++}`;
    mermaid.initialize(createMermaidConfig());
    const { svg } = await mermaid.render(diagramId, source);
    return svg;
  });
  mermaidRenderQueue = renderPromise.then(
    () => undefined,
    () => undefined
  );

  return renderPromise;
};

export const useMermaidRenderer = () => ({ renderMermaidDiagram });
