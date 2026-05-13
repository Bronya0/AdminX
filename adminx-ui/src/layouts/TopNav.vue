<template>
  <div class="top-nav" :class="{ 'mix-mode': isMix }">
    <div class="top-nav-logo" v-if="showLogo">
      <DashboardOutlined style="font-size: 28px; color: #1890ff" />
      <span v-if="!isMix">{{ userStore.siteName }}</span>
    </div>
    <a-menu
      :selectedKeys="selectedKeys"
      mode="horizontal"
      theme="dark"
      @select="handleMenuClick"
      style="flex: 1; min-width: 0; line-height: 56px;"
    >
      <template v-for="item in menus" :key="item.key">
        <a-sub-menu v-if="item.children && item.children.length" :key="item.key">
          <template #icon v-if="item.icon">
            <component :is="getIcon(item.icon)" />
          </template>
          <template #title>{{ item.title }}</template>
          <a-menu-item v-for="child in item.children" :key="child.key">
            <template #icon v-if="child.icon">
              <component :is="getIcon(child.icon)" />
            </template>
            {{ child.title }}
          </a-menu-item>
        </a-sub-menu>
        <a-menu-item v-else :key="item.key">
          <template #icon v-if="item.icon">
            <component :is="getIcon(item.icon)" />
          </template>
          {{ item.title }}
        </a-menu-item>
      </template>
    </a-menu>
  </div>
</template>

<script setup lang="ts">
import { DashboardOutlined } from '@ant-design/icons-vue'
import { resolveIcon } from '@/utils/iconResolver'
import { useUserStore } from '@/stores/user'
import type { SidebarItem } from '@/types'

const getIcon = resolveIcon
const userStore = useUserStore()

const props = defineProps<{
  menus: SidebarItem[]
  selectedKeys: string[]
  showLogo?: boolean
  isMix?: boolean
}>()

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
  height: 56px;
  background: #001529;
  padding: 0 16px;
  overflow: hidden;
}

.top-nav.mix-mode {
  height: 48px;
}

.top-nav-logo {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  padding-right: 24px;
  flex-shrink: 0;
}

:deep(.ant-menu-horizontal) {
  border-bottom: none;
}

:deep(.ant-menu-item),
:deep(.ant-menu-submenu-title) {
  padding: 0 20px;
}
</style>