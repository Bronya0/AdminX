<template>
  <div class="dashboard">
    <!-- ========== 顶部欢迎横幅 ========== -->
    <a-card :bordered="false" class="welcome-card">
      <div class="welcome-inner">
        <div class="welcome-left">
          <div class="welcome-greeting">{{ greeting }}{{ username }}</div>
          <div class="welcome-sub">
            <span>{{ siteName }}</span>
            <a-divider type="vertical" />
            <span>v{{ appVersion }}</span>
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
        <a-card :bordered="false" class="stat-card">
          <div class="stat-card-inner">
            <div class="stat-icon">
              <component :is="item.icon" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ item.value }}</div>
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
              <span class="cluster-dot" :style="{ background: item.color }"></span>
              <div class="cluster-info">
                <div class="cluster-value">{{ item.value }}</div>
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
                <component :is="item.icon" class="quick-link-icon" />
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
  ToolOutlined,
} from '@ant-design/icons-vue'
import { useUserStore } from '@/stores/user'
import { dashboardApi } from '@/api/dashboard'
import type { DashboardStats } from '@/api/dashboard'
import { formatDateTime } from '@/utils/format'

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
  { key: 'users',  label: '用户总数',  value: stats.userCount,     icon: UserOutlined,    path: '/system/users' },
  { key: 'roles',  label: '角色数量',  value: stats.roleCount,     icon: TeamOutlined,    path: '/system/roles' },
  { key: 'nodes',  label: '在线节点',  value: stats.onlineNodes,   icon: ClusterOutlined, path: '/system/nodes' },
  { key: 'menus',  label: '菜单总数',  value: stats.menuCount,     icon: MenuOutlined,    path: '/system/permissions' },
])

const clusterStatusList = computed(() => [
  { key: 'online',       label: '在线节点', value: stats.onlineNodes,      color: '#52c41a', path: '/system/nodes' },
  { key: 'offline',      label: '离线节点', value: stats.offlineNodes,     color: '#ff4d4f', path: '/system/nodes' },
  { key: 'maintenance',  label: '维护中',   value: stats.maintenanceNodes, color: '#faad14', path: '/system/nodes' },
])

// ── 快捷入口（路径全部修正，不再 404） ──
const quickLinks = [
  { label: '用户管理',  path: '/system/users',       icon: UserOutlined },
  { label: '角色管理',  path: '/system/roles',       icon: TeamOutlined },
  { label: '节点管理',  path: '/system/nodes',       icon: ClusterOutlined },
  { label: '资源监控',  path: '/system/resources',   icon: MonitorOutlined },
  { label: '配置中心',  path: '/system/config',      icon: SettingOutlined },
  { label: '权限管理',  path: '/system/permissions', icon: SafetyOutlined },
  { label: '定时任务',  path: '/system/scheduler',   icon: ToolOutlined },
]

// ── 登录日志 ──
const loginLogs = ref<DashboardStats['recent_logs']>([])
const logColumns = [
  { title: '用户名',   dataIndex: 'username', key: 'username', width: 120 },
  { title: 'IP 地址',  dataIndex: 'ip',       key: 'ip',       width: 150 },
  { title: '状态',     key: 'success',        width: 80 },
  { title: '消息',     dataIndex: 'message',  key: 'message',  ellipsis: true },
  { title: '时间',     key: 'created_at',     width: 180 },
]
const emptyLocale = { emptyText: '暂无登录记录' }

// ── 工具函数 ──
// 资源阈值配色：正常区间用主题色，接近上限才转告警色
function getPercentColor(p: number): string {
  if (p > 90) return '#cf1322'
  if (p > 70) return '#faad14'
  return 'var(--primary-color)'
}
function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}
// ── 数据加载（统一走 dashboard/stats/ 聚合接口）──
// inflight 标志：上轮请求未结束时跳过本轮，避免慢请求堆积造成并发覆盖与抖动
let statsInflight = false
async function loadStats() {
  if (statsInflight) return
  statsInflight = true
  try {
    const d = await dashboardApi.getStats()
    stats.userCount = d.user_count
    stats.roleCount = d.role_count
    stats.menuCount = d.menu_count
    stats.cpuUsage = d.cpu_usage
    stats.memoryUsage = d.memory_usage
    stats.memoryTotal = d.memory_total
    stats.memoryUsed = d.memory_used
    cpuCores.value = d.cpu_cores
    stats.onlineNodes = d.online_nodes
    stats.offlineNodes = d.offline_nodes
    stats.maintenanceNodes = d.maintenance_nodes
    loginLogs.value = d.recent_logs
  } catch {
    // 聚合接口失败时保持默认值
  } finally {
    statsInflight = false
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
/* 次要文字/弱底色/边框只在 .dashboard 上定义一次，暗色模式只覆盖这一处；
   正文颜色不写死，沿用 antd 主题 token 自动跟随暗色。 */
.dashboard {
  max-width: 1400px;
  margin: 0 auto;
  --dash-text-secondary: rgba(0, 0, 0, 0.45);
  --dash-surface: #f5f5f5;
  --dash-border: #f0f0f0;
  --dash-hover: rgba(0, 0, 0, 0.03);
}
:root.dark .dashboard {
  --dash-text-secondary: rgba(255, 255, 255, 0.45);
  --dash-surface: #262626;
  --dash-border: #303030;
  --dash-hover: rgba(255, 255, 255, 0.06);
}

.section {
  margin-top: 16px;
}

/* ── 欢迎横幅：纯排版，不用渐变/表情图标 ── */
.welcome-card {
  border-radius: 8px;
}
.welcome-card :deep(.ant-card-body) {
  padding: 20px 24px;
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
  gap: 4px;
}
.welcome-greeting {
  font-size: 18px;
  font-weight: 600;
}
.welcome-sub {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: var(--dash-text-secondary);
}
.welcome-right {
  text-align: right;
  flex-shrink: 0;
}
.current-time {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.3;
  /* 等宽数字：秒数跳动时整行不左右抖 */
  font-variant-numeric: tabular-nums;
}
.current-date {
  font-size: 12px;
  color: var(--dash-text-secondary);
  margin-top: 2px;
}

/* ── 统计卡片 ── */
.stat-card {
  border-radius: 8px;
}
.stat-card :deep(.ant-card-body) {
  padding: 18px 20px;
}
.stat-card-inner {
  display: flex;
  align-items: center;
  gap: 14px;
}
.stat-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
  background: var(--dash-surface);
  color: var(--dash-text-secondary);
}
.stat-info {
  flex: 1;
  min-width: 0;
}
.stat-value {
  font-size: 24px;
  font-weight: 600;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.stat-label {
  font-size: 13px;
  color: var(--dash-text-secondary);
  margin-top: 2px;
}

/* ── 区块卡片 ── */
.section-card {
  border-radius: 8px;
}
/* 不覆盖头部下边框：交给 antd 主题 token，否则暗色模式会留一条浅灰线 */
.section-card :deep(.ant-card-head) {
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
  align-items: baseline;
  margin-bottom: 8px;
}
.resource-title {
  font-size: 13px;
  color: var(--dash-text-secondary);
}
.resource-percent {
  font-size: 15px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.resource-footer {
  margin-top: 6px;
  font-size: 12px;
  color: var(--dash-text-secondary);
}

/* ── 集群状态：状态色只落在指示点上，数字与标签保持中性 ── */
.cluster-grid {
  display: flex;
  justify-content: space-around;
  gap: 12px;
}
.cluster-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}
.cluster-item:hover {
  background: var(--dash-hover);
}
.cluster-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.cluster-value {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.cluster-label {
  font-size: 12px;
  color: var(--dash-text-secondary);
  margin-top: 2px;
}

/* ── 快捷入口：中性描边方块，悬停时才用主题色 ── */
.quick-link-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 8px;
  border-radius: 8px;
  border: 1px solid var(--dash-border);
  cursor: pointer;
  transition: border-color 0.2s, color 0.2s;
}
.quick-link-card:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}
.quick-link-icon {
  font-size: 20px;
  color: var(--dash-text-secondary);
}
.quick-link-card:hover .quick-link-icon {
  color: var(--primary-color);
}
.quick-link-title {
  font-size: 13px;
}

/* ── 表格 ── */
.status-tag {
  border-radius: 4px;
  font-size: 12px;
  padding: 0 8px;
  line-height: 22px;
}
</style>
