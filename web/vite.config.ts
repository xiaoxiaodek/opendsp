import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_TARGET || 'http://localhost:8081',
        changeOrigin: true,
      },
      '^/creative/': {
        target: process.env.VITE_ASSETS_BASE_URL || 'http://localhost:9001',
        changeOrigin: true,
      },
      '^/proof/': {
        target: process.env.VITE_ASSETS_BASE_URL || 'http://localhost:9001',
        changeOrigin: true,
      },
      '^/asset/': {
        target: process.env.VITE_ASSETS_BASE_URL || 'http://localhost:9001',
        changeOrigin: true,
      },
    },
  },
})
