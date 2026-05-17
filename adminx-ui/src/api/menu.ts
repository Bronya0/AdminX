import { request } from '@/utils/request'
import type { Menu } from '@/types'

// 菜单管理 API
export const menuApi = {
  // 获取菜单列表（扁平，不分页）
  getMenus: (params?: { search?: string }): Promise<Menu[]> =>
    request.get('/menu/', { params }),

  // 获取菜单树
  getMenuTree: (): Promise<Menu[]> =>
    request.get('/menu/tree/'),

  // 获取菜单详情
  getMenu: (id: string): Promise<Menu> =>
    request.get(`/menu/${id}/`),

  // 创建菜单
  createMenu: (data: Partial<Menu>): Promise<Menu> =>
    request.post('/menu/', data),

  // 更新菜单
  updateMenu: (id: string, data: Partial<Menu>): Promise<Menu> =>
    request.put(`/menu/${id}/`, data),

  // 删除菜单
  deleteMenu: (id: string): Promise<void> =>
    request.delete(`/menu/${id}/`),

  // 移动菜单
  moveMenu: (data: { id: string; target_id: string; position: 'first-child' | 'last-child' | 'left' | 'right' }): Promise<void> =>
    request.post('/menu/move/', data),
}
