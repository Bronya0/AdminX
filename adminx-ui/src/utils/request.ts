import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosResponse, type AxiosError } from 'axios'
import { message } from 'ant-design-vue'
import { useUserStore } from '@/stores/user'
import router from '@/router'

const appBase = import.meta.env.BASE_URL.replace(/\/$/, '')
const defaultApiBaseURL = `${appBase}/api/v1`

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
    const { code, msg, data } = response.data

    // 业务成功：code < 400（包括 200, 201, 204 等）
    if (code >= 200 && code < 400) {
      return data
    }

    // 认证失败（token 过期/无效）→ 静默清理，由路由守卫统一引导到登录页
    if (code === 401) {
      const userStore = useUserStore()
      userStore.logout()
      return Promise.reject(new Error(msg || '认证失败'))
    }

    // 其他业务错误 — 仅显示一次通用提示
    message.error(msg || '请求失败')
    return Promise.reject(new Error(msg || '请求失败'))
  },
  (error: AxiosError) => {
    const { response } = error

    if (response) {
      const { status, data } = response
      const responseData = data as { msg?: string; detail?: string }
      const msg = responseData?.msg || responseData?.detail || '请求失败'

      switch (status) {
        case 401:
          message.error('登录已过期，请重新登录')
          const userStore = useUserStore()
          userStore.logout()
          router.push('/login')
          break
        case 403:
          message.error('没有权限执行此操作')
          break
        case 404:
          message.error('请求的资源不存在')
          break
        case 500:
        case 502:
        case 503:
          message.error('服务器错误，请稍后重试')
          break
        default:
          message.error(msg)
      }
    } else {
      message.error('网络错误，请检查网络连接')
    }

    return Promise.reject(error)
  }
)

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
