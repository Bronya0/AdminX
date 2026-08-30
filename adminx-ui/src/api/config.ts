import { request } from '@/utils/request'
import type { Config, PaginatedResponse } from '@/types'

// 配置中心 API
export const configApi = {
  // 获取配置列表
  getConfigs: (params?: { page?: number; size?: number; search?: string; group?: string; value_type?: string; desc?: string; is_active?: boolean; is_encrypted?: boolean }): Promise<PaginatedResponse<Config>> =>
    request.get('/config/', { params }),


  // 创建配置
  createConfig: (data: Partial<Config>): Promise<Config> =>
    request.post('/config/', data),

  // 更新配置
  updateConfig: (id: number, data: Partial<Config>): Promise<Config> =>
    request.patch(`/config/${id}/`, data),

  // 删除配置
  deleteConfig: (id: number): Promise<void> =>
    request.delete(`/config/${id}/`),


  // 获取所有分组列表
  getGroups: (): Promise<string[]> =>
    request.get('/config/groups/'),

}
