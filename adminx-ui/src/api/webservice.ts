import { request } from '@/utils/request'
import type { ScheduleJob, JobLog, PaginatedResponse } from '@/types'

// 定时任务 API
export const scheduleJobApi = {
  getJobs: (params?: { page?: number; size?: number; search?: string }): Promise<PaginatedResponse<ScheduleJob>> =>
    request.get('/jobs/', { params }),
  get: (id: string) =>
    request.get(`/jobs/${id}/`),
  create: (data: Partial<ScheduleJob>) =>
    request.post('/jobs/', data),
  update: (id: string, data: Partial<ScheduleJob>) =>
    request.patch(`/jobs/${id}/`, data),
  remove: (id: string) =>
    request.delete(`/jobs/${id}/`),
  runOnce: (id: string): Promise<{ result?: string }> =>
    request.post(`/jobs/${id}/run_once/`),
  getStatus: (): Promise<{ scheduler_alive: boolean; status: string }> =>
    request.get('/jobs/status/'),
  reload: () =>
    request.post('/jobs/reload/'),
  getLogs: (params?: Record<string, any>): Promise<PaginatedResponse<JobLog>> =>
    request.get('/jobs/logs/', { params }),
}
