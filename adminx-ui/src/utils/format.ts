/** 日期时间格式化工具 */

/**
 * 将 ISO 8601 时间字符串格式化为 "YYYY-MM-DD HH:mm:ss"
 * 输入: "2026-05-14T16:49:54.226083+08:00"
 * 输出: "2026-05-14 16:49:54"
 */
export function formatDateTime(raw: string | null | undefined): string {
  if (!raw) return '-'
  const d = new Date(raw)
  if (isNaN(d.getTime())) return raw // 解析失败则原样返回
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/**
 * 格式化为相对时间描述
 * 输入: "2026-05-14T16:49:54.226083+08:00"
 * 输出: "3分钟前" / "2小时前" / "3天前" / "2026-05-14"
 */
export function formatRelativeTime(raw: string | null | undefined): string {
  if (!raw) return '-'
  const d = new Date(raw)
  if (isNaN(d.getTime())) return raw
  const now = Date.now()
  const diff = now - d.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (seconds < 60) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 30) return `${days}天前`
  return formatDateTime(raw).split(' ')[0] ?? raw // 只返回日期部分
}
