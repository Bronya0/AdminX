import { request } from '@/utils/request'
import type { AuditLog, PaginatedResponse } from '@/types'

export const auditApi = {
  /** 获取审计日志列表 */
  getAuditLogs: (params?: {
    page?: number
    size?: number
    search?: string
    action?: string
    model_name?: string
    created_at__gte?: string
    created_at__lte?: string
  }): Promise<PaginatedResponse<AuditLog>> =>
    request.get('/audit/', { params }),
}
