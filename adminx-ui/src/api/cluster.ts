import { request } from '@/utils/request'
import type { ClusterNode, ClusterOverview, PaginatedResponse } from '@/types'

// 集群管理 API
export const clusterApi = {
  // 获取节点列表
  getNodes: (params?: { page?: number; size?: number; search?: string; status?: string }): Promise<PaginatedResponse<ClusterNode>> =>
    request.get('/cluster/nodes/', { params }),

  // 获取节点详情
  getNode: (id: string): Promise<ClusterNode> =>
    request.get(`/cluster/nodes/${id}/`),

  // 创建节点
  createNode: (data: Partial<ClusterNode>): Promise<ClusterNode> =>
    request.post('/cluster/nodes/', data),

  // 更新节点
  updateNode: (id: string, data: Partial<ClusterNode>): Promise<ClusterNode> =>
    request.put(`/cluster/nodes/${id}/`, data),

  // 删除节点
  deleteNode: (id: string): Promise<void> =>
    request.delete(`/cluster/nodes/${id}/`),

  // 获取集群概览
  getOverview: (): Promise<ClusterOverview> =>
    request.get('/cluster/nodes/overview/'),
}
