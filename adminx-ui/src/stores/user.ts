import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import { commonApi } from '@/api/common'
import type { User, UserInfo, ThemeConfig, Menu } from '@/types'

// 默认主题配置
const defaultTheme: ThemeConfig = {
  primaryColor: '#1890ff',
  isDark: false,
  layout: 'side',
  collapsed: false,
  showLogo: true,
  showBreadcrumb: true,
  showTabs: true,
  showFooter: true,
  showVersion: true,
  borderRadius: 6,
  fontSize: 14,
}

export const useUserStore = defineStore(
  'user',
  () => {
    // State
    const token = ref<string>('')
    const refreshToken = ref<string>('')
    const user = ref<User | null>(null)
    const permissions = ref<string[]>([])
    const menus = ref<Menu[]>([])
    const theme = ref<ThemeConfig>({ ...defaultTheme })
    const siteName = ref<string>('DjangoAdminX')
    const siteDesc = ref<string>('企业级 Django Admin 框架')
    const siteLogo = ref<string>('')
    const idleTimeout = ref<number>(30) // 会话空闲超时（分钟），0=不超时
    const appVersion = ref<string>('1.0.0')

    // Getters
    const isLoggedIn = computed(() => !!token.value)
    const username = computed(() => user.value?.username || '')
    const avatar = computed(() => user.value?.avatar || '')
    const hasPermission = (code: string): boolean => {
      if (!code) return true
      if (user.value?.is_superuser) return true
      return permissions.value.includes(code)
    }

    // Actions
    const setToken = (access: string, refresh: string) => {
      token.value = access
      refreshToken.value = refresh
    }

    const setUserInfo = (data: UserInfo) => {
      user.value = data.user
      permissions.value = data.permissions
      menus.value = data.menus
    }

    const login = async (username: string, password: string, captchaId?: string, captchaText?: string) => {
      const res = await authApi.login({
        username,
        password,
        captcha_id: captchaId,
        captcha_text: captchaText,
      })
      setToken(res.access, res.refresh)
      // 设置用户信息（登录返回的用户信息）
      user.value = res.user
      return res
    }

    const fetchUserInfo = async () => {
      const res = await authApi.getUserInfo()
      setUserInfo(res)
      return res
    }

    const fetchSiteInfo = async () => {
      try {
        const res = await commonApi.getSiteInfo()
        siteName.value = res.site_name
        siteDesc.value = res.site_desc
        siteLogo.value = res.site_logo
        idleTimeout.value = res.idle_timeout ?? 30
        appVersion.value = res.app_version || '1.0.0'
        // 注：主题色/布局等由前端主题设置页面管理，不再从配置中心同步
      } catch (e) {
        // 使用默认值
      }
    }

    const logout = async () => {
      if (refreshToken.value) {
        try {
          await authApi.logout(refreshToken.value)
        } catch (e) {
          // ignore
        }
      }
      token.value = ''
      refreshToken.value = ''
      user.value = null
      permissions.value = []
      menus.value = []
    }

    const updateTheme = (config: Partial<ThemeConfig>) => {
      theme.value = { ...theme.value, ...config }
    }

    const toggleCollapsed = () => {
      theme.value.collapsed = !theme.value.collapsed
    }

    const setPrimaryColor = (color: string) => {
      theme.value.primaryColor = color
      document.documentElement.style.setProperty('--primary-color', color)
    }

    const clearToken = () => {
      token.value = ''
      refreshToken.value = ''
      user.value = null
      permissions.value = []
      menus.value = []
    }

    const toggleDarkMode = () => {
      theme.value.isDark = !theme.value.isDark
      if (theme.value.isDark) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    }

    return {
      token,
      refreshToken,
      user,
      permissions,
      menus,
      theme,
      siteName,
      siteDesc,
      siteLogo,
      idleTimeout,
      appVersion,
      isLoggedIn,
      username,
      avatar,
      hasPermission,
      setToken,
      setUserInfo,
      login,
      fetchUserInfo,
      fetchSiteInfo,
      logout,
      updateTheme,
      toggleCollapsed,
      setPrimaryColor,
      clearToken,
      toggleDarkMode,
    }
  },
  {
    persist: {
      key: 'user-store',
      pick: ['token', 'refreshToken', 'theme', 'appVersion'],
    },
  }
)
