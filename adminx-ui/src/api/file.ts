import { request } from '@/utils/request'
import type { FileRecord, PaginatedResponse } from '@/types'

// 文件中心 API
export const fileApi = {
  list: (params?: { page?: number; size?: number }): Promise<PaginatedResponse<FileRecord>> =>
    request.get('/files/records/', { params }),

  // 上传：multipart 字段名必须是 file（后端流式读取该字段）
  upload: (file: File): Promise<FileRecord> => {
    const form = new FormData()
    form.append('file', file)
    return request.post('/files/upload/', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  // 下载走鉴权接口（需 JWT + files 权限），不能直接用 <a href>：
  // 取回 blob 再触发保存
  download: async (record: FileRecord): Promise<void> => {
    const blob = await request.get<Blob>(`/files/records/${record.id}/download/`, {
      responseType: 'blob',
    })
    const url = URL.createObjectURL(blob as unknown as Blob)
    const link = document.createElement('a')
    link.href = url
    link.download = record.original_name
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  },

  remove: (id: string): Promise<void> =>
    request.delete(`/files/records/${id}/`),
}
