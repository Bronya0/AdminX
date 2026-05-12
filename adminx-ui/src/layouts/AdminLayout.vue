<template>
  <div class="admin-wrapper" :class="layout">
    <!-- SIDE layout: sidebar only -->
    <SideBar
      v-if="layout === 'side'"
      :menus="sidebarMenus"
      :collapsed="collapsed"
      :selectedKeys="selectedKeys"
      :openKeys="openKeys"
      showLogo
      @select="handleMenuClick"
      @update:openKeys="openKeys = $event"
    />

    <!-- TOP layout: top nav only -->
    <TopNav
      v-if="layout === 'top'"
      :menus="sidebarMenus"
      :selectedKeys="selectedKeys"
      @select="handleMenuClick"
    />

    <!-- MIX layout: top nav + sidebar -->
    <template v-if="layout === 'mix'">
      <TopNav
        :menus="topLevelMenus"
        :selectedKeys="[selectedTopKey]"
        :isMix="true"
        :showLogo="true"
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
          @select="handleMenuClick"
          @update:openKeys="openKeys = $event"
        />
      </div>
    </template>

    <!-- 主区域（所有布局共用） -->
    <div class="admin-main" :class="{ collapsed: layout !== 'top' && collapsed }">
      <!-- 顶部导航栏 -->
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
            <a-tooltip title="切换主题">
              <a-button type="text" @click="toggleTheme">
                <BulbOutlined v-if="!isDark" />
                <BulbFilled v-else />
              </a-button>
            </a-tooltip>
            <a-tooltip title="全屏">
              <a-button type="text" @click="toggleFullscreen">
                <FullscreenOutlined />
              </a-button>
            </a-tooltip>
            <a-dropdown>
              <a-button type="text" style="display: inline-flex; align-items: center; justify-content: center; width: 40px; height: 40px; padding: 0;">
                <a-badge :count="5" :offset="[2, -2]">
                  <BellOutlined style="font-size: 18px;" />
                </a-badge>
              </a-button>
              <template #overlay>
                <a-menu>
                  <a-menu-item>系统通知</a-menu-item>
                  <a-menu-item>消息中心</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
            <a-dropdown>
              <a-space style="cursor: pointer; display: inline-flex; align-items: center;">
                <a-avatar :size="32">
                  <template #icon><UserOutlined /></template>
                </a-avatar>
                <span style="font-size: 14px; color: #333;">{{ username }}</span>
                <DownOutlined style="font-size: 12px; color: #999;" />
              </a-space>
              <template #overlay>
                <a-menu>
                  <a-menu-item @click="goToProfile">
                    <UserOutlined /> 个人中心
                  </a-menu-item>
                  <a-menu-item @click="showLayoutSettings = true">
                    <SettingOutlined /> 系统设置
                  </a-menu-item>
                  <a-menu-divider />
                  <a-menu-item @click="handleLogout">
                    <LogoutOutlined /> 退出登录
                  </a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </a-space>
        </div>
      </div>

      <!-- 内容区 -->
      <div class="admin-content">
        <router-view />
      </div>
    </div>

    <!-- 布局设置弹窗 -->
    <a-modal
      v-model:open="showLayoutSettings"
      title="系统设置"
      :footer="null"
      width="500px"
    >
      <a-form layout="vertical">
        <a-form-item label="布局模式">
          <a-radio-group v-model:value="currentLayout" @change="handleLayoutChange">
            <a-radio-button value="side">
              <MenuFoldOutlined /> 侧边栏
            </a-radio-button>
            <a-radio-button value="top">
              <MenuOutlined /> 顶部导航
            </a-radio-button>
            <a-radio-button value="mix">
              <AppstoreOutlined /> 混合布局
            </a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="主题">
          <a-radio-group v-model:value="currentDark" @change="handleDarkChange">
            <a-radio-button :value="false">
              <BulbOutlined /> 浅色
            </a-radio-button>
            <a-radio-button :value="true">
              <BulbFilled /> 深色
            </a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="主题色">
          <a-input v-model:value="currentColor" type="color" style="width: 60px; padding: 0; height: 32px;" />
          <span style="margin-left: 8px; color: #999;">{{ currentColor }}</span>
        </a-form-item>
      </a-form>
      <div style="margin-top: 16px;">
        <p style="font-size: 12px; color: #999; margin-bottom: 4px;">当前布局说明：</p>
        <p v-if="currentLayout === 'side'" style="font-size: 12px; color: #666;">菜单全部展示在左侧侧边栏</p>
        <p v-if="currentLayout === 'top'" style="font-size: 12px; color: #666;">菜单全部展示在顶部导航栏</p>
        <p v-if="currentLayout === 'mix'" style="font-size: 12px; color: #666;">一级菜单在顶部，子菜单在左侧侧边栏</p>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { message, Modal } from 'ant-design-vue'
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  DashboardOutlined,
  SettingOutlined,
  AppstoreOutlined,
  MenuOutlined,
  UserOutlined,
  BellOutlined,
  LogoutOutlined,
  FullscreenOutlined,
  BulbOutlined,
  BulbFilled,
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
const showLayoutSettings = ref(false)

// 布局设置表单
const currentLayout = ref(layout.value)
const currentDark = ref(isDark.value)
const currentColor = ref(userStore.theme.primaryColor || '#1890ff')

const username = computed(() => userStore.username || '未登录')

// 构建动态菜单树
const sidebarMenus = computed<SidebarItem[]>(() => {
  if (!userStore.menus || userStore.menus.length === 0) return []
  const tree = flatMenusToTreeByDepth(userStore.menus)
  return menusToSidebarItems(tree, (code: string) => userStore.hasPermission(code))
})

// MIX 布局的一级菜单
const topLevelMenus = computed(() => sidebarMenus.value)

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
const toggleTheme = () => userStore.toggleDarkMode()

const toggleFullscreen = () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
  } else {
    document.exitFullscreen()
  }
}

const handleMenuClick = (key: string) => {
  if (!key) return
  router.push(key).catch((err) => {
    // 忽略重复导航错误
    if (err.name !== 'NavigationDuplicated') {
      console.warn(`导航到 "${key}" 失败:`, err)
    }
  })
}

// MIX 布局：点击顶级菜单 -> 跳转到第一个子菜单
const handleTopMenuClick = ({ key }: { key: string }) => {
  selectedTopKey.value = key
  const parent = sidebarMenus.value.find(m => m.key === key)
  let target: string | undefined
  if (parent && parent.children && parent.children.length > 0) {
    target = parent.children[0].key
  } else if (parent) {
    target = parent.key
  }
  if (target) {
    const routeExists = router.getRoutes().some(r => r.path === target)
    if (routeExists) {
      router.push(target)
    }
  }
}

const goToProfile = () => message.info('个人中心功能开发中')

const handleLogout = () => {
  Modal.confirm({
    title: '确认退出',
    content: '确定要退出登录吗？',
    onOk: async () => {
      await userStore.logout()
      router.push('/login')
      message.success('已退出登录')
    },
  })
}

// 布局设置
const handleLayoutChange = (val: any) => {
  currentLayout.value = val.target ? val.target.value : val
  userStore.updateTheme({ layout: currentLayout.value })
  if (currentLayout.value !== 'mix') {
    selectedTopKey.value = ''
  }
}

const handleDarkChange = (val: any) => {
  currentDark.value = val.target ? val.target.value : val
  if (currentDark.value !== isDark.value) {
    toggleTheme()
  }
}

onMounted(() => {
  if (userStore.theme.isDark) {
    document.documentElement.classList.add('dark')
  }
  currentLayout.value = layout.value
  currentDark.value = isDark.value
  currentColor.value = userStore.theme.primaryColor || '#1890ff'
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
  background: #fff;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
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
  background: #f0f2f5;
  padding: 16px;
}

:deep(.ant-radio-group-solid) .ant-radio-button-wrapper {
  min-width: 100px;
  text-align: center;
}
</style>