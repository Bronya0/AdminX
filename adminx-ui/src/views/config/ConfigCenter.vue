<template>
  <div class="page-container">
    <a-card class="table-card">
      <a-tabs v-model:activeKey="activeTab" type="card">
        <a-tab-pane key="config" tab="字典配置">
          <ConfigList />
        </a-tab-pane>
        <a-tab-pane key="theme" tab="主题设置">
          <ThemeSettings />
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ConfigList from './ConfigList.vue'
import ThemeSettings from '@/views/system/ThemeSettings.vue'

const TABS = ['config', 'theme'] as const
type TabKey = (typeof TABS)[number]

const isTabKey = (value: unknown): value is TabKey => TABS.includes(value as TabKey)

const route = useRoute()
const router = useRouter()

// 当前 tab 以 URL 的 ?tab= 为唯一事实源：页头齿轮按钮只是 push 一个新的 query，
// 组件不会重建，靠下面这个 computed 把 query 变化落到 tab 上；
// 反过来切 tab 也写回 URL，这样刷新、分享链接、回退都能回到同一个 tab。
const activeTab = computed<TabKey>({
  get: () => {
    const tab = route.query.tab
    return isTabKey(tab) ? tab : 'config'
  },
  set: (tab) => {
    if (route.query.tab === tab) return
    router.replace({ query: { ...route.query, tab } })
  },
})
</script>

<style scoped>
.page-container {
  padding: 24px;
}
.table-card {
  margin-bottom: 16px;
}
</style>
