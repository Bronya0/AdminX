<template>
  <div class="admin-sider" :class="{ collapsed }">
    <div class="logo" v-if="showLogo">
      <DashboardOutlined style="font-size: 32px; color: #1890ff" v-if="!collapsed" />
      <span v-if="!collapsed" style="margin-left: 12px">{{ userStore.siteName }}</span>
      <DashboardOutlined style="font-size: 24px; color: #1890ff" v-else />
    </div>
    <a-menu
      :selectedKeys="selectedKeys"
      :openKeys="collapsed ? [] : openKeys"
      mode="inline"
      theme="dark"
      :inline-collapsed="collapsed"
      @select="handleMenuClick"
      @openChange="handleOpenChange"
    >
      <template v-for="item in menus" :key="item.key">
        <a-sub-menu v-if="item.children && item.children.length" :key="item.key">
          <template #icon>
            <component :is="getIcon(item.icon)" />
          </template>
          <template #title>{{ item.title }}</template>
          <a-menu-item v-for="child in item.children" :key="child.key">
            <template #icon>
              <component :is="getIcon(child.icon)" />
            </template>
            {{ child.title }}
          </a-menu-item>
        </a-sub-menu>
        <a-menu-item v-else :key="item.key">
          <template #icon>
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
  collapsed: boolean
  selectedKeys: string[]
  openKeys: string[]
  showLogo?: boolean
}>()

const emit = defineEmits<{
  select: [key: string]
  'update:openKeys': [keys: string[]]
}>()

const handleMenuClick = (info: any) => {
  const key = info?.key
  if (key) emit('select', key)
}

const handleOpenChange = (keys: string[]) => {
  emit('update:openKeys', keys)
}
</script>

<style scoped>
.admin-sider {
  width: 256px;
  min-width: 256px;
  height: 100vh;
  background: #001529;
  overflow-y: auto;
  overflow-x: hidden;
  transition: all 0.2s;
}

.admin-sider.collapsed {
  width: 80px;
  min-width: 80px;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
  font-weight: bold;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}
</style>