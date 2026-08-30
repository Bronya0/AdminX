import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

const appBase = '/adminx/'
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
      // 保留 /adminx 前缀直传后端：后端 base_path 默认 /adminx，
      // 前端 base（BASE_URL）+ 后端挂载路径 + RBAC 种子规则三方必须一致
      [apiProxyPrefix]: {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
})
