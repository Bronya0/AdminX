import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { message } from 'ant-design-vue'

// 布局组件（始终需要，保持静态导入）
import AdminLayout from '@/layouts/AdminLayout.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/login/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/dashboard/DashboardView.vue'),
          meta: { title: '仪表盘', icon: 'DashboardOutlined' },
        },
        {
          path: 'system',
          name: 'system',
          meta: { title: '系统管理', icon: 'SettingOutlined', permission: 'system:view' },
          children: [
            {
              path: 'users',
              name: 'users',
              component: () => import('@/views/system/UserList.vue'),
              meta: { title: '用户管理', permission: 'accounts:user:list' },
            },
            {
              path: 'roles',
              name: 'roles',
              component: () => import('@/views/system/RoleList.vue'),
              meta: { title: '角色管理', permission: 'accounts:role:list' },
            },
            {
              path: 'permissions',
              name: 'permissions',
              component: () => import('@/views/system/PermissionList.vue'),
              meta: { title: '权限管理', icon: 'SafetyOutlined', permission: 'accounts:permission:list' },
            },
            {
              path: 'config',
              name: 'system-config',
              component: () => import('@/views/config/ConfigCenter.vue'),
              meta: { title: '配置中心', icon: 'ControlOutlined', permission: 'config_center:config:list' },
            },
            {
              path: 'resources',
              name: 'system-resources',
              component: () => import('@/views/monitor/SystemMonitor.vue'),
              meta: { title: '系统资源', permission: 'monitor:resource:list' },
            },
            {
              path: 'components',
              name: 'system-components',
              component: () => import('@/views/monitor/ComponentStatus.vue'),
              meta: { title: '组件管理', permission: 'monitor:resource:list' },
            },
            {
              path: 'nodes',
              name: 'system-nodes',
              component: () => import('@/views/cluster/ClusterNodes.vue'),
              meta: { title: '节点管理', permission: 'cluster:node:list' },
            },
            {
              path: 'scheduler',
              name: 'system-scheduler',
              component: () => import('@/views/scheduler/ScheduleJobList.vue'),
              meta: { title: '定时任务', icon: 'ClockCircleOutlined', permission: 'webservice:schedulejob:list' },
            },
            {
              path: 'notification',
              name: 'system-notification',
              component: () => import('@/views/notification/NotificationCenter.vue'),
              meta: { title: '通知中心', icon: 'BellOutlined', permission: 'notification:notification:list' },
            },
            {
              path: 'theme',
              redirect: '/system/config',
            },
          ],
        },
        {
          path: 'audit',
          name: 'audit',
          meta: { title: '安全审计', icon: 'AuditOutlined', permission: 'audit:view' },
          children: [
            {
              path: 'log',
              name: 'audit-log',
              component: () => import('@/views/audit/AuditLogList.vue'),
              meta: { title: '操作审计', permission: 'audit:auditlog:list' },
            },
            {
              path: 'login-log',
              name: 'audit-login-log',
              component: () => import('@/views/audit/LoginLogList.vue'),
              meta: { title: '登录日志', permission: 'accounts:userloginlog:list' },
            },
          ],
        },
        {
          path: 'iframe',
          name: 'iframe',
          component: () => import('@/views/iframe/IframeView.vue'),
          meta: { title: '外部页面' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFound.vue'),
    },
  ],
})

// 路由守卫
router.beforeEach(async (to, from) => {
  const userStore = useUserStore()

  // 公开页面直接放行
  if (to.meta.public) {
    return true
  }

  // 未登录跳转到登录页
  if (!userStore.isLoggedIn) {
    return '/login'
  }

  // 已登录但没有用户信息或菜单数据，获取用户信息
  if (!userStore.user || !userStore.menus || userStore.menus.length === 0) {
    try {
      await Promise.all([
        userStore.fetchUserInfo(),
        userStore.fetchSiteInfo(),
      ])
    } catch (e) {
      message.error('获取用户信息失败')
      userStore.clearToken()
      return '/login'
    }
  }

  // 权限检查
  const requiredPermission = to.meta.permission as string
  if (requiredPermission && !userStore.hasPermission(requiredPermission)) {
    message.error('没有权限访问该页面')
    const fallback = userStore.user?.home_page || userStore.menus?.[0]?.path
    if (fallback && fallback !== to.path) return fallback
    return '/dashboard'
  }

  // 访问根路径时，按 home_page 跳转
  if (to.path === '/') {
    const target = userStore.user?.home_page || userStore.menus?.[0]?.path || '/dashboard'
    if (target !== to.path) return target
  }

  return true
})

export default router
