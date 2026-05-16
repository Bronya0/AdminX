import { request } from '@/utils/request'
import type { Notification, PaginatedResponse } from '@/types'

export const notificationApi = {
  list: (params?: object): Promise<PaginatedResponse<Notification>> =>
    request.get('/notification/messages/', { params }),

  markRead: (id: string): Promise<void> =>
    request.post('/notification/messages/' + id + '/mark_read/'),

  markAllRead: (): Promise<void> =>
    request.post('/notification/messages/mark_all_read/'),

  unreadCount: (): Promise<{ count: number }> =>
    request.get('/notification/messages/unread_count/'),
}
