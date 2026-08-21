import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const backend = 'http://localhost:8080'
const localRoutes = ['/api', '/healthz', '/readyz'] as const

export default defineConfig(({ mode }) => {
  const proxy = Object.fromEntries(localRoutes.map(route => [route, { target: backend, changeOrigin: false }]))
  return {
    appType: 'spa',
    plugins: [vue()],
    server: { host: '127.0.0.1', strictPort: true, proxy },
    preview: { host: '127.0.0.1', strictPort: true },
    build: { sourcemap: mode !== 'production', reportCompressedSize: true },
    test: { environment: 'node', restoreMocks: true, clearMocks: true }
  }
})
