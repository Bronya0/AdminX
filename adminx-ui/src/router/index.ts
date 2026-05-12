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
import MenuList from '@/views/system/MenuList.vue'
import ConfigList from '@/views/config/ConfigList.vue'

// 监控
import DashboardView from '@/views/dashboard/DashboardView.vue'
import SystemMonitor from '@/views/monitor/SystemMonitor.vue'
import ComponentStatus from '@/views/monitor/ComponentStatus.vue'
import ClusterNodes from '@/views/cluster/ClusterNodes.vue'

// WebService
import WebServiceConfig from '@/views/webservice/WebServiceConfig.vue'
import ScheduleJob from '@/views/webservice/ScheduleJob.vue'

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
          meta: { title: '系统管理', icon: 'SettingOutlined' },
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
              path: 'menus',
              name: 'menus',
              component: MenuList,
              meta: { title: '菜单管理', permission: 'menu:menu:list' },
            },
            {
              path: 'resources',
              name: 'system-resources',
              component: SystemMonitor,
              meta: { title: '系统资源', permission: 'monitor:view' },
            },
            {
              path: 'components',
              name: 'system-components',
              component: ComponentStatus,
              meta: { title: '组件管理', permission: 'monitor:view' },
            },
          ],
        },
        {
          path: 'config',
          name: 'config',
          component: ConfigList,
          meta: { title: '配置中心', icon: 'AppstoreOutlined', permission: 'config_center:config:list' },
        },
        {
          path: 'cluster',
          name: 'cluster',
          meta: { title: '集群管理', icon: 'ClusterOutlined' },
          children: [
            {
              path: 'nodes',
              name: 'cluster-nodes',
              component: ClusterNodes,
              meta: { title: '节点管理', permission: 'cluster:clusternode:list' },
            },
          ],
        },
        {
          path: 'webservice',
          name: 'webservice',
          meta: { title: 'WebService', icon: 'CloudOutlined' },
          children: [
            {
              path: 'config',
              name: 'webservice-config',
              component: WebServiceConfig,
              meta: { title: '服务配置', permission: 'webservice:config:list' },
            },
            {
              path: 'job',
              name: 'webservice-job',
              component: ScheduleJob,
              meta: { title: '定时任务', permission: 'webservice:job:list' },
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
router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()

  // 公开页面直接放行
  if (to.meta.public) {
    next()
    return
  }

  // 未登录跳转到登录页
  if (!userStore.isLoggedIn) {
    next('/login')
    return
  }

  // 已登录但没有用户信息或菜单数据，获取用户信息
  if (!userStore.user || !userStore.menus || userStore.menus.length === 0) {
    try {
      await userStore.fetchUserInfo()
    } catch (e) {
      message.error('获取用户信息失败')
      userStore.logout()
      next('/login')
      return
    }
  }

  // 权限检查
  const requiredPermission = to.meta.permission as string
  if (requiredPermission && !userStore.hasPermission(requiredPermission)) {
    message.error('没有权限访问该页面')
    next('/dashboard')
    return
  }

  next()
})

export default router
