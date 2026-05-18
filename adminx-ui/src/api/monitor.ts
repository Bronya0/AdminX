import { request } from '@/utils/request'
import type { SystemResources, NetstatInfo, SystemResourceHistory } from '@/types'

// 监控 API
export const monitorApi = {
  getSystemResources: (): Promise<SystemResources> =>
    request.get('/monitor/resources/'),

  getSystemResourceHistory: (params?: { range?: string; interval?: string }): Promise<SystemResourceHistory> =>
    request.get('/monitor/resources/history/', { params }),

  getNetstat: (): Promise<NetstatInfo> =>
    request.get('/monitor/netstat/'),
}
