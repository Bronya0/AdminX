import { request } from '@/utils/request'
import type {
  LoginRequest,
  LoginResponse,
  User,
  UserInfo,
  Role,
  Permission,
  BusinessPermission,
  BusinessCommand,
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

  updateUser: (id: string, data: Partial<User> & { roles?: string[] }): Promise<User> =>
    request.put(`/accounts/users/${id}/`, data),

  deleteUser: (id: string): Promise<void> =>
    request.delete(`/accounts/users/${id}/`),
}

export const roleApi = {
  getRoles: (params?: { page?: number; size?: number; search?: string; is_active?: boolean; desc?: string }): Promise<PaginatedResponse<Role>> =>
    request.get('/accounts/roles/', { params }),

  getMenuTree: (): Promise<any[]> =>
    request.get('/accounts/roles/menu_tree/'),

  getRole: (id: string): Promise<Role> =>
    request.get(`/accounts/roles/${id}/`),

  createRole: (data: Partial<Role>): Promise<Role> =>
    request.post('/accounts/roles/', data),

  updateRole: (id: string, data: Partial<Role>): Promise<Role> =>
    request.put(`/accounts/roles/${id}/`, data),

  patchRole: (id: string, data: Partial<Role>): Promise<Role> =>
    request.patch(`/accounts/roles/${id}/`, data),

  deleteRole: (id: string): Promise<void> =>
    request.delete(`/accounts/roles/${id}/`),

  accessibleMenus: (roles: string[]): Promise<{ path: string; name: string }[]> =>
    request.post('/accounts/roles/accessible_menus/', { roles }),
}

export const permissionApi = {
  getPermissions: (params?: { search?: string }): Promise<Permission[]> =>
    request.get('/accounts/permissions/', { params }),
}

export const businessPermissionApi = {
  list: (params?: { search?: string; app_label?: string }): Promise<BusinessPermission[]> =>
    request.get('/accounts/business-permissions/', { params }),

  create: (data: Partial<BusinessPermission>): Promise<BusinessPermission> =>
    request.post('/accounts/business-permissions/', data),

  update: (id: string, data: Partial<BusinessPermission>): Promise<BusinessPermission> =>
    request.put(`/accounts/business-permissions/${id}/`, data),

  delete: (id: string): Promise<void> =>
    request.delete(`/accounts/business-permissions/${id}/`),
}

export const businessCommandApi = {
  list: (params?: { search?: string; app_label?: string }): Promise<BusinessCommand[]> =>
    request.get('/accounts/business-commands/', { params }),

  create: (data: Partial<BusinessCommand>): Promise<BusinessCommand> =>
    request.post('/accounts/business-commands/', data),

  update: (id: string, data: Partial<BusinessCommand>): Promise<BusinessCommand> =>
    request.put(`/accounts/business-commands/${id}/`, data),

  delete: (id: string): Promise<void> =>
    request.delete(`/accounts/business-commands/${id}/`),
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

  verifyCaptcha: (data: { captcha_id: string; captcha_text: string }): Promise<{ verified: boolean }> =>
    request.post('/captcha/captcha/verify/', data),
}
