<template>
  <div class="dashboard">
    <!-- ========== 顶部欢迎横幅 ========== -->
    <a-card :bordered="false" class="welcome-card">
      <div class="welcome-inner">
        <div class="welcome-left">
          <div class="welcome-greeting">
            <span class="greeting-icon">{{ greetingIcon }}</span>
            <span class="greeting-text">{{ greeting }}{{ username }}</span>
          </div>
          <div class="welcome-sub">
            <span class="site-name">{{ siteName }}</span>
            <span class="version-badge">v{{ appVersion }}</span>
          </div>
        </div>
        <div class="welcome-right">
          <div class="current-time">{{ currentTime }}</div>
          <div class="current-date">{{ currentDate }}</div>
        </div>
      </div>
    </a-card>

    <!-- ========== 统计卡片 ========== -->
    <a-row :gutter="[16, 16]" class="section">
      <a-col v-for="item in statCards" :key="item.key" :xs="24" :sm="12" :lg="6">
        <a-card :bordered="false" class="stat-card" :style="statCardStyle(item.color)">
          <div class="stat-card-inner">
            <div class="stat-icon" :style="{ background: item.color + '15', color: item.color }">
              <component :is="item.icon" />
            </div>
            <div class="stat-info">
              <div class="stat-value" :style="{ color: item.color }">{{ item.value }}</div>
              <div class="stat-label">{{ item.label }}</div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- ========== 系统资源 + 集群状态 ========== -->
    <a-row :gutter="[16, 16]" class="section">
      <!-- 系统资源 -->
      <a-col :xs="24" :lg="14">
        <a-card :bordered="false" title="系统资源" class="section-card">
          <a-row :gutter="[24, 24]">
            <a-col :span="12">
              <div class="resource-item">
                <div class="resource-header">
                  <span class="resource-title">CPU 使用率</span>
                  <span class="resource-percent" :style="{ color: getPercentColor(stats.cpuUsage) }">
                    {{ Math.round(stats.cpuUsage) }}%
                  </span>
                </div>
                <a-progress
                  :percent="Math.round(stats.cpuUsage)"
                  :stroke-color="getPercentColor(stats.cpuUsage)"
                  :trail-color="getTrailColor(stats.cpuUsage)"
                  size="small"
                  :show-info="false"
                />
                <div class="resource-footer">
                  <span class="resource-detail">物理核心: {{ cpuCores }}</span>
                </div>
              </div>
            </a-col>
            <a-col :span="12">
              <div class="resource-item">
                <div class="resource-header">
                  <span class="resource-title">内存使用率</span>
                  <span class="resource-percent" :style="{ color: getPercentColor(stats.memoryUsage) }">
                    {{ Math.round(stats.memoryUsage) }}%
                  </span>
                </div>
                <a-progress
                  :percent="Math.round(stats.memoryUsage)"
                  :stroke-color="getPercentColor(stats.memoryUsage)"
                  :trail-color="getTrailColor(stats.memoryUsage)"
                  size="small"
                  :show-info="false"
                />
                <div class="resource-footer">
                  <span class="resource-detail">{{ formatBytes(stats.memoryUsed) }} / {{ formatBytes(stats.memoryTotal) }}</span>
                </div>
              </div>
            </a-col>
          </a-row>
        </a-card>
      </a-col>

      <!-- 集群状态 -->
      <a-col :xs="24" :lg="10">
        <a-card :bordered="false" title="集群状态" class="section-card">
          <div class="cluster-grid">
            <div
              v-for="item in clusterStatusList"
              :key="item.key"
              class="cluster-item"
              @click="item.path && $router.push(item.path)"
            >
              <div class="cluster-icon" :class="item.key" :style="{ background: item.bg, color: item.color }">
                <component :is="item.icon" />
              </div>
              <div class="cluster-info">
                <div class="cluster-value" :style="{ color: item.color }">{{ item.value }}</div>
                <div class="cluster-label">{{ item.label }}</div>
              </div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- ========== 快捷入口 ========== -->
    <a-row :gutter="[16, 16]" class="section">
      <a-col :xs="24">
        <a-card :bordered="false" title="快捷入口" class="section-card">
          <a-row :gutter="[16, 16]">
            <a-col v-for="item in quickLinks" :key="item.path" :xs="12" :sm="8" :md="6" :lg="4">
              <div class="quick-link-card" @click="$router.push(item.path)">
                <div class="quick-link-icon" :style="{ background: item.bg, color: item.color }">
                  <component :is="item.icon" />
                </div>
                <div class="quick-link-title">{{ item.label }}</div>
              </div>
            </a-col>
          </a-row>
        </a-card>
      </a-col>
    </a-row>

    <!-- ========== 最近登录记录 ========== -->
    <a-row :gutter="[16, 16]" class="section">
      <a-col :xs="24">
        <a-card :bordered="false" title="最近登录记录" class="section-card">
          <a-table
            :columns="logColumns"
            :data-source="loginLogs"
            :pagination="false"
            size="middle"
            row-key="id"
            :locale="emptyLocale"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'success'">
                <a-tag :color="record.success ? 'success' : 'error'" class="status-tag">
                  {{ record.success ? '成功' : '失败' }}
                </a-tag>
              </template>
              <template v-if="column.key === 'created_at'">
                {{ formatDateTime(record.created_at) }}
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import {
  UserOutlined,
  TeamOutlined,
  ClusterOutlined,
  MenuOutlined,
  MonitorOutlined,
  SettingOutlined,
  SafetyOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ToolOutlined,
} from '@ant-design/icons-vue'
import { useUserStore } from '@/stores/user'
import { monitorApi } from '@/api/monitor'
import { clusterApi } from '@/api/cluster'
import { userApi, roleApi, loginLogApi } from '@/api/auth'
import { menuApi } from '@/api/menu'
import { formatDateTime } from '@/utils/format'
import type { LoginLog, SystemResources } from '@/types'

// ── 用户存储 ──
const userStore = useUserStore()
const username = computed(() => userStore.username || '未知用户')
const siteName = computed(() => userStore.siteName)
const appVersion = computed(() => userStore.appVersion)

// ── 问候语 ──
const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了，'
  if (hour < 9) return '早上好，'
  if (hour < 12) return '上午好，'
  if (hour < 14) return '中午好，'
  if (hour < 18) return '下午好，'
  return '晚上好，'
})

const greetingIcon = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6 || hour >= 19) return '🌙'
  if (hour < 12) return '🌅'
  return '☀️'
})

// ── 当前时间 ──
const currentTime = ref('')
const currentDate = ref('')
function updateClock() {
  const now = new Date()
  currentTime.value = now.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  currentDate.value = now.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' })
}

// ── 统计数据 ──
const stats = reactive({
  userCount: 0,
  roleCount: 0,
  menuCount: 0,
  onlineNodes: 0,
  offlineNodes: 0,
  maintenanceNodes: 0,
  cpuUsage: 0,
  memoryUsage: 0,
  memoryTotal: 0,
  memoryUsed: 0,
})
const cpuCores = ref(1)

const statCards = computed(() => [
  { key: 'users',  label: '用户总数',  value: stats.userCount,     icon: UserOutlined,    color: '#1890ff', path: '/system/users' },
  { key: 'roles',  label: '角色数量',  value: stats.roleCount,     icon: TeamOutlined,    color: '#722ed1', path: '/system/roles' },
  { key: 'nodes',  label: '在线节点',  value: stats.onlineNodes,   icon: ClusterOutlined,  color: '#52c41a', path: '/system/nodes' },
  { key: 'menus',  label: '菜单总数',  value: stats.menuCount,     icon: MenuOutlined,    color: '#fa8c16', path: '/system/permissions' },
])

const clusterStatusList = computed(() => [
  { key: 'online',       label: '在线节点', value: stats.onlineNodes,      icon: CheckCircleOutlined,  color: '#52c41a', bg: '#f6ffed', path: '/system/nodes' },
  { key: 'offline',      label: '离线节点', value: stats.offlineNodes,     icon: CloseCircleOutlined,  color: '#ff4d4f', bg: '#fff2f0', path: '/system/nodes' },
  { key: 'maintenance',  label: '维护中',   value: stats.maintenanceNodes, icon: ToolOutlined,          color: '#faad14', bg: '#fff7e6', path: '/system/nodes' },
])

// ── 快捷入口（路径全部修正，不再 404） ──
const quickLinks = [
  { label: '用户管理',  path: '/system/users',       icon: UserOutlined,    color: '#1890ff', bg: '#e6f7ff' },
  { label: '角色管理',  path: '/system/roles',       icon: TeamOutlined,    color: '#722ed1', bg: '#f9f0ff' },
  { label: '节点管理',  path: '/system/nodes',       icon: ClusterOutlined, color: '#52c41a', bg: '#f6ffed' },
  { label: '资源监控',  path: '/system/resources',   icon: MonitorOutlined, color: '#fa8c16', bg: '#fff7e6' },
  { label: '配置中心',  path: '/system/config',      icon: SettingOutlined, color: '#eb2f96', bg: '#fff0f6' },
  { label: '权限管理',  path: '/system/permissions', icon: SafetyOutlined,  color: '#fa541c', bg: '#fff2e8' },
  { label: '定时任务',  path: '/system/scheduler',   icon: ToolOutlined,    color: '#2f54eb', bg: '#f0f5ff' },
]

// ── 登录日志 ──
const loginLogs = ref<LoginLog[]>([])
const logColumns = [
  { title: '用户名',   dataIndex: 'username', key: 'username', width: 120 },
  { title: 'IP 地址',  dataIndex: 'ip',       key: 'ip',       width: 150 },
  { title: '状态',     key: 'success',        width: 80 },
  { title: '消息',     dataIndex: 'message',  key: 'message',  ellipsis: true },
  { title: '时间',     key: 'created_at',     width: 180 },
]
const emptyLocale = { emptyText: '暂无登录记录' }

// ── 工具函数 ──
function getPercentColor(p: number): string {
  if (p > 90) return '#cf1322'
  if (p > 70) return '#faad14'
  return '#52c41a'
}
function getTrailColor(p: number): string {
  if (p > 90) return '#fff1f0'
  if (p > 70) return '#fffbe6'
  return '#f6ffed'
}
function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}
function statCardStyle(color: string) {
  return { borderLeft: `4px solid ${color}` }
}

// ── 数据加载 ──
async function loadStats() {
  try {
    const [resources, clusterOverview, users, roles, menus, logs] = await Promise.all([
      monitorApi.getSystemResources() as Promise<SystemResources>,
      clusterApi.getOverview(),
      userApi.getUsers({ page: 1 }),
      roleApi.getRoles({ page: 1 }),
      menuApi.getMenus(),
      loginLogApi.getLoginLogs({ page: 1 }),
    ])

    stats.cpuUsage = resources.cpu.percent
    stats.memoryUsage = resources.memory.percent
    stats.memoryTotal = resources.memory.total
    stats.memoryUsed = resources.memory.used
    cpuCores.value = resources.cpu.count

    stats.onlineNodes = clusterOverview.online
    stats.offlineNodes = clusterOverview.offline
    stats.maintenanceNodes = (clusterOverview.nodes || []).filter(n => n.status === 'maintenance').length

    stats.userCount = users.count
    stats.roleCount = roles.count
    stats.menuCount = menus.length

    loginLogs.value = logs.results.slice(0, 5)
  } catch (e) {
    console.error('加载仪表盘数据失败', e)
  }
}

let timer: number | null = null

onMounted(() => {
  updateClock()
  loadStats()
  timer = window.setInterval(() => {
    updateClock()
    loadStats()
  }, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.dashboard {
  max-width: 1400px;
  margin: 0 auto;
}

.section {
  margin-top: 8px;
}

/* ── 欢迎横幅 ── */
.welcome-card {
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  margin-bottom: 8px;
}
.welcome-card :deep(.ant-card-body) {
  padding: 28px 32px;
}
.welcome-inner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}
.welcome-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.welcome-greeting {
  display: flex;
  align-items: center;
  gap: 10px;
}
.greeting-icon {
  font-size: 28px;
  line-height: 1;
}
.greeting-text {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: 0.5px;
}
.welcome-sub {
  display: flex;
  align-items: center;
  gap: 12px;
}
.site-name {
  font-size: 14px;
  opacity: 0.85;
}
.version-badge {
  display: inline-block;
  padding: 1px 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  font-size: 12px;
  font-weight: 500;
}
.welcome-right {
  text-align: right;
  flex-shrink: 0;
}
.current-time {
  font-size: 28px;
  font-weight: 600;
  letter-spacing: 2px;
  line-height: 1.2;
}
.current-date {
  font-size: 13px;
  opacity: 0.75;
  margin-top: 4px;
}

/* ── 统计卡片 ── */
.stat-card {
  border-radius: 10px;
  transition: all 0.3s ease;
  cursor: pointer;
}
.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
}
.stat-card :deep(.ant-card-body) {
  padding: 20px 24px;
}
.stat-card-inner {
  display: flex;
  align-items: center;
  gap: 18px;
}
.stat-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}
.stat-info {
  flex: 1;
  min-width: 0;
}
.stat-value {
  font-size: 26px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 13px;
  color: #8c8c8c;
  margin-top: 4px;
}

/* ── 区块卡片 ── */
.section-card {
  border-radius: 10px;
}
.section-card :deep(.ant-card-head) {
  border-bottom: 1px solid #f0f0f0;
  padding: 0 20px;
  min-height: 48px;
}
.section-card :deep(.ant-card-head-title) {
  font-size: 15px;
  font-weight: 600;
  padding: 12px 0;
}
.section-card :deep(.ant-card-body) {
  padding: 20px;
}

/* ── 系统资源 ── */
.resource-item {
  padding: 4px 0;
}
.resource-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.resource-title {
  font-size: 13px;
  color: #595959;
}
.resource-percent {
  font-size: 15px;
  font-weight: 600;
}
.resource-footer {
  margin-top: 6px;
  font-size: 12px;
  color: #bfbfbf;
}
.resource-detail {
  font-size: 12px;
}

/* ── 集群状态 ── */
.cluster-grid {
  display: flex;
  justify-content: space-around;
  gap: 12px;
}
.cluster-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}
.cluster-item:hover {
  background: #fafafa;
}
.cluster-icon {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}
.cluster-info {
  text-align: center;
}
.cluster-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
.cluster-label {
  font-size: 12px;
  color: #8c8c8c;
  margin-top: 2px;
}

/* ── 快捷入口 ── */
.quick-link-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 20px 8px;
  border-radius: 10px;
  background: #fafafa;
  cursor: pointer;
  transition: all 0.25s ease;
  border: 1px solid transparent;
}
.quick-link-card:hover {
  background: #fff;
  border-color: #e8e8e8;
  transform: translateY(-3px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.08);
}
.quick-link-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}
.quick-link-title {
  font-size: 13px;
  color: #434343;
  font-weight: 500;
}

/* ── 表格 ── */
.status-tag {
  border-radius: 4px;
  font-size: 12px;
  padding: 0 8px;
  line-height: 22px;
}

/* ── 暗色模式适配 ── */
:root.dark .welcome-card {
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}
:root.dark .stat-card:hover {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}
:root.dark .quick-link-card {
  background: #1f1f1f;
}
:root.dark .quick-link-card:hover {
  background: #262626;
  border-color: #434343;
}
:root.dark .resource-title {
  color: #a6a6a6;
}
:root.dark .cluster-item:hover {
  background: #262626;
}
</style>
