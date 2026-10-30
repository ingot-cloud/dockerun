import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_')
  const backend = env.VITE_BACKEND_URL || 'http://127.0.0.1:8080'
  const proxy = { target: backend, changeOrigin: true, ws: true }
  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src')
      }
    },
    server: {
      port: 5999,
      strictPort: true,
      proxy: {
        '/api': proxy,
        '/v2': proxy,
        '/registry/token': proxy,
        '/health': proxy
      }
    }
  }
})

