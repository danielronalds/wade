import { createHash } from 'node:crypto';
import { cp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compileScript, compileStyle, compileTemplate, parse } from '@vue/compiler-sfc';
import * as esbuild from 'esbuild';

const root = dirname(fileURLToPath(import.meta.url));
const distDir = join(root, '..', 'internal', 'web', '.dist');
const staticDir = join(distDir, 'static');

const createVueDescriptorId = (filePath) =>
  createHash('sha256').update(relative(root, filePath)).digest('hex').slice(0, 8);

const formatVueError = (error) => (error instanceof Error ? error.message : String(error));

const splitVueRequest = (request) => {
  const [filePath, query = ''] = request.split('?');

  return {
    filePath,
    params: new URLSearchParams(query)
  };
};

const vuePlugin = {
  name: 'vue',
  setup(build) {
    build.onLoad({ filter: /\.vue$/ }, async (args) => {
      const source = await readFile(args.path, 'utf8');
      const { descriptor, errors } = parse(source, { filename: args.path });

      if (errors.length > 0) {
        throw new Error(errors.map(formatVueError).join('\n'));
      }

      const id = createVueDescriptorId(args.path);
      const scopeId = `data-v-${id}`;
      const hasScopedStyles = descriptor.styles.some((style) => style.scoped);
      const styleImports = descriptor.styles
        .map((_, index) => `import ${JSON.stringify(`${args.path}?vue&type=style&index=${index}`)};`)
        .join('\n');
      const script =
        descriptor.script || descriptor.scriptSetup
          ? compileScript(descriptor, {
              id: scopeId,
              genDefaultAs: '__sfc__'
            })
          : {
              bindings: {},
              content: 'const __sfc__ = {}'
            };
      const template = descriptor.template
        ? compileTemplate({
            id: scopeId,
            source: descriptor.template.content,
            filename: args.path,
            scoped: hasScopedStyles,
            isProd: true,
            compilerOptions: {
              bindingMetadata: script.bindings
            }
          })
        : undefined;

      if (template?.errors.length) {
        throw new Error(template.errors.map(formatVueError).join('\n'));
      }

      const renderCode = template?.code.replace('export function render', 'function render') ?? '';
      const renderAssignment = template ? '__sfc__.render = render;' : '';
      const scopeAssignment = hasScopedStyles ? `__sfc__.__scopeId = ${JSON.stringify(scopeId)};` : '';

      return {
        contents: [
          script.content,
          renderCode,
          renderAssignment,
          scopeAssignment,
          styleImports,
          'export default __sfc__;'
        ]
          .filter(Boolean)
          .join('\n'),
        loader: 'ts',
        resolveDir: dirname(args.path)
      };
    });

    build.onResolve({ filter: /\.vue\?vue&type=style/ }, (args) => ({
      path: args.path,
      namespace: 'vue-style'
    }));

    build.onLoad({ filter: /.*/, namespace: 'vue-style' }, async (args) => {
      const { filePath, params } = splitVueRequest(args.path);
      const source = await readFile(filePath, 'utf8');
      const { descriptor, errors } = parse(source, { filename: filePath });
      const styleIndex = Number(params.get('index'));
      const style = descriptor.styles[styleIndex];

      if (errors.length > 0) {
        throw new Error(errors.map(formatVueError).join('\n'));
      }

      if (!style) {
        throw new Error(`Missing Vue style block ${styleIndex} in ${filePath}`);
      }

      if (style.lang && style.lang !== 'css') {
        throw new Error(`Unsupported Vue style language ${style.lang} in ${filePath}`);
      }

      if (style.module) {
        throw new Error(`Vue CSS modules are not supported in ${filePath}`);
      }

      const id = createVueDescriptorId(filePath);
      const result = compileStyle({
        id: `data-v-${id}`,
        filename: filePath,
        source: style.content,
        scoped: style.scoped
      });

      if (result.errors.length > 0) {
        throw new Error(result.errors.map(formatVueError).join('\n'));
      }

      return {
        contents: result.code,
        loader: 'css',
        resolveDir: dirname(filePath)
      };
    });
  }
};

const writeIndexHtml = () =>
  writeFile(
    join(distDir, 'index.html'),
    `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="theme-color" content="#17181c">
  <title>WADE</title>
  <link rel="icon" href="/static/favicon.ico" type="image/x-icon" sizes="any">
  <link rel="icon" href="/static/favicon-32x32.png" type="image/png" sizes="32x32">
  <link rel="icon" href="/static/favicon-16x16.png" type="image/png" sizes="16x16">
  <link rel="apple-touch-icon" href="/static/apple-touch-icon.png">
  <link rel="manifest" href="/static/site.webmanifest">
  <link rel="stylesheet" href="/static/app.css">
  <script src="/static/app.js" type="module"></script>
</head>
<body>
  <div id="app"></div>
</body>
</html>
`
  );

const main = async () => {
  await rm(distDir, { recursive: true, force: true });
  await mkdir(staticDir, { recursive: true });
  await cp(join(root, 'static'), staticDir, { recursive: true });

  const fallbackPage = await readFile(join(root, 'static', 'server-unavailable.html'));
  const serviceWorkerSource = await readFile(join(root, 'service-worker.js'), 'utf8');
  const fallbackPageVersion = createHash('sha256').update(fallbackPage).digest('hex').slice(0, 12);
  const serviceWorker = serviceWorkerSource.replace('__FALLBACK_PAGE_VERSION__', fallbackPageVersion);

  await writeFile(join(distDir, 'service-worker.js'), serviceWorker);
  await mkdir(join(staticDir, 'monaco'), { recursive: true });
  await cp(join(root, 'node_modules', 'monaco-editor', 'min', 'vs'), join(staticDir, 'monaco', 'vs'), {
    recursive: true
  });

  await esbuild.build({
    entryPoints: [join(root, 'src/main.ts')],
    bundle: true,
    alias: {
      '@': join(root, 'src')
    },
    platform: 'browser',
    format: 'esm',
    target: ['es2022'],
    external: ['/static/*'],
    outdir: staticDir,
    entryNames: 'app',
    assetNames: 'assets/[name]-[hash]',
    sourcemap: true,
    legalComments: 'linked',
    loader: {
      '.ttf': 'file',
      '.woff': 'file',
      '.woff2': 'file'
    },
    define: {
      'process.env.NODE_ENV': '"production"',
      __VUE_OPTIONS_API__: 'false',
      __VUE_PROD_DEVTOOLS__: 'false',
      __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'false'
    },
    plugins: [vuePlugin]
  });

  await writeIndexHtml();
};

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
