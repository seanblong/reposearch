import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // API endpoints proxied during local dev so fetch() calls go to the API
      '/search': 'http://localhost:8080',
      '/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/repositories': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/healthz': 'http://localhost:8080',
    }
  }
})
