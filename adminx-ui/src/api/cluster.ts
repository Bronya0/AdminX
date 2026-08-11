import { request } from '@/utils/request'
import type { ClusterNode, ClusterOverview, PaginatedResponse, ServiceComponent, SystemComponentsData } from '@/types'

// 集群管理 API
export const clusterApi = {
  getNodes: (params?: { page?: number; size?: number; search?: string; status?: string }): Promise<PaginatedResponse<ClusterNode>> =>
    request.get('/cluster/nodes/', { params }),
  getNode: (id: string): Promise<ClusterNode> =>
    request.get(`/cluster/nodes/${id}/`),
  createNode: (data: Partial<ClusterNode>): Promise<ClusterNode> =>
    request.post('/cluster/nodes/', data),
  updateNode: (id: string, data: Partial<ClusterNode>): Promise<ClusterNode> =>
    request.patch(`/cluster/nodes/${id}/`, data),
  deleteNode: (id: string): Promise<void> =>
    request.delete(`/cluster/nodes/${id}/`),
  getOverview: (): Promise<ClusterOverview> =>
    request.get('/cluster/nodes/overview/'),
}

// 业务组件 API
export const componentApi = {
  list: (params?: { search?: string }): Promise<PaginatedResponse<ServiceComponent>> =>
    request.get('/cluster/components/', { params }),
  get: (id: string): Promise<ServiceComponent> =>
    request.get(`/cluster/components/${id}/`),
  delete: (id: string): Promise<void> =>
    request.delete(`/cluster/components/${id}/`),
  setUpgrade: (id: string, data: { upgrade_version: string; upgrade_url: string; upgrade_checksum?: string }): Promise<any> =>
    request.post(`/cluster/components/${id}/set_upgrade/`, data),
  cancelUpgrade: (id: string): Promise<any> =>
    request.post(`/cluster/components/${id}/cancel_upgrade/`),
  confirmUpgrade: (id: string): Promise<any> =>
    request.post(`/cluster/components/${id}/confirm_upgrade/`),
  setUninstall: (id: string): Promise<any> =>
    request.post(`/cluster/components/${id}/set_uninstall/`),
  cancelUninstall: (id: string): Promise<any> =>
    request.post(`/cluster/components/${id}/cancel_uninstall/`),
  // 系统内置组件状态
  systemComponents: (): Promise<SystemComponentsData> =>
    request.get('/common/components/'),
}

