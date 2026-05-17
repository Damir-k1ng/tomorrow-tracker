import { fileURLToPath, URL } from 'node:url';
import { defineConfig, type Plugin } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

// buildMeta writes frontend/dist/.build_meta.json on every real `vite build`.
// The Go server embeds this marker and refuses to boot in production without
// it, so a skipped or failed frontend build can never ship the committed
// placeholder index.html. `generateBundle` runs only during `vite build`
// (apply: 'build'), so the dev server and the placeholder dist stay
// marker-free.
function buildMeta(): Plugin {
  return {
    name: 'tomorrow-tracker:build-meta',
    apply: 'build',
    generateBundle() {
      this.emitFile({
        type: 'asset',
        fileName: '.build_meta.json',
        source: JSON.stringify({ built: true }, null, 2) + '\n',
      });
    },
  };
}

// Vite configuration for the Telegram Mini App.
//
// - Output goes to `dist/` (→ frontend/dist), which the Go binary embeds.
// - `base: './'` keeps asset URLs relative so the SPA works no matter what
//   path Telegram mounts it under.
// - `target: es2020` keeps the bundle compatible with older Android Telegram
//   WebViews.
// - The dev server proxies /api to the Go backend so local dev is same-origin.
export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss(), buildMeta()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2020',
    sourcemap: false,
    rollupOptions: {
      output: {
        // Split heavy, rarely-changing vendor code from app code.
        manualChunks: {
          react: ['react', 'react-dom', 'react-router-dom'],
          query: ['@tanstack/react-query'],
          motion: ['framer-motion'],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
