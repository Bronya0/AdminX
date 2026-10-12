<template>
  <div class="top-nav" :class="{ 'mix-mode': isMix }">
    <div class="top-nav-logo" v-if="showLogo">
      <DashboardOutlined style="font-size: 28px; color: #1890ff" />
      <span>{{ userStore.siteName }}</span>
    </div>
    <a-menu
      :selectedKeys="selectedKeys"
      mode="horizontal"
      :theme="navTheme"
      @select="handleMenuClick"
      style="flex: 1; min-width: 0;"
    >
      <!-- 递归渲染（复用 SidebarMenuItem，与侧栏同一套无限层级逻辑；此前只渲染两层） -->
      <SidebarMenuItem v-for="item in menus" :key="item.key" :item="item" :getIcon="getIcon" />
    </a-menu>
  </div>
</template>

<script setup lang="ts">
import { DashboardOutlined } from '@ant-design/icons-vue'
import SidebarMenuItem from '@/components/SidebarMenuItem.vue'
import { resolveIcon } from '@/utils/iconResolver'
import { useUserStore } from '@/stores/user'
import type { SidebarItem } from '@/types'

const getIcon = resolveIcon
const userStore = useUserStore()

const props = withDefaults(defineProps<{
  menus: SidebarItem[]
  selectedKeys: string[]
  showLogo?: boolean
  isMix?: boolean
  /** 导航底色是浅色时，antd 菜单要跟着切浅色主题，否则白字看不见 */
  navTheme?: 'light' | 'dark'
}>(), {
  navTheme: 'dark',
})

const emit = defineEmits<{
  select: [key: string]
}>()

const handleMenuClick = (info: any) => {
  const key = info?.key
  if (key) emit('select', key)
}
</script>

<style scoped>
.top-nav {
  display: flex;
  align-items: center;
  /* 行高由主题配置下发（AdminLayout 注入），top 与 mix 共用同一个值 */
  height: var(--admin-nav-height, 56px);
  background: var(--admin-nav-top-bg, #001529);
  padding: 0 16px;
  overflow: hidden;
}

.top-nav-logo {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--admin-nav-top-text, #fff);
  font-size: 18px;
  font-weight: bold;
  padding-right: 24px;
  flex-shrink: 0;
}

:deep(.ant-menu-horizontal) {
  border-bottom: none;
  line-height: var(--admin-nav-height, 56px);
  /* 菜单底透明，导航底色才能透出来（色由主题配置给） */
  background: transparent;
}

:deep(.ant-menu-horizontal),
:deep(.ant-menu-item),
:deep(.ant-menu-submenu-title) {
  font-size: var(--admin-nav-font-size, 14px);
}

:deep(.ant-menu-item),
:deep(.ant-menu-submenu-title) {
  padding: 0 20px;
}
</style>