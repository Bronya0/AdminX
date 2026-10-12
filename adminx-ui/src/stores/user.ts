import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import { commonApi } from '@/api/common'
import { themeApi } from '@/api/theme'
import type { User, UserInfo, ThemeConfig, Menu } from '@/types'

// 默认主题配置
// 导出：主题持久化在 localStorage，旧会话存的 theme 不含后来新增的字段，
// 直接读会拿到 undefined（拼进 CSS 变量即为非法值，行高会退回 auto），
// 所以消费方一律用 defaultTheme 兜底。
export const defaultTheme: ThemeConfig = {
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
  navHeight: 56,
  headerHeight: 64,
  navTopBgColor: '#001529',
  navSideBgColor: '#001529',
  navTextSize: 14,
  navTextAuto: true,
  navTextColor: '#ffffff',
}

// 感知亮度判断（sRGB 权重）：决定导航的文字/边框走深色还是浅色
const isLightColor = (color: string): boolean => {
  const hex = /^#?([0-9a-f]{6})$/i.exec(String(color || '').trim())?.[1]
  if (!hex) return false
  const n = parseInt(hex, 16)
  const r = (n >> 16) & 255
  const g = (n >> 8) & 255
  const b = n & 255
  return (r * 299 + g * 587 + b * 114) / 1000 > 150
}

// refresh token 会话级存储：不进 localStorage（XSS 无法长期续期），
// 但进 sessionStorage —— F5/新标签后仍能正确调用服务端登出吊销会话，
// 关闭标签页即失效。
const REFRESH_KEY = 'adminx_refresh_token'

const readSessionRefresh = (): string => {
  try {
    return sessionStorage.getItem(REFRESH_KEY) || ''
  } catch {
    return ''
  }
}

const writeSessionRefresh = (token: string) => {
  try {
    if (token) sessionStorage.setItem(REFRESH_KEY, token)
    else sessionStorage.removeItem(REFRESH_KEY)
  } catch {
    /* 隐私模式等场景下静默降级为仅内存 */
  }
}

export const useUserStore = defineStore(
  'user',
  () => {
    // State
    const token = ref<string>('')
    const refreshToken = ref<string>(readSessionRefresh())
    const user = ref<User | null>(null)
    const permissions = ref<string[]>([])
    const menus = ref<Menu[]>([])
    const theme = ref<ThemeConfig>({ ...defaultTheme })
    const siteName = ref<string>('AdminX')
    const siteDesc = ref<string>('企业级 Admin 框架')
    const siteLogo = ref<string>('')
    const favicon = ref<string>('')
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

    // 导航区配色：顶栏与侧栏各一套（底色可不同，antd 菜单的 light/dark 主题随之不同）。
    // 文字颜色默认按底色亮度自动取，也可以在主题设置里改成固定值；边框线一律跟随底色。
    const navTokens = (bg: string) => {
      const light = isLightColor(bg)
      return {
        bg,
        menuTheme: (light ? 'light' : 'dark') as 'light' | 'dark',
        text:
          theme.value.navTextAuto || !theme.value.navTextColor
            ? light
              ? 'rgba(0, 0, 0, 0.88)'
              : 'rgba(255, 255, 255, 0.85)'
            : theme.value.navTextColor,
        border: light ? 'rgba(0, 0, 0, 0.06)' : 'rgba(255, 255, 255, 0.1)',
      }
    }
    const navTopBg = computed(() => theme.value.navTopBgColor ?? defaultTheme.navTopBgColor)
    const navSideBg = computed(() => theme.value.navSideBgColor ?? defaultTheme.navSideBgColor)
    const navTop = computed(() => navTokens(navTopBg.value))
    const navSide = computed(() => navTokens(navSideBg.value))

    // Actions
    const setToken = (access: string, refresh: string) => {
      token.value = access
      refreshToken.value = refresh
      writeSessionRefresh(refresh)
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
        favicon.value = res.favicon || ''
        idleTimeout.value = res.idle_timeout ?? 30
        appVersion.value = res.app_version || '1.0.0'
        if (res.site_name) document.title = res.site_name
        if (res.favicon) {
          const link = document.querySelector('link[rel="icon"]') as HTMLLinkElement
          if (link) link.href = res.favicon
        }
        // 注：主题色/布局等由前端主题设置页面管理，不再从配置中心同步
      } catch (e) {
        // 使用默认值
      }
    }

    const logout = async () => {
      // 服务端吊销 refresh token + last_logout 二次失效（仅凭 refresh 也可吊销）
      if (refreshToken.value) {
        try {
          await authApi.logout(refreshToken.value)
        } catch (e) {
          // ignore：服务端不可达也要完成本地清理
        }
      }
      clearToken()
    }

    const updateTheme = (config: Partial<ThemeConfig>) => {
      theme.value = { ...theme.value, ...config }
    }

    // 主题以服务端为准（全站一份）；localStorage 里的 theme 只是首屏缓存。
    const setTheme = (config: ThemeConfig) => {
      theme.value = { ...defaultTheme, ...config }
    }

    // 拉取失败时沿用本地缓存，不打断页面
    const fetchTheme = async () => {
      try {
        setTheme(await themeApi.get())
      } catch {
        /* 沿用本地缓存 */
      }
    }

    const saveTheme = async (config: ThemeConfig) => {
      setTheme(await themeApi.save(config))
    }

    const toggleCollapsed = () => {
      theme.value.collapsed = !theme.value.collapsed
    }

    const clearToken = () => {
      token.value = ''
      refreshToken.value = ''
      writeSessionRefresh('')
      user.value = null
      permissions.value = []
      menus.value = []
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
      favicon,
      idleTimeout,
      appVersion,
      isLoggedIn,
      username,
      avatar,
      hasPermission,
      navTop,
      navSide,
      setToken,
      setUserInfo,
      login,
      fetchUserInfo,
      fetchSiteInfo,
      logout,
      updateTheme,
      toggleCollapsed,
      clearToken,
      setTheme,
      fetchTheme,
      saveTheme,
    }
  },
  {
    persist: {
      key: 'user-store',
      // refreshToken 不持久化到 localStorage：避免 XSS 窃取后长期保持登录，仅保留会话内
      pick: ['token', 'theme', 'appVersion'],
    },
  }
)
