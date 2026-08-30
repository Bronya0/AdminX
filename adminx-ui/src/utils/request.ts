import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosResponse, type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { message } from 'ant-design-vue'
import { useUserStore } from '@/stores/user'
import router from '@/router'

const appBase = import.meta.env.BASE_URL.replace(/\/$/, '')
const defaultApiBaseURL = `${appBase}/api/v1`

// 防止并发 401 响应导致重复跳转登录页
let isRedirectingToLogin = false

// access token 过期时，用 refresh token 自动换新。
// 同一时间只允许一个刷新请求，其余并发请求挂起等待新 token 后重放。
let isRefreshing = false
let refreshQueue: Array<{ resolve: (token: string) => void; reject: (e: unknown) => void }> = []

function flushRefreshQueue(token: string | null, error: unknown) {
  isRefreshing = false
  const q = refreshQueue
  refreshQueue = []
  q.forEach(({ resolve, reject }) => {
    if (token) resolve(token)
    else reject(error)
  })
}

// 登出：清状态 + 跳登录（避免与响应拦截器中的 401 处理重复）
function forceLogout(msg?: string) {
  const userStore = useUserStore()
  userStore.clearToken()
  if (!isRedirectingToLogin) {
    isRedirectingToLogin = true
    if (msg) message.error(msg)
    router.push('/login').finally(() => {
      isRedirectingToLogin = false
    })
  }
}

// 登录/刷新/登出接口自身失败不触发 token 刷新重试：
// - 登录失败重试会造成"输错密码被误判登录过期"；
// - 登出失败走刷新会导致重放的登出报文携带已被轮换作废的旧 refresh，
//   真正存活的 refresh 永远吊销不掉。
const NO_AUTH_RETRY_PATHS = ['/accounts/login/', '/accounts/refresh/', '/accounts/logout/']

function isNoRetryUrl(url?: string): boolean {
  if (!url) return false
  return NO_AUTH_RETRY_PATHS.some((p) => url.includes(p))
}

// 非 401 错误提示的短窗口去重：并发失败请求只弹一次同文案 toast
let lastToast = { msg: '', ts: 0 }
function showErrorOnce(msg: string) {
  const text = msg || '请求失败'
  const now = Date.now()
  if (text === lastToast.msg && now - lastToast.ts < 1500) return
  lastToast = { msg: text, ts: now }
  message.error(text)
}

// 创建 axios 实例
const apiClient: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || defaultApiBaseURL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
  },
  (error: AxiosError) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    // 防御非 JSON 响应（204/Blob/代理错误页等）：
    // 无 code 字段（Blob、纯文本、第三方代理响应）时原样透传，交给调用方处理
    const body = response.data as { code?: number; msg?: string; data?: any } | undefined
    const code = body?.code

    // 业务成功：code < 400（包括 200, 201, 204 等）
    if (typeof code === 'number' && code >= 200 && code < 400) {
      return body?.data
    }
    if (typeof code !== 'number') {
      return body as any
    }

    // 认证失败（token 过期/无效）→ 尝试用 refresh token 换新并重放请求
    if (code === 401) {
      const cfg = response.config as InternalAxiosRequestConfig & { _retried?: boolean }
      // 登录/刷新/登出接口自身的失败：不重试、不强制登出（如密码错误）
      if (isNoRetryUrl(cfg?.url)) {
        message.error(body?.msg || '认证失败')
        return Promise.reject(new Error(body?.msg || '认证失败'))
      }
      if (cfg?._retried) {
        forceLogout('登录已过期，请重新登录')
        return Promise.reject(new Error('认证失败'))
      }
      return handleTokenExpired(response.config)
    }

    // 其他业务错误 — 同文案短窗口去重，避免并发失败弹一串 toast
    showErrorOnce(body?.msg || '请求失败')
    return Promise.reject(new Error(body?.msg || '请求失败'))
  },
  async (error: AxiosError) => {
    const { response, config } = error

    if (!response) {
      if (axios.isCancel(error)) return Promise.reject(error)
      showErrorOnce('网络错误，请检查网络连接')
      return Promise.reject(error)
    }

    const { status, data } = response
    const responseData = data as { msg?: string; detail?: string }
    const msg = responseData?.msg || responseData?.detail || '请求失败'
    const cfg = config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined
    const isAuthEndpoint = isNoRetryUrl(cfg?.url)

    // access token 过期：HTTP 401，尝试 refresh 后重放（登录/刷新/登出端点除外）
    if (status === 401 && !isAuthEndpoint && cfg?._retried !== true) {
      return handleTokenExpired(config || {})
    }

    switch (status) {
      case 401:
        if (isAuthEndpoint) {
          // 登录/刷新/登出自身失败（HTTP 层）：直接提示，不强制登出
          message.error(msg)
        } else {
          // 已经重试过仍失败 → 强制登出
          forceLogout('登录已过期，请重新登录')
        }
        break
      case 403:
        showErrorOnce('没有权限执行此操作')
        break
      case 404:
        showErrorOnce('请求的资源不存在')
        break
      case 500:
      case 502:
      case 503:
        showErrorOnce('服务器错误，请稍后重试')
        break
      default:
        showErrorOnce(msg)
    }

    return Promise.reject(error)
  }
)

// 统一处理 token 过期：刷新或跳登录。
// config 为触发过期的请求配置，刷新成功后会用新 token 重放该请求。
function handleTokenExpired(config: AxiosRequestConfig): Promise<unknown> {
  const userStore = useUserStore()
  const refreshToken = userStore.refreshToken

  // 没有 refresh token，直接登出
  if (!refreshToken) {
    forceLogout('登录已过期，请重新登录')
    return Promise.reject(new Error('认证失败'))
  }

  // 已有刷新在进行中，挂起等待结果，新 token 到手后用其重放原请求
  if (isRefreshing) {
    return new Promise<string>((resolve, reject) => {
      refreshQueue.push({ resolve, reject })
    }).then((token) => replayWithToken(config, token))
  }

  isRefreshing = true
  return apiClient
    .post<{ access: string; refresh: string }>('/accounts/refresh/', { refresh: refreshToken })
    .then((res) => {
      const payload = res as unknown as { access: string; refresh: string }
      userStore.setToken(payload.access, payload.refresh)
      flushRefreshQueue(payload.access, null)
      // 重放原请求（标记 _retried，防止重放后再次 401 时无限刷新）
      return replayWithToken(config, payload.access)
    })
    .catch((e) => {
      flushRefreshQueue(null, e)
      // 不传 msg：拦截器已提示过 refresh 失败原因，避免双 toast
      forceLogout()
      return Promise.reject(e)
    })
}

// 用新 token 重放原请求（保留原 transformRequest 等配置，避免 FormData 等被错误序列化）
function replayWithToken(config: AxiosRequestConfig, token: string) {
  return apiClient({
    ...config,
    headers: { ...config.headers, Authorization: `Bearer ${token}` },
    _retried: true,
  } as AxiosRequestConfig & { _retried?: boolean })
}

// 封装请求方法
export const request = {
  get: <T>(url: string, config?: AxiosRequestConfig): Promise<T> =>
    apiClient.get(url, config),

  post: <T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> =>
    apiClient.post(url, data, config),

  put: <T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> =>
    apiClient.put(url, data, config),

  patch: <T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> =>
    apiClient.patch(url, data, config),

  delete: <T>(url: string, config?: AxiosRequestConfig): Promise<T> =>
    apiClient.delete(url, config),
}

export default apiClient
