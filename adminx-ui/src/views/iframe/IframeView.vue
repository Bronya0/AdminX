<template>
  <div class="iframe-container">
    <div class="iframe-header" v-if="title">
      <span class="iframe-title">{{ title }}</span>
    </div>
    <iframe
      v-if="iframeUrl"
      ref="iframeRef"
      :key="iframeUrl + attempt"
      title="业务页面"
      :src="iframeUrl"
      class="iframe-full"
      frameborder="0"
      allow="fullscreen"
    ></iframe>
    <div v-if="iframeUrl && waiting" class="bridge-overlay"><a-spin tip="正在连接业务服务…" /></div>
    <div v-if="iframeUrl && failed" class="bridge-overlay">
      <a-result status="warning" title="业务服务未就绪" sub-title="请检查业务服务和平台接入配置后重试。">
        <template #extra><a-button type="primary" @click="retry">重新加载</a-button></template>
      </a-result>
    </div>
    <a-empty v-else-if="!iframeUrl" :description="emptyReason" style="margin-top: 120px;" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
import { Modal } from 'ant-design-vue'
import { useBusinessBridge } from '@/composables/useBusinessBridge'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
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
    const u = new URL(url, window.location.origin)
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

// 通信能力与新标签页共用；业务路由只更新外壳 URL，不重载 iframe。
const iframeRef = ref<HTMLIFrameElement | null>(null)
const waiting = ref(true)
const failed = ref(false)
const dirty = ref(false)
const attempt = ref(0)
let timer: ReturnType<typeof setTimeout> | undefined
function startWaiting() {
  clearTimeout(timer)
  waiting.value = true
  failed.value = false
  dirty.value = false
  if (iframeUrl.value) timer = setTimeout(() => { waiting.value = false; failed.value = true }, 45000)
}
const retry = () => { attempt.value++; startWaiting() }
watch(iframeUrl, startWaiting, { immediate: true })
useBusinessBridge(() => {
  const win = iframeRef.value?.contentWindow
  return win && iframeUrl.value ? [{ win, origin: new URL(iframeUrl.value, window.location.origin).origin }] : []
}, {
  ready: () => { clearTimeout(timer); waiting.value = false; failed.value = false },
  dirty: value => { dirty.value = value },
  initialRoute: () => typeof route.query.businessRoute === 'string' ? route.query.businessRoute : '',
  route: hash => {
    if (hash !== (route.query.businessRoute || '')) {
      void router.replace({ path: route.path, query: { ...route.query, businessRoute: hash || undefined } })
    }
  },
})
const canLeave = () => !userStore.token || !dirty.value || new Promise<boolean>(resolve => {
  Modal.confirm({ title: '还有未保存的内容，确定离开？', okText: '确定', cancelText: '取消', onOk: () => resolve(true), onCancel: () => resolve(false) })
})
onBeforeRouteLeave(canLeave)
onBeforeRouteUpdate((to, from) => to.query.url !== from.query.url ? canLeave() : true)
const beforeUnload = (event: BeforeUnloadEvent) => { if (dirty.value) { event.preventDefault(); event.returnValue = '' } }
window.addEventListener('beforeunload', beforeUnload)

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
  clearTimeout(timer)
  window.removeEventListener('beforeunload', beforeUnload)
})
</script>

<style scoped>
.bridge-overlay { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: var(--admin-bg-header, #fff); z-index: 1; }

.iframe-container {
  position: relative;
  min-height: 0;
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
