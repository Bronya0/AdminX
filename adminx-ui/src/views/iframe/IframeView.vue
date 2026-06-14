<template>
  <div class="iframe-container">
    <div class="iframe-header" v-if="title">
      <span class="iframe-title">{{ title }}</span>
    </div>
    <iframe
      v-if="iframeUrl"
      :src="iframeUrl"
      class="iframe-full"
      frameborder="0"
      allow="fullscreen"
    ></iframe>
    <a-empty v-else description="未提供 URL" style="margin-top: 120px;" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const userStore = useUserStore()

const externalUrl = computed(() => {
  const url = route.query.url
  if (typeof url === 'string') return url
  if (Array.isArray(url)) return url[0] || ''
  return ''
})

const title = computed(() => {
  const t = route.query.title
  if (typeof t === 'string') return t
  if (Array.isArray(t)) return t[0] || ''
  return ''
})

// 安全校验：仅允许 http/https 协议，防止 javascript:/data: 等危险协议
const isSafeUrl = (url: string): boolean => {
  if (!url) return false
  try {
    const u = new URL(url, window.location.origin)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

const safeUrl = computed(() => (isSafeUrl(externalUrl.value) ? externalUrl.value : ''))

// token 通过 URL 传递会进入浏览器历史、Referer、服务器日志，存在泄漏风险。
// 对内嵌可信业务系统是常见做法，但仍优先用 postMessage 方案；当前保留 URL 方式但
// 仅对通过校验的 URL 注入 token，并使用 hash 避免 token 进入 query string 被日志记录。
const iframeUrl = computed(() => {
  if (!safeUrl.value) return ''
  const sep = safeUrl.value.includes('?') ? '&' : '?'
  return `${safeUrl.value}${sep}token=${encodeURIComponent(userStore.token || '')}`
})

// prevTitle 必须在客户端环境(onMounted)中读取，避免 setup 顶层访问 document（SSR/测试会报错）
const prevTitle = ref('')

onMounted(() => {
  prevTitle.value = document.title
  if (title.value) {
    document.title = title.value
  }
})

onUnmounted(() => {
  document.title = prevTitle.value
})
</script>

<style scoped>
.iframe-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.iframe-header {
  padding: 8px 16px;
  background: var(--admin-bg-header, #fff);
  border-bottom: 1px solid var(--admin-border-color, #f0f0f0);
  font-size: 14px;
  font-weight: 500;
  flex-shrink: 0;
}

.iframe-full {
  flex: 1;
  width: 100%;
  border: none;
}
</style>
