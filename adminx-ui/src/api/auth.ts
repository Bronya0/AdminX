import { request } from '@/utils/request'
import type {
  LoginRequest,
  LoginResponse,
  User,
  UserInfo,
  Role,
  Permission,
  PaginatedResponse,
  LoginLog,
} from '@/types'

// 登录相关 API
export const authApi = {
  // 登录
  login: (data: LoginRequest): Promise<LoginResponse> =>
    request.post('/accounts/login/', data),

  // 登出
  logout: (refreshToken: string): Promise<void> =>
    request.post('/accounts/logout/', { refresh: refreshToken }),

  // 获取当前用户信息
  getUserInfo: (): Promise<UserInfo> =>
    request.get('/accounts/users/me/'),

  // 修改密码
  changePassword: (data: { old_password: string; new_password: string }): Promise<void> =>
    request.post('/policy/change-password/', data),
}

// 用户管理 API
export const userApi = {
  // 获取用户列表
  getUsers: (params?: { page?: number; size?: number; search?: string; is_active?: boolean; role?: string; is_online?: string }): Promise<PaginatedResponse<User>> =>
    request.get('/accounts/users/', { params }),

  // 获取用户详情
  getUser: (id: string): Promise<User> =>
    request.get(`/accounts/users/${id}/`),

  // 创建用户
  createUser: (data: Partial<User> & { password: string; roles?: string[] }): Promise<User> =>
    request.post('/accounts/users/', data),

  // 更新用户
  updateUser: (id: string, data: Partial<User> & { roles?: string[] }): Promise<User> =>
    request.put(`/accounts/users/${id}/`, data),

  // 删除用户
  deleteUser: (id: string): Promise<void> =>
    request.delete(`/accounts/users/${id}/`),
}

// 角色管理 API
export const roleApi = {
  // 获取角色列表
  getRoles: (params?: { page?: number; size?: number; search?: string; is_active?: boolean; code?: string; desc?: string }): Promise<PaginatedResponse<Role>> =>
    request.get('/accounts/roles/', { params }),

  // 获取角色详情
  getRole: (id: string): Promise<Role> =>
    request.get(`/accounts/roles/${id}/`),

  // 创建角色
  createRole: (data: Partial<Role>): Promise<Role> =>
    request.post('/accounts/roles/', data),

  // 更新角色
  updateRole: (id: string, data: Partial<Role>): Promise<Role> =>
    request.put(`/accounts/roles/${id}/`, data),

  // 删除角色
  deleteRole: (id: string): Promise<void> =>
    request.delete(`/accounts/roles/${id}/`),
}

// 权限管理 API
export const permissionApi = {
  // 获取权限列表（不分页）
  getPermissions: (params?: { search?: string }): Promise<Permission[]> =>
    request.get('/accounts/permissions/', { params }),
}

// 登录日志 API
export const loginLogApi = {
  // 获取登录日志列表
  getLoginLogs: (params?: { page?: number; size?: number; search?: string; ip?: string; success?: boolean; created_at__gte?: string; created_at__lte?: string }): Promise<PaginatedResponse<LoginLog>> =>
    request.get('/accounts/login-logs/', { params }),
}

// 验证码 API
export const captchaApi = {
  // 获取验证码
  getCaptcha: (): Promise<{ captchaId: string; svg: string }> =>
    request.get('/captcha/captcha/', { responseType: 'text' }).then((res: any) => ({
      captchaId: res.headers['x-captcha-id'],
      svg: res.data,
    })),

  // 验证验证码
  verifyCaptcha: (data: { captcha_id: string; captcha_text: string }): Promise<{ verified: boolean }> =>
    request.post('/captcha/captcha/verify/', data),
}
