import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import path from 'path';

export default defineConfig({
  base: '/default/',
  root: '.',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  plugins: [react()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    modulePreload: {
      resolveDependencies() {
        return [];
      },
    },
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined;
          if (id.includes('/react') || id.includes('/react-dom')) {
            return 'react-core';
          }
          if (id.includes('/lucide-react')) {
            return 'icons';
          }
          if (id.includes('/axios')) {
            return 'api-client';
          }
          return undefined;
        },
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
    },
  },
});
