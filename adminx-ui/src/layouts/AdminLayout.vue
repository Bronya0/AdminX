<template>
  <div class="admin-wrapper" :class="layout">
    <!-- SIDE layout: sidebar only -->
    <SideBar
      v-if="layout === 'side'"
      :menus="sidebarMenus"
      :collapsed="collapsed"
      :selectedKeys="selectedKeys"
      :openKeys="openKeys"
      :showLogo="userStore.theme.showLogo"
      :showVersion="userStore.theme.showVersion !== false"
      @select="handleMenuClick"
      @update:openKeys="openKeys = $event"
    />

    <!-- TOP layout: top nav only -->
    <TopNav
      v-if="layout === 'top'"
      :menus="sidebarMenus"
      :selectedKeys="selectedKeys"
      :showLogo="userStore.theme.showLogo"
      @select="handleMenuClick"
    />

    <!-- MIX layout: top nav + sidebar -->
    <template v-if="layout === 'mix'">
      <TopNav
        :menus="topLevelMenus"
        :selectedKeys="[selectedTopKey]"
        :isMix="true"
        :showLogo="userStore.theme.showLogo"
        @select="handleTopMenuClick"
      />
      <div class="mix-body">
        <SideBar
          v-if="currentChildMenus.length"
          :menus="currentChildMenus"
          :collapsed="collapsed"
          :selectedKeys="selectedKeys"
          :openKeys="openKeys"
          :showLogo="false"
          :showVersion="userStore.theme.showVersion !== false"
          @select="handleMenuClick"
          @update:openKeys="openKeys = $event"
        />
        <!-- 主区域（MIX 模式放在 mix-body 内） -->
        <div class="admin-main" :class="{ collapsed }">
          <div class="admin-header">
            <div class="header-left">
              <a-button type="text" @click="toggleCollapsed">
                <MenuFoldOutlined v-if="!collapsed" />
                <MenuUnfoldOutlined v-else />
              </a-button>
              <breadcrumb-nav v-if="showBreadcrumb" />
            </div>
            <div class="header-right">
              <a-space>
                <a-tooltip title="全屏">
                  <a-button type="text" @click="toggleFullscreen">
                    <FullscreenOutlined />
                  </a-button>
                </a-tooltip>
                <a-tooltip title="通知中心">
                  <a-button type="text" style="display: inline-flex; align-items: center; justify-content: center; width: 40px; height: 40px; padding: 0;" @click="goToNotification">
                     <a-badge :count="unreadCount" :overflow-count="99" :offset="[2, -2]">
                      <BellOutlined style="font-size: 18px;" />
                    </a-badge>
                  </a-button>
                </a-tooltip>
                <a-dropdown>
                  <a-space style="cursor: pointer; display: inline-flex; align-items: center;">
                    <a-avatar :size="32">
                      <template #icon><UserOutlined /></template>
                    </a-avatar>
                    <span style="font-size: 14px; color: #333;">{{ username }}</span>
                    <span v-if="userRoles" style="font-size: 12px; color: #999; margin-right: 0;">({{ userRoles }})</span>
                    <DownOutlined style="font-size: 12px; color: #999;" />
                  </a-space>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item @click="handleLogout">
                        <LogoutOutlined /> 退出登录
                      </a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
              </a-space>
            </div>
          </div>
          <div class="admin-content">
            <router-view />
          </div>
        </div>
      </div>
    </template>

    <!-- 主区域（SIDE / TOP 布局） -->
    <div v-else class="admin-main" :class="{ collapsed: layout !== 'top' && collapsed }">
      <div class="admin-header">
        <div class="header-left">
          <a-button v-if="layout !== 'top'" type="text" @click="toggleCollapsed">
            <MenuFoldOutlined v-if="!collapsed" />
            <MenuUnfoldOutlined v-else />
          </a-button>
          <breadcrumb-nav v-if="showBreadcrumb" />
        </div>
        <div class="header-right">
          <a-space>
            <a-tooltip title="全屏">
              <a-button type="text" @click="toggleFullscreen">
                <FullscreenOutlined />
              </a-button>
            </a-tooltip>
            <a-tooltip title="通知中心">
              <a-button type="text" style="display: inline-flex; align-items: center; justify-content: center; width: 40px; height: 40px; padding: 0;" @click="goToNotification">
                 <a-badge :count="unreadCount" :overflow-count="99" :offset="[2, -2]">
                  <BellOutlined style="font-size: 18px;" />
                </a-badge>
              </a-button>
            </a-tooltip>
            <a-dropdown>
              <a-space style="cursor: pointer; display: inline-flex; align-items: center;">
                <a-avatar :size="32">
                  <template #icon><UserOutlined /></template>
                </a-avatar>
                <span style="font-size: 14px; color: #333;">{{ username }}</span>
                <span v-if="userRoles" style="font-size: 12px; color: #999; margin-right: 0;">({{ userRoles }})</span>
                <DownOutlined style="font-size: 12px; color: #999;" />
              </a-space>
              <template #overlay>
                <a-menu>
                  <a-menu-item @click="handleLogout">
                    <LogoutOutlined /> 退出登录
                  </a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </a-space>
        </div>
      </div>
      <div class="admin-content">
        <router-view />
      </div>
    </div>

    <!-- 空闲超时检测 -->
    <IdleWatcher />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { message, Modal } from 'ant-design-vue'
import IdleWatcher from '@/components/IdleWatcher.vue'
import { notificationApi } from '@/api/notification'
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  BellOutlined,
  LogoutOutlined,
  FullscreenOutlined,
  DownOutlined,
} from '@ant-design/icons-vue'
import { flatMenusToTreeByDepth, menusToSidebarItems } from '@/utils/menuTree'
import BreadcrumbNav from '@/components/BreadcrumbNav.vue'
import SideBar from './SideBar.vue'
import TopNav from './TopNav.vue'
import type { SidebarItem } from '@/types'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const collapsed = computed({
  get: () => userStore.theme.collapsed,
  set: (val) => userStore.updateTheme({ collapsed: val }),
})
const isDark = computed(() => userStore.theme.isDark)
const layout = computed(() => userStore.theme.layout)
const showBreadcrumb = computed(() => userStore.theme.showBreadcrumb !== false)

const selectedKeys = ref<string[]>([])
const openKeys = ref<string[]>([])
const username = computed(() => userStore.username || '未登录')
const userRoles = computed(() => userStore.user?.role_names?.join(', ') || '')

// 未读通知数量
const unreadCount = ref(0)
let unreadPollTimer: ReturnType<typeof setInterval> | null = null

const fetchUnreadCount = async () => {
  try {
    const res = await notificationApi.unreadCount()
    unreadCount.value = res.count
  } catch {
    // 获取失败时保持旧值
  }
}

const startUnreadPolling = () => {
  stopUnreadPolling()
  unreadPollTimer = setInterval(fetchUnreadCount, 30000)
}

const stopUnreadPolling = () => {
  if (unreadPollTimer !== null) {
    clearInterval(unreadPollTimer)
    unreadPollTimer = null
  }
}

const sidebarMenus = computed<SidebarItem[]>(() => {
  const tree = flatMenusToTreeByDepth(userStore.menus || [])
  return menusToSidebarItems(tree, (code: string) => userStore.hasPermission(code))
})

// MIX 布局的一级菜单（只保留顶级，不显示子菜单）
const topLevelMenus = computed(() =>
  sidebarMenus.value.map(m => ({ ...m, children: undefined }))
)

// MIX 布局：当前选中的一级菜单的子菜单
const selectedTopKey = ref('')
const currentChildMenus = computed(() => {
  if (!selectedTopKey.value) return []
  const parent = sidebarMenus.value.find(m => m.key === selectedTopKey.value)
  return parent?.children || []
})

// 监听路由变化
watch(
  () => route.path,
  (path) => {
    selectedKeys.value = [path]

    // 更新 MIX layout 的选中顶级菜单
    if (layout.value === 'mix') {
      const parts = path.split('/').filter(Boolean)
      if (parts.length >= 1) {
        const topKey = '/' + parts[0]
        selectedTopKey.value = topKey
      }
    }

    // 自动展开父菜单
    const parentPath = path.split('/').slice(0, 2).join('/')
    if (parentPath !== path && !openKeys.value.includes(parentPath)) {
      openKeys.value.push(parentPath)
    }
  },
  { immediate: true }
)

const toggleCollapsed = () => userStore.toggleCollapsed()

const toggleFullscreen = () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
  } else {
    document.exitFullscreen()
  }
}

const findSidebarItem = (key: string, items: SidebarItem[]): SidebarItem | undefined => {
  for (const item of items) {
    if (item.key === key) return item
    if (item.children) {
      const found = findSidebarItem(key, item.children)
      if (found) return found
    }
  }
  return undefined
}

// ── 外链 token 下发：postMessage 单次下发，token 不进 URL（hash 会进浏览器历史且可被页面脚本读取）──
// 打开的新标签页加载后发送 { type: 'adminx:request-token' }，本布局校验来源后回传。
const pendingTokenWindows: Array<{ win: Window; origin: string }> = []

const onTokenRequest = (event: MessageEvent) => {
  const data = event.data as { type?: string } | null
  if (data?.type !== 'adminx:request-token') return
  const entry = pendingTokenWindows.find((e) => e.win === event.source)
  if (!entry || entry.origin !== event.origin) return
  entry.win.postMessage(
    { type: 'adminx:token', token: userStore.token || '' },
    { targetOrigin: entry.origin }
  )
}

window.addEventListener('message', onTokenRequest)
onUnmounted(() => window.removeEventListener('message', onTokenRequest))

const openExternal = (item: SidebarItem) => {
  const targetUrl = item.url!
  if (item.menu_type === 'iframe') {
    router.push(`/iframe?url=${encodeURIComponent(targetUrl)}&title=${encodeURIComponent(item.title)}`)
  } else {
    // 不携带 token 打开；业务页加载后通过 postMessage 请求 token
    let expectedOrigin = ''
    try {
      expectedOrigin = new URL(targetUrl).origin
    } catch {
      message.error('外部链接非法')
      return
    }
    const win = window.open(targetUrl, '_blank')
    if (win) {
      pendingTokenWindows.push({ win, origin: expectedOrigin })
    }
  }
}

const handleMenuClick = (key: string) => {
  if (!key) return
  const item = findSidebarItem(key, sidebarMenus.value)
  if (item?.url) {
    openExternal(item)
    return
  }
  router.push(key).catch((err) => {
    if (err.name !== 'NavigationDuplicated') {
      console.warn(`导航到 "${key}" 失败:`, err)
    }
  })
}

// MIX 布局：点击顶级菜单 -> 跳转到第一个子菜单
const handleTopMenuClick = (key: string) => {
  selectedTopKey.value = key
  const parent = sidebarMenus.value.find(m => m.key === key)
  const targetItem = parent?.children?.[0] ?? parent
  if (!targetItem) return
  if (targetItem.url) {
    openExternal(targetItem)
    return
  }
  const routeExists = router.getRoutes().some(r => r.path === targetItem.key)
  if (routeExists) {
    router.push(targetItem.key)
  }
}

const goToNotification = () => {
  router.push('/system/notification')
}

const handleLogout = () => {
  Modal.confirm({
    title: '确认退出',
    content: '确定要退出登录吗？',
    onOk: async () => {
      // 统一收口到 store.logout()（服务端吊销 + 清本地状态）
      await userStore.logout()
      message.success('已退出登录')
      await router.replace('/login').catch(() => {})
    },
  })
}

onMounted(() => {
  // 应用暗黑模式
  if (userStore.theme.isDark) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  // 应用主题色
  document.documentElement.style.setProperty('--primary-color', userStore.theme.primaryColor)
  // 应用字体大小
  const fontSize = userStore.theme.fontSize ?? 14
  if (fontSize !== 14) {
    document.documentElement.style.fontSize = fontSize + 'px'
  }
  fetchUnreadCount()
  startUnreadPolling()
})

onUnmounted(() => {
  stopUnreadPolling()
})
</script>

<style scoped>
.admin-wrapper {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

/* TOP layout：纵向排列 */
.admin-wrapper.top {
  flex-direction: column;
}

/* MIX layout：顶部导航 + 下方 sidebar+content 横向 */
.admin-wrapper.mix {
  flex-direction: column;
}

.mix-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* 主区域 */
.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.admin-header {
  height: 64px;
  min-height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: var(--admin-bg-header);
  box-shadow: var(--admin-header-shadow);
  transition: background 0.3s, box-shadow 0.3s;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
  min-width: 0;
}

.header-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.admin-content {
  flex: 1;
  overflow-y: auto;
  background: var(--admin-bg-content);
  padding: 16px;
  transition: background 0.3s;
}

:deep(.ant-radio-group-solid) .ant-radio-button-wrapper {
  min-width: 100px;
  text-align: center;
}
</style>