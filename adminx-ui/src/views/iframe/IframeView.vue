<template>
  <div class="iframe-container">
    <div class="iframe-header" v-if="title">
      <span class="iframe-title">{{ title }}</span>
    </div>
    <iframe
      v-if="iframeUrl"
      ref="iframeRef"
      :src="iframeUrl"
      class="iframe-full"
      frameborder="0"
      allow="fullscreen"
    ></iframe>
    <a-empty v-else :description="emptyReason" style="margin-top: 120px;" />
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

// 白名单校验：url 必须与当前用户菜单中 menu_type === 'iframe' 的菜单一致。
// 防止通过构造 /iframe?url=https://evil.com 的链接，把登录者的 token 注入任意站点。
const normalizeUrl = (url: string): string => {
  try {
    const u = new URL(url)
    // 比较协议 + host + path，忽略末尾斜杠与查询参数差异
    return `${u.protocol}//${u.host}${u.pathname.replace(/\/+$/, '')}`
  } catch {
    return ''
  }
}

const isWhitelistedUrl = (url: string): boolean => {
  const target = normalizeUrl(url)
  if (!target) return false
  return userStore.menus.some(
    (m) => m.menu_type === 'iframe' && m.path && normalizeUrl(m.path) === target
  )
}

const emptyReason = computed(() => {
  if (!externalUrl.value || !safeUrl.value) return '未提供 URL 或 URL 非法'
  return '该地址不在可信任的业务系统列表中，未加载'
})

const iframeUrl = computed(() => (safeUrl.value && isWhitelistedUrl(safeUrl.value) ? safeUrl.value : ''))

// ── token 下发：postMessage 单次下发，token 不再进 URL（hash/query 都会留痕）──
// 业务页面加载后发送 { type: 'adminx:request-token' }，本页校验来源后回传 token。
const iframeRef = ref<HTMLIFrameElement | null>(null)

const onMessage = (event: MessageEvent) => {
  if (!safeUrl.value || !iframeUrl.value) return
  let expectedOrigin = ''
  try {
    expectedOrigin = new URL(safeUrl.value).origin
  } catch {
    return
  }
  if (event.origin !== expectedOrigin) return
  const source = event.source
  if (!source || source !== iframeRef.value?.contentWindow) return
  const data = event.data as { type?: string } | null
  if (data?.type === 'adminx:request-token') {
    source.postMessage(
      { type: 'adminx:token', token: userStore.token || '' },
      { targetOrigin: expectedOrigin }
    )
  }
}

window.addEventListener('message', onMessage)

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
  window.removeEventListener('message', onMessage)
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
