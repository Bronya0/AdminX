import { request } from '@/utils/request'
import type { WebhookConfig, WebhookLog, PaginatedResponse } from '@/types'

export const webhookApi = {
  getList: (params?: { page?: number; size?: number; search?: string }): Promise<PaginatedResponse<WebhookConfig>> =>
    request.get('/notification/webhooks/', { params }),

  getDetail: (id: string): Promise<WebhookConfig> =>
    request.get(`/notification/webhooks/${id}/`),

  create: (data: Partial<WebhookConfig>): Promise<WebhookConfig> =>
    request.post('/notification/webhooks/', data),

  update: (id: string, data: Partial<WebhookConfig>): Promise<WebhookConfig> =>
    request.patch(`/notification/webhooks/${id}/`, data),

  delete: (id: string): Promise<void> =>
    request.delete(`/notification/webhooks/${id}/`),

  test: (id: string): Promise<void> =>
    request.post(`/notification/webhooks/${id}/test/`),

  getLogs: (params?: { page?: number; size?: number; webhook?: string; status?: string }): Promise<PaginatedResponse<WebhookLog>> =>
    request.get('/notification/webhook-logs/', { params }),
}
