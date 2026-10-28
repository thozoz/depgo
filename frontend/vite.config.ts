import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: path.resolve(__dirname, '../backend/public/dist'),
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:5244',
        changeOrigin: true,
      },
      '/d': {
        target: 'http://localhost:5244',
        changeOrigin: true,
      },
      '/p': {
        target: 'http://localhost:5244',
        changeOrigin: true,
      },
      '/sp': {
        target: 'http://localhost:5244',
        changeOrigin: true,
      },
      '/sd': {
        target: 'http://localhost:5244',
        changeOrigin: true,
      },
    },
  },
});
