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

    // Getters
    const isLoggedIn = computed(() => !!token.value)
    const username = computed(() => user.value?.username || '')
    const avatar = computed(() => user.value?.avatar || '')
    const hasPermission = computed(() => {
      return (code: string) => {
        if (!code) return true
        if (user.value?.is_superuser) return true
        return permissions.value.includes(code)
      }
    })

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
      console.log('Store user set to:', user.value)
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
        // 从配置中心同步主题色
        if (res.site_theme_color) {
          theme.value.primaryColor = res.site_theme_color
          document.documentElement.style.setProperty('--primary-color', res.site_theme_color)
        }
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
      toggleDarkMode,
    }
  },
  {
    persist: {
      key: 'user-store',
      pick: ['token', 'refreshToken', 'theme'],
    },
  }
)
