// API 类型定义

// 通用响应结构
export interface ApiResponse<T = any> {
  code: number
  msg: string
  data: T
}

// 分页响应
export interface PaginatedResponse<T> {
  count: number
  next: string | null
  previous: string | null
  results: T[]
}

// 用户相关
export interface User {
  id: string
  username: string
  email: string
  phone: string
  avatar: string
  desc?: string
  is_active: boolean
  is_superuser?: boolean
  is_online?: boolean
  last_activity?: string | null
  roles: string[]
  role_names: string[]
  date_joined: string
  last_login: string | null
}

export interface LoginRequest {
  username: string
  password: string
  captcha_id?: string
  captcha_text?: string
}

export interface LoginResponse {
  access: string
  refresh: string
  user: User
}

export interface UserInfo {
  user: User
  permissions: string[]
  menus: Menu[]
}

// 角色相关
export interface Role {
  id: string
  name: string
  code: string
  desc: string
  is_active: boolean
  permissions?: string[]
  business_permissions: string[]
  menus: string[]
  created_at: string
  updated_at: string
}

// 权限相关
export interface Permission {
  id: number
  name: string
  codename: string
  content_type: number
  content_type_name: string
  app_label: string
}

export interface BusinessPermission {
  id: string
  app_label: string
  codename: string
  name: string
  desc: string
  created_at: string
}

export interface BusinessCommand {
  id: string
  app_label: string
  name: string
  icon: string
  menu_path: string
  allowed_paths: string
  description: string
  is_active: boolean
  created_at: string
  updated_at: string
}

// 菜单相关
export interface Menu {
  id: string
  code: string
  name: string
  icon: string
  path: string
  component: string
  permission_code: string
  menu_type: 'menu' | 'button' | 'iframe'
  is_active: boolean
  is_visible: boolean
  sort_order: number
  depth: number
  numchild: number
  parent?: string
  children?: Menu[]
  allowed_paths: string
  created_at: string
  updated_at: string
}

// 配置相关
export interface Config {
  id: string
  key: string
  value: string
  value_type: 'string' | 'int' | 'bool' | 'json' | 'encrypted' | 'options'
  encrypted_value: string
  desc: string
  group: string
  is_active: boolean
  display_value: string
  options: any
  created_at: string
  updated_at: string
}

// 集群节点相关
export interface ClusterNode {
  id: string
  name: string
  host: string
  port: number
  role: 'master' | 'slave'
  status: 'online' | 'offline' | 'maintenance'
  version: string
  last_heartbeat: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface ClusterOverview {
  total: number
  online: number
  offline: number
  nodes: ClusterNode[]
}

// 系统资源监控
export interface SystemResources {
  cpu: {
    percent: number
    count: number
  }
  memory: {
    total: number
    available: number
    percent: number
    used: number
    free: number
  }
  disk: Array<{
    device: string
    mountpoint: string
    fstype: string
    total: number
    used: number
    free: number
    percent: number
  }>
  disk_io: {
    read_count: number
    write_count: number
    read_bytes: number
    write_bytes: number
    read_time: number
    write_time: number
  }
  network_io: {
    bytes_sent: number
    bytes_recv: number
    packets_sent: number
    packets_recv: number
    errin: number
    errout: number
    dropin: number
    dropout: number
  }
  load_avg: number[] | null
  timestamp: number
}

export interface NetstatInfo {
  LISTEN: number
  ESTABLISHED: number
  TIME_WAIT: number
  CLOSE_WAIT: number
  OTHER: number
}

// 定时任务相关
export interface ScheduleJob {
  id: string
  name: string
  command_type: 'python' | 'shell'
  handler: string
  command: string
  trigger_type: 'cron' | 'interval' | 'date'
  trigger_config: string
  args: string
  kwargs: string
  is_active: boolean
  last_run: { status: string; result: string; started_at: string; finished_at: string } | null
  created_at: string
  updated_at: string
}

export interface JobLog {
  id: string
  job: string
  job_name: string
  status: 'running' | 'success' | 'failed'
  result: string
  started_at: string
  finished_at: string | null
}

// 通知
export interface Notification {
  id: string
  title: string
  content: string
  notification_type: 'info' | 'success' | 'warning' | 'error'
  is_read: boolean
  created_at: string
}

// Webhook 配置
export interface WebhookConfig {
  id: string
  name: string
  url: string
  secret: string
  events: string
  is_active: boolean
  created_at: string
  updated_at: string
}

// Webhook 发送日志
export interface WebhookLog {
  id: string
  webhook: string
  webhook_name: string
  notification: string | null
  status: 'success' | 'failed'
  response_status: number | null
  response_body: string
  error_message: string
  created_at: string
}

// 审计日志
export interface AuditLog {
  id: string
  operator: string
  object_repr: string
  model_name: string
  action: string
  diff_summary: string
  created_at: string
}

// 登录日志
export interface LoginLog {
  id: string
  user: string | null
  username: string
  ip: string
  user_agent: string
  success: boolean
  message: string
  created_at: string
}

// 验证码
export interface CaptchaResponse {
  captcha_id: string
  svg: string
}

// 侧边栏/导航菜单项（由 Menu 转换而来）
export interface SidebarItem {
  key: string
  title: string
  icon?: string
  children?: SidebarItem[]
}

// 主题配置
export interface ThemeConfig {
  primaryColor: string
  isDark: boolean
  layout: 'side' | 'top' | 'mix'
  collapsed: boolean
  showLogo: boolean
  showBreadcrumb: boolean
  showTabs: boolean
  showFooter: boolean
  showVersion: boolean
  borderRadius: number
  fontSize: number
}
