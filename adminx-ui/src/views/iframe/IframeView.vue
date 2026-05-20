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
import { computed, onMounted, onUnmounted } from 'vue'
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

const iframeUrl = computed(() => {
  if (!externalUrl.value) return ''
  const sep = externalUrl.value.includes('?') ? '&' : '?'
  return `${externalUrl.value}${sep}token=${userStore.token}`
})

const prevTitle = document.title

onMounted(() => {
  if (title.value) {
    document.title = title.value
  }
})

onUnmounted(() => {
  document.title = prevTitle
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
