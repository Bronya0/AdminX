import type { Menu, SidebarItem } from '@/types'

/**
 * 将扁平菜单列表构建为树形结构。
 *
 * 由于 Menu 模型的 `path` 字段覆盖了 treebeard 内部物化路径，
 * get_parent() 不可用，parent 字段始终为 None。
 *
 * 改用路由 path 前缀匹配推断层级关系。
 */
export function flatMenusToTree(menus: Menu[]): Menu[] {
  if (!menus.length) return []

  const sorted = [...menus].sort((a, b) => a.sort_order - b.sort_order)
  const nodeMap = new Map<string, Menu & { children: Menu[] }>()
  const roots: Menu[] = []

  for (const menu of sorted) {
    nodeMap.set(menu.id, { ...menu, children: [] })
  }

  for (const menu of sorted) {
    const node = nodeMap.get(menu.id)!
    const parent = _findParentByPathPrefix(node, sorted, nodeMap)

    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  }

  return roots
}

/** 用路由 path 前缀匹配找父节点：/system/user -> 父级可能是 /system */
function _findParentByPathPrefix(
  menu: Menu,
  allMenus: Menu[],
  nodeMap: Map<string, Menu & { children: Menu[] }>,
): (Menu & { children: Menu[] }) | null {
  const p = menu.path
  if (!p || p === '/') return null

  const parts = p.split('/').filter(Boolean)
  for (let i = parts.length - 1; i >= 1; i--) {
    const prefix = '/' + parts.slice(0, i).join('/')
    for (const m of allMenus) {
      if (m.path === prefix && m.depth === menu.depth - 1 && nodeMap.has(m.id)) {
        return nodeMap.get(m.id)!
      }
    }
  }
  return null
}

/**
 * 按 depth 排序构建树。
 * 先排 depth=1（成为根），depth>1 时通过路由 path 前缀找父节点。
 */
export function flatMenusToTreeByDepth(menus: Menu[]): Menu[] {
  if (!menus.length) return []

  // 先按 depth 排序（父节点一定先于子节点），同 depth 按 sort_order
  const sorted = [...menus].sort((a, b) => {
    if (a.depth !== b.depth) return a.depth - b.depth
    return a.sort_order - b.sort_order
  })

  const roots: Menu[] = []
  const nodeMap = new Map<string, Menu & { children: Menu[] }>()

  for (const menu of sorted) {
    const node: Menu & { children: Menu[] } = { ...menu, children: [] }
    nodeMap.set(menu.id, node)

    if (menu.depth <= 1) {
      roots.push(node)
    } else {
      const parent = _findParentByPathPrefix(menu, sorted, nodeMap)
      if (parent) {
        parent.children.push(node)
      } else {
        roots.push(node)
      }
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
    sort_order: menu.sort_order,
    path: menu.path,
    permission_code: menu.permission_code,
    allowed_paths: menu.allowed_paths,
    depth: menu.depth,
    children: menu.children ? menusToTreeData(menu.children, keyField) : undefined,
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

    // 优先使用默认图标映射，否则用数据库图标名，最后兜底
    const iconName = defaultIcons[menu.name] || menu.icon || 'FileOutlined'

    const item: SidebarItem = {
      key: menu.path,
      title: menu.name,
      icon: iconName,
    }

    if (children && children.length > 0) {
      item.children = children
    }

    result.push(item)
  }

  return result
}

/**
 * 将 Vue Router 路由配置递归转换为 SidebarItem[]，
 * 同时根据权限过滤。这样侧边栏和路由结构保持完全一致。
 */
export function routesToSidebar(
  routes: any[],
  hasPermission: (code: string) => boolean,
  parentPath = '',
): SidebarItem[] {
  const result: SidebarItem[] = []

  for (const route of routes) {
    const meta = route.meta || {}

    // 跳过没有 title 的路由（布局容器、重定向等）
    if (!meta.title) continue

    // 权限检查
    const perm = meta.permission as string | undefined
    if (perm && !hasPermission(perm)) continue

    // 构建完整路径
    const fullPath = parentPath ? `${parentPath}/${route.path}` : `/${route.path}`

    const item: SidebarItem = {
      key: fullPath.replace(/\/+/g, '/'),
      title: meta.title,
      icon: defaultIcons[meta.title] || meta.icon || 'FileOutlined',
    }

    if (route.children && route.children.length > 0) {
      const children = routesToSidebar(route.children, hasPermission, item.key)
      if (children.length > 0) {
        item.children = children
      }
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
        return { ...node, children: filteredChildren }
      }
      return null
    })
    .filter(Boolean) as any[]
}