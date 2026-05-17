import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { message } from 'ant-design-vue'

// 布局组件
import AdminLayout from '@/layouts/AdminLayout.vue'

// 页面组件
import LoginView from '@/views/login/LoginView.vue'

// 系统管理
import UserList from '@/views/system/UserList.vue'
import RoleList from '@/views/system/RoleList.vue'
import PermissionList from '@/views/system/PermissionList.vue'

import ConfigCenter from '@/views/config/ConfigCenter.vue'

// 监控
import DashboardView from '@/views/dashboard/DashboardView.vue'
import SystemMonitor from '@/views/monitor/SystemMonitor.vue'
import ComponentStatus from '@/views/monitor/ComponentStatus.vue'
import ClusterNodes from '@/views/cluster/ClusterNodes.vue'

// 通知中心
import NotificationCenter from '@/views/notification/NotificationCenter.vue'

// 定时任务
import ScheduleJobList from '@/views/scheduler/ScheduleJobList.vue'

// 审计
import AuditLogList from '@/views/audit/AuditLogList.vue'
import LoginLogList from '@/views/audit/LoginLogList.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: DashboardView,
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
              component: UserList,
              meta: { title: '用户管理', permission: 'accounts:user:list' },
            },
            {
              path: 'roles',
              name: 'roles',
              component: RoleList,
              meta: { title: '角色管理', permission: 'accounts:role:list' },
            },
            {
              path: 'permissions',
              name: 'permissions',
              component: PermissionList,
              meta: { title: '权限管理', icon: 'SafetyOutlined', permission: 'accounts:permission:list' },
            },
            {
              path: 'config',
              name: 'system-config',
              component: ConfigCenter,
              meta: { title: '配置中心', icon: 'ControlOutlined', permission: 'config_center:config:list' },
            },
            {
              path: 'resources',
              name: 'system-resources',
              component: SystemMonitor,
              meta: { title: '系统资源', permission: 'monitor:resource:list' },
            },
            {
              path: 'components',
              name: 'system-components',
              component: ComponentStatus,
              meta: { title: '组件管理', permission: 'monitor:resource:list' },
            },
            {
              path: 'nodes',
              name: 'system-nodes',
              component: ClusterNodes,
              meta: { title: '节点管理', permission: 'cluster:node:list' },
            },
            {
              path: 'scheduler',
              name: 'system-scheduler',
              component: ScheduleJobList,
              meta: { title: '定时任务', icon: 'ClockCircleOutlined', permission: 'webservice:schedulejob:list' },
            },
            {
              path: 'notification',
              name: 'system-notification',
              component: NotificationCenter,
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
              component: AuditLogList,
              meta: { title: '操作审计', permission: 'audit:auditlog:list' },
            },
            {
              path: 'login-log',
              name: 'audit-login-log',
              component: LoginLogList,
              meta: { title: '登录日志', permission: 'accounts:userloginlog:list' },
            },
          ],
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
      userStore.logout()
      return '/login'
    }
  }

  // 权限检查
  const requiredPermission = to.meta.permission as string
  if (requiredPermission && !userStore.hasPermission(requiredPermission)) {
    message.error('没有权限访问该页面')
    return '/dashboard'
  }

  // 访问根路径时跳转
  if (to.path === '/') {
    if (userStore.user?.home_page) {
      return userStore.user.home_page
    }
    const firstPath = userStore.menus?.[0]?.path
    if (firstPath) {
      return firstPath
    }
  }

  return true
})

export default router
