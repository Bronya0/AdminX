import { request } from '@/utils/request'
import type { ScheduleJob, JobLog, PaginatedResponse } from '@/types'

// 定时任务 API
export const scheduleJobApi = {
  getJobs: (params?: { page?: number; size?: number; search?: string; command_type?: string; trigger_type?: string; is_active?: boolean }): Promise<PaginatedResponse<ScheduleJob>> =>
    request.get('/jobs/', { params }),
  get: (id: string) =>
    request.get(`/jobs/${id}/`),
  create: (data: Partial<ScheduleJob>) =>
    request.post('/jobs/', data),
  update: (id: string, data: Partial<ScheduleJob>) =>
    request.put(`/jobs/${id}/`, data),
  remove: (id: string) =>
    request.delete(`/jobs/${id}/`),
  runOnce: (id: string): Promise<{ result?: string }> =>
    request.post(`/jobs/${id}/run_once/`),
  getStatus: (): Promise<{ running: boolean; job_count: number }> =>
    request.get('/jobs/status/'),
  reload: () =>
    request.post('/jobs/reload/'),
  getLogs: (params?: Record<string, any>) =>
    request.get('/jobs/logs/', { params }),
}

export const jobLogApi = {
  list: (params?: Record<string, any>): Promise<PaginatedResponse<JobLog>> =>
    request.get('/jobs/logs/', { params }),
}
