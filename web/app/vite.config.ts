import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: resolve(__dirname, '../static/app'),
    emptyOutDir: true
  },
  server: {
    port: 5173,
    proxy: {
      '/user': 'http://localhost:8080',
      '/admin': 'http://localhost:8080',
      '/api': 'http://localhost:8080',
      '/static': 'http://localhost:8080'
    }
  }
})

