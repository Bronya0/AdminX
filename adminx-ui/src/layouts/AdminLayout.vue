<template>
  <div
    class="admin-wrapper"
    :class="layout"
    :style="{
      '--admin-nav-height': navHeight + 'px',
      '--admin-header-height': headerHeight + 'px',
    }"
  >
    <!-- 顶部导航（top / mix）：side 布局没有这一行 -->
    <TopNav
      v-if="layout !== 'side'"
      :menus="layout === 'mix' ? topLevelMenus : sidebarMenus"
      :navTheme="userStore.navTop.menuTheme"
      :selectedKeys="layout === 'mix' ? [selectedTopKey] : selectedKeys"
      :isMix="layout === 'mix'"
      :showLogo="userStore.theme.showLogo"
      @select="handleTopNavSelect"
    />

    <!-- 侧栏 + 主区：三种布局共用同一棵子树。
         若把主区放进不同的 v-if 分支，切布局会卸载并重建 <router-view>：
         页面状态丢失，页面 onMounted 里的 fetchTheme 还会把未保存的主题预览覆盖回服务端值，
         表现就是「点混合布局没反应」。 -->
    <div class="layout-body">
      <SideBar
        v-if="layout === 'side' || (layout === 'mix' && currentChildMenus.length > 0)"
        :menus="layout === 'mix' ? currentChildMenus : sidebarMenus"
        :navTheme="userStore.navSide.menuTheme"
        :collapsed="collapsed"
        :selectedKeys="selectedKeys"
        :openKeys="openKeys"
        :showLogo="layout === 'side' && userStore.theme.showLogo"
        :showVersion="userStore.theme.showVersion !== false"
        @select="handleMenuClick"
        @update:openKeys="openKeys = $event"
      />

      <div class="admin-main" :class="{ collapsed: layout !== 'top' && collapsed }">
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
              <a-button
                v-if="canConfigCenter"
                class="header-icon-btn"
                type="text"
                aria-label="配置中心"
                @click="goToConfigCenter"
              >
                <SettingOutlined style="font-size: 18px;" />
              </a-button>
              <a-button
                class="header-icon-btn"
                type="text"
                aria-label="通知中心"
                @click="goToNotification"
              >
                <a-badge :count="unreadCount" :overflow-count="99" :offset="[2, -2]">
                  <BellOutlined style="font-size: 18px;" />
                </a-badge>
              </a-button>
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

    <!-- 空闲超时检测 -->
    <IdleWatcher />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, watchEffect, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore, defaultTheme } from '@/stores/user'
import { message, Modal } from 'ant-design-vue'
import IdleWatcher from '@/components/IdleWatcher.vue'
import { notificationApi } from '@/api/notification'
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  BellOutlined,
  LogoutOutlined,
  SettingOutlined,
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
// 行高在「配置中心 → 主题设置 → 布局」里调，由这两个 CSS 变量下发到顶栏/页头/侧栏 Logo
const navHeight = computed(() => userStore.theme.navHeight ?? defaultTheme.navHeight)
const headerHeight = computed(() => userStore.theme.headerHeight ?? defaultTheme.headerHeight)
// 导航配色变量挂在 <html> 上：顶部导航的子菜单弹出层是 teleport 到 body 的，
// 挂在 .admin-wrapper 上的话它读不到变量，会跟导航区两个色。
watchEffect(() => {
  const el = document.documentElement.style
  const top = userStore.navTop
  const side = userStore.navSide
  el.setProperty('--admin-nav-top-bg', top.bg)
  el.setProperty('--admin-nav-top-text', top.text)
  el.setProperty('--admin-nav-top-border', top.border)
  el.setProperty('--admin-nav-side-bg', side.bg)
  el.setProperty('--admin-nav-side-text', side.text)
  el.setProperty('--admin-nav-side-border', side.border)
  el.setProperty('--admin-nav-font-size', `${userStore.theme.navTextSize ?? defaultTheme.navTextSize}px`)
})
// 页头「配置中心」入口的准入码取自路由定义，不和 router/index.ts 各写一份
const canConfigCenter = computed(() => {
  const code = router.getRoutes().find(r => r.name === 'system-config')?.meta?.permission
  return !code || userStore.hasPermission(String(code))
})

const selectedKeys = ref<string[]>([])
const openKeys = ref<string[]>([])
const username = computed(() => userStore.username || '未登录')
const userRoles = computed(() => userStore.user?.role_names?.join(', ') || '')

// 未读通知数量
const unreadCount = ref(0)
let unreadPollTimer: ReturnType<typeof setInterval> | null = null

// 进入通知页后跳过一次布局轮询：通知页自己会拉未读数，避免重复请求
let skipUnreadOnce = false

const fetchUnreadCount = async () => {
  if (skipUnreadOnce) {
    skipUnreadOnce = false
    return
  }
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

// 菜单归属：这条菜单属于哪个一级菜单。菜单树是权威（按 parent_id 建出来的），
// 不再按路径首段猜：父 /reports、子 /daily 这种「层级不体现在路径里」的配置下，
// 猜路径只能得到空串，混合布局的左侧栏会整块消失。
const topKeyOfItem = (key: string): string => {
  for (const top of sidebarMenus.value) {
    if (top.key === key) return top.key
    if (top.children?.length && findSidebarItem(key, top.children)) return top.key
  }
  return ''
}

const clickedTopKey = ref('')
// 混合布局：当前选中的一级菜单。点击时写一份 override（外链 / 内嵌页的跳转不改变路由，
// 靠它立即换掉左侧子菜单）；否则按当前路由在菜单树里定位它所属的一级菜单。
const selectedTopKey = computed(() => {
  const keys = topLevelMenus.value.map(m => m.key)
  if (clickedTopKey.value && keys.includes(clickedTopKey.value)) return clickedTopKey.value
  const topKey = topKeyOfItem(route.path)
  return keys.includes(topKey) ? topKey : ''
})
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

    // 进入通知页时标记跳过下一次布局轮询（通知页自会拉取未读数）
    if (path === '/system/notification') skipUnreadOnce = true

    // 路由变了就以路由为准：清掉点击一级菜单时写的 override（见 selectedTopKey）。
    // 例外：外链/内嵌菜单会 push 到 /iframe，此时菜单归属仍然是刚点的那个一级菜单，
    // 清掉它会让混合布局的左侧菜单整块消失。
    if (path !== '/iframe') clickedTopKey.value = ''

    // 自动展开父菜单
    const parentPath = path.split('/').slice(0, 2).join('/')
    if (parentPath !== path && !openKeys.value.includes(parentPath)) {
      openKeys.value.push(parentPath)
    }
  },
  { immediate: true }
)

// 首次拿到菜单时默认展开所有含子菜单的一级菜单：
// 否则子项全部藏在折叠的子菜单里，看上去就像“菜单没了”（仅在首次初始化时设置，
// 之后完全交给用户手动展开/收起）。
const openKeysInitialized = ref(false)
watch(
  sidebarMenus,
  (menus) => {
    if (openKeysInitialized.value || menus.length === 0) return
    openKeys.value = menus.filter((m) => m.children?.length).map((m) => m.key)
    openKeysInitialized.value = true
  },
  { immediate: true }
)

const toggleCollapsed = () => userStore.toggleCollapsed()

const goToConfigCenter = () => {
  router.push('/system/config?tab=theme')
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

// 清掉已关闭窗口的句柄：用户反复打开外链会不断累积（窗口关闭后引用不释放）。
// 保留仍打开的窗口，使业务页刷新后能重新请求 token。
const pruneClosedTokenWindows = () => {
  for (let i = pendingTokenWindows.length - 1; i >= 0; i--) {
    const entry = pendingTokenWindows[i]
    if (!entry || entry.win.closed) pendingTokenWindows.splice(i, 1)
  }
}

const onTokenRequest = (event: MessageEvent) => {
  const data = event.data as { type?: string } | null
  if (data?.type !== 'adminx:request-token') return
  pruneClosedTokenWindows()
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
      pruneClosedTokenWindows()
      pendingTokenWindows.push({ win, origin: expectedOrigin })
    }
  }
}

const handleMenuClick = (key: string) => {
  if (!key) return
  const item = findSidebarItem(key, sidebarMenus.value)
  // 先记下它属于哪个一级菜单：外链/内嵌（push 到 /iframe）之后路径不再能反推归属
  const topKey = topKeyOfItem(key)
  if (topKey) clickedTopKey.value = topKey
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
  clickedTopKey.value = key
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
  } else {
    message.warning('该菜单尚未发布前端页面')
  }
}

// 顶栏点击：mix 的顶栏是一级菜单（选中后跳首个二级菜单），top 的顶栏就是完整菜单树
const handleTopNavSelect = (key: string) => {
  if (layout.value === 'mix') handleTopMenuClick(key)
  else handleMenuClick(key)
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
  fetchUnreadCount()
  startUnreadPolling()
})

onUnmounted(() => {
  stopUnreadPolling()
})
</script>

<style scoped>
/* 三种布局统一：外层纵向（顶栏在上，下面一行），行内放侧栏 + 主区。
   主区不放进 v-if 分支，切布局才不会重建当前页面。 */
.admin-wrapper {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.layout-body {
  display: flex;
  flex: 1;
  min-height: 0;
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
  height: var(--admin-header-height, 64px);
  min-height: var(--admin-header-height, 64px);
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

/* 页头图标按钮：固定 40×40（不写死的话文字按钮是 46×32，和铃铛对不齐）。
   这里刻意不留任何气泡：a-tooltip 在贴顶页头会反复翻转让位、晃；换成原生 title
   又会冒出系统深色提示框，鼠标停/移、点一下都闪一下。无障碍名字交给 aria-label，
   齿轮 / 铃铛的图标含义本身就自解释。 */
.header-icon-btn {
  width: 40px;
  height: 40px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

/* 点一下会闪的默认装饰也一并压平：点击水波纹画的那圈 box-shadow、按下态偏深的底色 */
.header-icon-btn :deep(.ant-wave) {
  display: none;
}
.header-icon-btn:active {
  background-color: rgba(0, 0, 0, 0.06);
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