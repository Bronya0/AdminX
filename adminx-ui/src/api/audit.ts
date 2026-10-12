import { request } from '@/utils/request'
import type { AuditLog, PaginatedResponse } from '@/types'

export const auditApi = {
  /** 获取审计日志列表 */
  getAuditLogs: (params?: {
    page?: number
    size?: number
    action?: string
    model?: string
    start?: string
    end?: string
  }): Promise<PaginatedResponse<AuditLog>> =>
    request.get('/audit/', { params }),
}
