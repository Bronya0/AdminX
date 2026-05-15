import { request } from '@/utils/request'
import type { Notification, PaginatedResponse } from '@/types'

export const notificationApi = {
  /** 获取通知列表 */
  list: (params?: { page?: number; size?: number; search?: string; notification_type?: string; is_read?: boolean }): Promise<PaginatedResponse<Notification>> =>
    request.get('/notification/', { params }),

  /** 标记已读 */
  markRead: (id: string): Promise<void> =>
    request.post(`/notification/${id}/mark_read/`),

  /** 全部标记已读 */
  markAllRead: (): Promise<void> =>
    request.post('/notification/mark_all_read/'),

  /** 未读数量 */
  unreadCount: (): Promise<{ count: number }> =>
    request.get('/notification/unread_count/'),
}
