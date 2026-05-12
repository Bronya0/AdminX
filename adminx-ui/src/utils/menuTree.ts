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
      if (!parent.children) parent.children = []
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

    if (menu.depth === 1) {
      roots.push(node)
    } else {
      // depth > 1: 用路由 path 前缀匹配找父节点
      const parent = _findParentByPathPrefix(menu, sorted, nodeMap)
      if (parent) {
        if (!parent.children) parent.children = []
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
    icon: menu.icon,
    menu_type: menu.menu_type,
    is_active: menu.is_active,
    is_visible: menu.is_visible,
    code: menu.code,
    sort_order: menu.sort_order,
    path: menu.path,
    component: menu.component,
    permission_code: menu.permission_code,
    depth: menu.depth,
    children: menu.children ? menusToTreeData(menu.children, keyField) : undefined,
  }))
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
    if (!menu.is_visible || !menu.is_active) continue
    if (menu.permission_code && !hasPermission(menu.permission_code)) continue

    let children: SidebarItem[] | undefined
    if (menu.children && menu.children.length > 0) {
      children = menusToSidebarItems(menu.children, hasPermission)
    }

    const item: SidebarItem = {
      key: menu.path,
      title: menu.name,
      icon: menu.icon || undefined,
    }

    if (children && children.length > 0) {
      item.children = children
    }

    result.push(item)
  }

  return result
}

/**
 * 递归过滤树节点，保留标题包含搜索文本的节点。
 * 如果父节点的任一子节点匹配，父节点也会保留。
 */
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