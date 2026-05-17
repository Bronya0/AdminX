import { request } from '@/utils/request'

export interface DashboardStats {
  user_count: number
  role_count: number
  menu_count: number
  cpu_usage: number
  cpu_cores: number
  memory_usage: number
  memory_total: number
  memory_used: number
  online_nodes: number
  offline_nodes: number
  maintenance_nodes: number
  recent_logs: {
    username: string
    ip: string
    success: boolean
    message: string
    created_at: string
  }[]
}

export const dashboardApi = {
  getStats: (): Promise<DashboardStats> =>
    request.get('/common/dashboard/stats/'),
}