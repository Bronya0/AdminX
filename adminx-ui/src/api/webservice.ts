import { request } from '@/utils/request'
import type { ScheduleJob, JobLog, PaginatedResponse } from '@/types'

// 定时任务 API
export const scheduleJobApi = {
  getJobs: (params?: { page?: number; size?: number; search?: string; command_type?: string; trigger_type?: string; is_active?: boolean }): Promise<PaginatedResponse<ScheduleJob>> =>
    request.get('/webservice/jobs/', { params }),

  getJob: (id: string): Promise<ScheduleJob> =>
    request.get(`/webservice/jobs/${id}/`),

  createJob: (data: Partial<ScheduleJob>): Promise<ScheduleJob> =>
    request.post('/webservice/jobs/', data),

  updateJob: (id: string, data: Partial<ScheduleJob>): Promise<ScheduleJob> =>
    request.put(`/webservice/jobs/${id}/`, data),

  deleteJob: (id: string): Promise<void> =>
    request.delete(`/webservice/jobs/${id}/`),

  runOnce: (id: string): Promise<{ result: string }> =>
    request.post(`/webservice/jobs/${id}/run_once/`),

  getStatus: (): Promise<{ running: boolean; job_count: number; jobs: any[] }> =>
    request.get('/webservice/jobs/status/'),

  reloadJobs: (): Promise<void> =>
    request.post('/webservice/jobs/reload/'),
}

// 任务日志 API
export const jobLogApi = {
  list: (params?: { page?: number; size?: number; job?: string }): Promise<PaginatedResponse<JobLog>> =>
    request.get('/webservice/job-logs/', { params }),
}
