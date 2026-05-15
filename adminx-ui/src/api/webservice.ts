import { request } from '@/utils/request'
import type { WebService, ScheduleJob, JobLog, PaginatedResponse } from '@/types'

// WebService 管理 API
export const webserviceApi = {
  // 获取 WebService 列表
  getServices: (params?: { page?: number; search?: string }): Promise<PaginatedResponse<WebService>> =>
    request.get('/webservice/configs/', { params }),

  // 获取 WebService 详情
  getService: (id: string): Promise<WebService> =>
    request.get(`/webservice/configs/${id}/`),

  // 创建 WebService
  createService: (data: Partial<WebService>): Promise<WebService> =>
    request.post('/webservice/configs/', data),

  // 更新 WebService
  updateService: (id: string, data: Partial<WebService>): Promise<WebService> =>
    request.put(`/webservice/configs/${id}/`, data),

  // 删除 WebService
  deleteService: (id: string): Promise<void> =>
    request.delete(`/webservice/configs/${id}/`),

  // 调用 WebService
  invokeService: (id: string, params?: Record<string, any>): Promise<{ result: string; cost_ms: number }> =>
    request.post(`/webservice/configs/${id}/invoke/`, { params }),
}

// 定时任务 API
export const scheduleJobApi = {
  // 获取任务列表
  getJobs: (params?: { page?: number; size?: number; search?: string; command_type?: string; trigger_type?: string; is_active?: boolean }): Promise<PaginatedResponse<ScheduleJob>> =>
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
  reloadJobs: (): Promise<void> =>
    request.post('/webservice/jobs/reload/'),
}

// 任务日志 API
export const jobLogApi = {
  list: (params?: { page?: number; size?: number; job?: string }): Promise<PaginatedResponse<JobLog>> =>
    request.get('/webservice/job-logs/', { params }),
}