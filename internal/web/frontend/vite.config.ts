import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In `kapish serve --dev`, the Go server proxies `/` here; this dev server
// proxies /api (including WebSocket upgrades) back to the Go server. The Go
// server's port is passed via VITE_KAPISH_API (default localhost:0 won't work,
// so --dev passes the real port through an env var).
const apiTarget = process.env.VITE_KAPISH_API || 'http://127.0.0.1:8765';

export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    proxy: {
      '/api': { target: apiTarget, changeOrigin: false, ws: true },
    },
  },
});
