import { request } from '@/utils/request'
import { downloadBlob } from '@/utils/download'
import type { FileRecord, PaginatedResponse } from '@/types'

// 全局默认超时 30s（utils/request.ts）对整包上传/下载的大文件不够用：
// 后端上限 100MB，30s 传完需要 >3.3MB/s，超时后拦截器提示“网络错误”，用户会误以为断网。
// 按每 MB 给 20s、最少 60s 折算（100MB → 2000s），既有上限也不会永久挂起。
const transferTimeout = (size?: number): number =>
  Math.max(60000, Math.ceil((size || 0) / 1024 / 1024) * 20000)

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
      // 按实际体积换算超时（入参就是 File，能拿到 size）
      timeout: transferTimeout(file.size),
    })
  },

  // 下载走鉴权接口（需 JWT + files 权限），不能直接用 <a href>：
  // 取回 blob 再触发保存。错误响应（JSON 小体）直接存盘会落下一个打不开的
  // “文件”，必须先嗅探 JSON 并抛错走统一错误流。
  download: async (record: FileRecord): Promise<void> => {
    const blob = await request.get<Blob>(`/files/records/${record.id}/download/`, {
      responseType: 'blob',
      // 下载大文件同样会超过 30s 默认值
      timeout: transferTimeout(record.size),
    }) as unknown as Blob
    if (blob?.type?.includes('json')) {
      try {
        const body = JSON.parse(await blob.text())
        const err = new Error(body?.msg || '下载失败') as Error & { code?: number }
        err.code = body?.code
        throw err
      } catch (e) {
        if (e instanceof SyntaxError) throw new Error('下载失败')
        throw e
      }
    }
    downloadBlob(blob, record.original_name)
  },

  remove: (id: string): Promise<void> =>
    request.delete(`/files/records/${id}/`),
}
