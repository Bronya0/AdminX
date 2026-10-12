import { request } from '@/utils/request'

// Site info
export interface SiteInfo {
  site_name: string
  site_desc: string
  site_logo: string
  site_theme_color: string
  idle_timeout: number
  login_bg_image: string
  app_version: string
  favicon?: string
  // 登录验证码开关（公共端点返回，登录页据此决定是否展示；后端缺省 = 开启）
  login_captcha_required?: boolean
}

// 组件监控 API
export const commonApi = {
  // 站点信息
  getSiteInfo: (): Promise<SiteInfo> =>
    request.get('/common/site-info/'),

  saveSiteInfo: (data: Partial<SiteInfo>): Promise<void> =>
    request.post('/common/site-info/', data),
}