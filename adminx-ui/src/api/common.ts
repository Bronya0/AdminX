import { request } from '@/utils/request'

// 组件监控 API
export const commonApi = {
  // 健康检查
  health: (): Promise<{
    status: string
    cpu_percent: number
    memory: { total: number; available: number; percent: number; used: number; free: number }
    disk: { total: number; used: number; free: number; percent: number }
    network: { bytes_sent: number; bytes_recv: number }
  }> => request.get('/common/health/'),

  // 缓存统计
  cacheStats: (): Promise<{
    backend: string
    used_memory?: string
    keys?: string
    uptime_days?: string
    hit_rate?: string
    msg?: string
  }> => request.get('/common/cache/stats/'),

  // 清理缓存
  cacheClear: (prefix?: string): Promise<{ cleared: number | string }> =>
    request.post('/common/cache/clear/', { prefix }),
}