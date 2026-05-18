import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

const appBase = '/djangoadminx/'
const apiProxyPrefix = `${appBase}api`

// https://vite.dev/config/
export default defineConfig({
  base: appBase,
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    port: 5173,
    proxy: {
      [apiProxyPrefix]: {
        target: 'http://localhost:9999',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/djangoadminx\/api/, '/api'),
      },
    },
  },
})
