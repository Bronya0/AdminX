import { request } from '@/utils/request'
import type { ThemeConfig, ThemeExportFile } from '@/types'

// 全局主题设置 API（存在服务端配置中心，全站一份，不分用户）
export const themeApi = {
  // 获取全局主题（公共端点，登录页也需要先应用主题）
  get: (): Promise<ThemeConfig> =>
    request.get('/common/theme/'),

  // 覆盖式保存：传入的字段覆盖服务端当前值，返回保存后的完整主题
  save: (data: ThemeConfig): Promise<ThemeConfig> =>
    request.post('/common/theme/', data),

  // 导出当前全局主题（文件内容）
  exportTheme: (): Promise<ThemeExportFile> =>
    request.get('/common/theme/export/'),

  // 导入主题文件（file 为解析后的 JSON，格式由服务端校验），返回合并后的完整主题
  importTheme: (file: unknown): Promise<ThemeConfig> =>
    request.post('/common/theme/import/', file),
}
