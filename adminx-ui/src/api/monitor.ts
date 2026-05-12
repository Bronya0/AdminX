import { request } from '@/utils/request'
import type { SystemResources, NetstatInfo, WebService, ScheduleJob, PaginatedResponse } from '@/types'

// 监控 API
export const monitorApi = {
  // 获取系统资源
  getSystemResources: (): Promise<SystemResources> =>
    request.get('/monitor/resources/'),

  // 获取网络连接统计
  getNetstat: (): Promise<NetstatInfo> =>
    request.get('/monitor/netstat/'),
}

// WebService API
export const webserviceApi = {
  // 获取服务列表
  getServices: (params?: { page?: number; search?: string }): Promise<PaginatedResponse<WebService>> =>
    request.get('/webservice/configs/', { params }),

  // 获取服务详情
  getService: (id: string): Promise<WebService> =>
    request.get(`/webservice/configs/${id}/`),

  // 创建服务
  createService: (data: Partial<WebService>): Promise<WebService> =>
    request.post('/webservice/configs/', data),

  // 更新服务
  updateService: (id: string, data: Partial<WebService>): Promise<WebService> =>
    request.put(`/webservice/configs/${id}/`, data),

  // 删除服务
  deleteService: (id: string): Promise<void> =>
    request.delete(`/webservice/configs/${id}/`),

  // 调用服务
  invokeService: (id: string, params: Record<string, any>): Promise<{ result: string; cost_ms: number }> =>
    request.post(`/webservice/configs/${id}/invoke/`, { params }),
}

// 定时任务 API
export const scheduleJobApi = {
  // 获取任务列表
  getJobs: (params?: { page?: number; search?: string }): Promise<PaginatedResponse<ScheduleJob>> =>
    request.get('/webservice/jobs/', { params }),

  // 获取任务详情
  getJob: (id: string): Promise<ScheduleJob> =>
    request.get(`/webservice/jobs/${id}/`),

  // 创建任务
  createJob: (data: Partial<ScheduleJob>): Promise<ScheduleJob> =>
    request.post('/webservice/jobs/', data),

  // 更新任务
  updateJob: (id: string, data: Partial<ScheduleJob>): Promise<ScheduleJob> =>
    request.put(`/webservice/jobs/${id}/`, data),

  // 删除任务
  deleteJob: (id: string): Promise<void> =>
    request.delete(`/webservice/jobs/${id}/`),

  // 立即执行一次
  runOnce: (id: string): Promise<{ result: string }> =>
    request.post(`/webservice/jobs/${id}/run_once/`),

  // 获取调度器状态
  getStatus: (): Promise<{ running: boolean; job_count: number; jobs: any[] }> =>
    request.get('/webservice/jobs/status/'),

  // 重载所有任务
  reload: (): Promise<void> =>
    request.post('/webservice/jobs/reload/'),
}
