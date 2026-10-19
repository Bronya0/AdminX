import type { Menu, SidebarItem } from '@/types'

/**
 * 将扁平菜单列表构建为树形结构。
 *
 * 后端 `/menu/user_tree/` 与 `/accounts/users/me/` 下发的是扁平列表，父子关系有两条线索：
 *   ① `parent_id`：真实字段，菜单保存时由后端写入，取决于层级而不是取决于路径写法；
 *   ② `depth` + 路由 path 前缀：老数据 parent_id 为空时的兜底推断。
 * 一律优先 ①，拿不到父节点时才退回 ②。
 */

/** 归一化菜单 path：去掉 scheme://host，只留路径部分。 */
function _normPath(p: string): string {
  return (p || '').replace(/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\/[^/]+/, '')
}

/**
 * 用路由 path 前缀匹配找父节点：/system/user -> 父级可能是 /system。
 *
 * 比较前必须归一化：外链与 iframe 菜单的 path 是完整 URL（`https://host/a/b`），
 * 直接按 `/` 切分会拼出 `/https:` 这种永远匹配不上的前缀，
 * 结果这类菜单连同它的子树会被当成顶级挂在树根。
 */
function _findParentByPathPrefix(
  menu: Menu,
  allMenus: Menu[],
  nodeMap: Map<number, Menu & { children: Menu[] }>,
): (Menu & { children: Menu[] }) | null {
  const p = _normPath(menu.path)
  if (!p || p === '/') return null

  const parts = p.split('/').filter(Boolean)
  for (let i = parts.length - 1; i >= 1; i--) {
    const prefix = '/' + parts.slice(0, i).join('/')
    for (const m of allMenus) {
      if (_normPath(m.path) === prefix && m.depth === menu.depth - 1 && nodeMap.has(m.id)) {
        return nodeMap.get(m.id)!
      }
    }
  }
  return null
}

/**
 * 找父节点：parent_id 优先（真实关系），path 前缀兜底（老数据 parent_id 为空）。
 * 父节点不在本次下发的菜单里（例如角色只授权了子菜单、或父菜单已停用）时返回 null，
 * 该菜单按顶级渲染，不会整块消失。
 */
function _findParent(
  menu: Menu,
  allMenus: Menu[],
  nodeMap: Map<number, Menu & { children: Menu[] }>,
): (Menu & { children: Menu[] }) | null {
  if (menu.parent_id != null) {
    const byId = nodeMap.get(menu.parent_id)
    if (byId) return byId
  }
  if (menu.depth <= 1) return null
  return _findParentByPathPrefix(menu, allMenus, nodeMap)
}

/**
 * 按 depth 排序构建树（父节点一定先于子节点，才能一次遍历就挂上）。
 */
export function flatMenusToTreeByDepth(menus: Menu[]): Menu[] {
  if (!menus.length) return []

  // 先按 depth 排序（父节点一定先于子节点），同 depth 按 sort_order
  const sorted = [...menus].sort((a, b) => {
    if (a.depth !== b.depth) return a.depth - b.depth
    return a.sort_order - b.sort_order
  })

  const roots: Menu[] = []
  const nodeMap = new Map<number, Menu & { children: Menu[] }>()

  for (const menu of sorted) {
    const node: Menu & { children: Menu[] } = { ...menu, children: [] }
    nodeMap.set(menu.id, node)

    const parent = _findParent(menu, sorted, nodeMap)
    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  }

  return roots
}

/**
 * 将 Menu[] 树转换为 Ant Design `<a-tree>` / `<a-tree-select>` 的 treeData 格式。
 */
export function menusToTreeData(
  menus: Menu[],
  keyField: 'id' | 'code' = 'id',
): any[] {
  return menus.map((menu) => ({
    title: menu.name,
    key: menu[keyField],
    value: menu.id,
    is_active: menu.is_active,
    code: menu.code,
    icon: menu.icon,
    sort_order: menu.sort_order,
    path: menu.path,
    permission_code: menu.permission_code,
    allowed_paths: menu.allowed_paths,
    depth: menu.depth,
    // 叶子节点必须输出 undefined 而不是 []：
    // rc-table 的树形模式只判断 children 字段是否存在（空数组是 truthy），
    // 叶子带 children: [] 会被当成有子节点，渲染出多余的展开图标。
    children: menu.children && menu.children.length ? menusToTreeData(menu.children, keyField) : undefined,
  }))
}

// 默认图标映射（数据库存的是短名如 "User"、"Settings"，ant-design 用全名 "UserOutlined"）
const defaultIcons: Record<string, string> = {
  '系统管理': 'SettingOutlined',
  '角色管理': 'TeamOutlined',
  '用户管理': 'UserOutlined',
  '配置中心': 'ControlOutlined',
  '集群管理': 'ClusterOutlined',
  '节点管理': 'CloudServerOutlined',
  '系统资源': 'MonitorOutlined',
  '系统监控': 'MonitorOutlined',
  '组件管理': 'BlockOutlined',
  '定时任务': 'ClockCircleOutlined',
  '通知中心': 'BellOutlined',
  '安全审计': 'AuditOutlined',
  '操作审计': 'FileSearchOutlined',
  '登录日志': 'LoginOutlined',
}

/**
 * 将 Menu[] 树转换为 SidebarItem[]（用于 AdminLayout menu 渲染），
 * 同时过滤掉无权限、隐藏的菜单。
 */
export function menusToSidebarItems(
  menus: Menu[],
  hasPermission: (code: string) => boolean,
): SidebarItem[] {
  const result: SidebarItem[] = []

  for (const menu of menus) {
    if (!menu.is_visible || !menu.is_active) {
      continue
    }
    if (menu.permission_code && !hasPermission(menu.permission_code)) {
      continue
    }

    let children: SidebarItem[] | undefined
    if (menu.children && menu.children.length > 0) {
      children = menusToSidebarItems(menu.children, hasPermission)
    }

    // 管理员在数据库配置的图标优先，未配置时用默认映射，最后兜底
    const iconName = menu.icon || defaultIcons[menu.name] || 'FileOutlined'

    const isExternal = menu.path?.startsWith('http://') || menu.path?.startsWith('https://') || false
    const item: SidebarItem = {
      key: isExternal ? menu.code : menu.path,
      title: menu.name,
      icon: iconName,
      menu_type: menu.menu_type,
      url: isExternal ? menu.path : undefined,
    }

    if (children && children.length > 0) {
      item.children = children
    }

    result.push(item)
  }

  return result
}

export function filterTreeBySearch(
  tree: any[],
  searchText: string,
  titleKey = 'title',
): any[] {
  if (!searchText) return tree

  const q = searchText.toLowerCase()
  return tree
    .map((node) => {
      const filteredChildren = node.children
        ? filterTreeBySearch(node.children, searchText, titleKey)
        : undefined

      const match = String(node[titleKey]).toLowerCase().includes(q)

      if (match || (filteredChildren && filteredChildren.length > 0)) {
        // 同上：过滤后没有子节点的行不能留空数组，否则会多出展开图标
        return { ...node, children: filteredChildren && filteredChildren.length > 0 ? filteredChildren : undefined }
      }
      return null
    })
    .filter(Boolean) as any[]
}