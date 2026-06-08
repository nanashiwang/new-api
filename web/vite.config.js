/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import react from '@vitejs/plugin-react';
import { defineConfig, transformWithEsbuild } from 'vite';
import pkg from '@douyinfe/vite-plugin-semi';
import path from 'path';
import { codeInspectorPlugin } from 'code-inspector-plugin';
const { vitePluginSemi } = pkg;

const nodeModuleId = (id, name) =>
  id.includes(`/node_modules/${name}/`) ||
  id.includes(`/node_modules/${name}\\`);

const scopedNodeModuleId = (id, scope, name) =>
  id.includes(`/node_modules/${scope}/${name}/`) ||
  id.includes(`/node_modules/${scope}\\${name}\\`);

const manualChunks = (id) => {
  if (id.includes('vite/preload-helper')) {
    return 'preload-helper';
  }

  if (id.includes('commonjsHelpers.js')) {
    return 'commonjs-helpers';
  }

  if (!id.includes('node_modules')) {
    return;
  }

  if (scopedNodeModuleId(id, '@babel', 'runtime')) {
    return 'babel-runtime';
  }

  if (
    nodeModuleId(id, 'react') ||
    nodeModuleId(id, 'react-dom') ||
    nodeModuleId(id, 'react-router-dom')
  ) {
    return 'react-core';
  }

  if (
    scopedNodeModuleId(id, '@douyinfe', 'semi-icons') ||
    scopedNodeModuleId(id, '@douyinfe', 'semi-ui')
  ) {
    return 'semi-ui';
  }

  if (nodeModuleId(id, 'axios') || nodeModuleId(id, 'history')) {
    return 'tools';
  }

  if (nodeModuleId(id, 'lucide-react')) {
    return 'icons';
  }

  if (nodeModuleId(id, 'marked')) {
    return 'marked';
  }

  if (nodeModuleId(id, 'react-toastify')) {
    return 'toastify';
  }

  if (nodeModuleId(id, 'react-fireworks')) {
    return 'fireworks';
  }

  if (
    nodeModuleId(id, 'react-dropzone') ||
    nodeModuleId(id, 'react-telegram-login') ||
    nodeModuleId(id, 'react-turnstile')
  ) {
    return 'react-integrations';
  }

  if (
    nodeModuleId(id, 'i18next') ||
    nodeModuleId(id, 'react-i18next') ||
    nodeModuleId(id, 'i18next-browser-languagedetector')
  ) {
    return 'i18n';
  }

  if (nodeModuleId(id, 'mermaid')) {
    return 'mermaid';
  }

  if (
    scopedNodeModuleId(id, '@visactor', 'vchart') ||
    scopedNodeModuleId(id, '@visactor', 'react-vchart') ||
    scopedNodeModuleId(id, '@visactor', 'vchart-semi-theme')
  ) {
    return 'vchart';
  }

  if (
    nodeModuleId(id, 'react-markdown') ||
    nodeModuleId(id, 'remark-gfm') ||
    nodeModuleId(id, 'remark-math') ||
    nodeModuleId(id, 'remark-breaks') ||
    nodeModuleId(id, 'rehype-highlight') ||
    nodeModuleId(id, 'rehype-katex')
  ) {
    return 'markdown';
  }

  if (nodeModuleId(id, 'katex')) {
    return 'katex';
  }
};

const shouldPreloadFromHtml = (file) =>
  /^assets\/(react-core|tools|preload-helper)-/.test(file);

const stripNonCriticalHtmlAssets = () => ({
  name: 'strip-non-critical-html-assets',
  apply: 'build',
  enforce: 'post',
  generateBundle(_, bundle) {
    for (const asset of Object.values(bundle)) {
      if (asset.type !== 'asset' || !asset.fileName.endsWith('.html')) {
        continue;
      }
      asset.source = String(asset.source).replace(
        /\s*<link rel="stylesheet" crossorigin href="\/assets\/(?:katex|MarkdownRenderer)-[^"]+\.css">/g,
        '',
      );
    }
  },
});

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  plugins: [
    codeInspectorPlugin({
      bundler: 'vite',
    }),
    {
      name: 'treat-js-files-as-jsx',
      async transform(code, id) {
        if (!/src\/.*\.js$/.test(id)) {
          return null;
        }

        // Use the exposed transform from vite, instead of directly
        // transforming with esbuild
        return transformWithEsbuild(code, id, {
          loader: 'jsx',
          jsx: 'automatic',
        });
      },
    },
    react(),
    vitePluginSemi({
      cssLayer: true,
    }),
    stripNonCriticalHtmlAssets(),
  ],
  optimizeDeps: {
    esbuildOptions: {
      loader: {
        '.js': 'jsx',
        '.json': 'json',
      },
    },
  },
  build: {
    modulePreload: {
      resolveDependencies(filename, deps, { hostType }) {
        if (hostType !== 'html') {
          return [];
        }
        return deps.filter(shouldPreloadFromHtml);
      },
    },
    rollupOptions: {
      output: {
        hoistTransitiveImports: false,
        manualChunks,
      },
    },
  },
  server: {
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/mj': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/pg': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
});
