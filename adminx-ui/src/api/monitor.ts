import { request } from '@/utils/request'
import type { SystemResources, NetstatInfo } from '@/types'

// 监控 API
export const monitorApi = {
  getSystemResources: (): Promise<SystemResources> =>
    request.get('/monitor/resources/'),

  getNetstat: (): Promise<NetstatInfo> =>
    request.get('/monitor/netstat/'),
}
