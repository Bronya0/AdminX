import { request } from '@/utils/request'
import type {
  LoginRequest,
  LoginResponse,
  User,
  UserInfo,
  Role,
  Menu,
  PaginatedResponse,
  LoginLog,
} from '@/types'

export const authApi = {
  login: (data: LoginRequest): Promise<LoginResponse> =>
    request.post('/accounts/login/', data),

  logout: (refreshToken: string): Promise<void> =>
    request.post('/accounts/logout/', { refresh: refreshToken }),

  refresh: (refreshToken: string): Promise<{ access: string; refresh: string }> =>
    request.post('/accounts/refresh/', { refresh: refreshToken }),

  getUserInfo: (): Promise<UserInfo> =>
    request.get('/accounts/users/me/'),

  changePassword: (data: { old_password: string; new_password: string }): Promise<void> =>
    request.post('/policy/change-password/', data),

  // 密码策略（登录后任意用户可读，供改密/建用户表单做前端校验）
  getPasswordPolicy: (): Promise<{
    min_length: number
    require_upper: boolean
    require_lower: boolean
    require_digit: boolean
    require_special: boolean
    expire_days: number
    history_count: number
    is_active: boolean
  }> => request.get('/policy/policy/'),

  updateMe: (data: Partial<User>): Promise<User> =>
    request.patch('/accounts/users/me/', data),
}

export const userApi = {
  getUsers: (params?: { page?: number; size?: number; search?: string; is_active?: boolean; role?: string; is_online?: string }): Promise<PaginatedResponse<User>> =>
    request.get('/accounts/users/', { params }),

  getRoleOptions: (): Promise<{ name: string }[]> =>
    request.get('/accounts/users/roles/'),

  getUser: (id: string): Promise<User> =>
    request.get(`/accounts/users/${id}/`),

  createUser: (data: Partial<User> & { password: string; roles?: string[] }): Promise<User> =>
    request.post('/accounts/users/', data),

  updateUser: (id: string, data: Partial<User> & { roles?: string[]; password?: string }): Promise<User> =>
    request.patch(`/accounts/users/${id}/`, data),

  deleteUser: (id: string): Promise<void> =>
    request.delete(`/accounts/users/${id}/`),
}

export const roleApi = {
  getRoles: (params?: { page?: number; size?: number; search?: string; is_active?: boolean; desc?: string }): Promise<PaginatedResponse<Role>> =>
    request.get('/accounts/roles/', { params }),

  getMenuTree: (): Promise<Menu[]> =>
    request.get('/accounts/roles/menu_tree/'),


  getRole: (id: string): Promise<Role> =>
    request.get(`/accounts/roles/${id}/`),

  createRole: (data: Partial<Role>): Promise<Role> =>
    request.post('/accounts/roles/', data),

  updateRole: (id: string, data: Partial<Role>): Promise<Role> =>
    request.patch(`/accounts/roles/${id}/`, data),

  deleteRole: (id: string): Promise<void> =>
    request.delete(`/accounts/roles/${id}/`),

  accessibleMenus: (roles: string[]): Promise<{ path: string; name: string }[]> =>
    request.post('/accounts/roles/accessible_menus/', { roles }),
}


export const loginLogApi = {
  getLoginLogs: (params?: { page?: number; size?: number; search?: string; ip?: string; success?: boolean; created_at__gte?: string; created_at__lte?: string }): Promise<PaginatedResponse<LoginLog>> =>
    request.get('/accounts/login-logs/', { params }),
}

export const captchaApi = {
  getCaptcha: (): Promise<{ captchaId: string; svg: string }> =>
    request.get('/captcha/captcha/').then((res: any) => ({
      captchaId: res.captcha_id,
      svg: res.svg,
    })),

}
