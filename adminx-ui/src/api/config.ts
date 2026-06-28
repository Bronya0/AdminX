import { request } from '@/utils/request'
import type { Config, PaginatedResponse } from '@/types'

// 配置中心 API
export const configApi = {
  // 获取配置列表
  getConfigs: (params?: { page?: number; size?: number; search?: string; group?: string; value_type?: string; desc?: string; is_active?: boolean; is_encrypted?: boolean }): Promise<PaginatedResponse<Config>> =>
    request.get('/config/', { params }),

  // 获取配置详情
  getConfig: (id: number): Promise<Config> =>
    request.get(`/config/${id}/`),

  // 创建配置
  createConfig: (data: Partial<Config>): Promise<Config> =>
    request.post('/config/', data),

  // 更新配置
  updateConfig: (id: number, data: Partial<Config>): Promise<Config> =>
    request.patch(`/config/${id}/`, data),

  // 删除配置
  deleteConfig: (id: number): Promise<void> =>
    request.delete(`/config/${id}/`),

  // 按分组获取配置
  getConfigsByGroup: (group: string): Promise<Record<string, any>> =>
    request.get('/config/by_group/', { params: { group } }),

  // 按 key 获取单个配置值（业务容器友好）
  getValue: (key: string): Promise<{ key: string; value: any }> =>
    request.get('/config/get_value/', { params: { key } }),

  // 获取所有分组列表
  getGroups: (): Promise<string[]> =>
    request.get('/config/groups/'),

  // NTP 时间同步
  ntpSync: (): Promise<any> =>
    request.post('/config/ntp/sync/'),
}
