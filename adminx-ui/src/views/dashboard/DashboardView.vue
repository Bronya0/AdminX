<template>
  <div>
    <!-- 统计卡片 -->
    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="总用户数"
            :value="stats.userCount"
            :value-style="{ color: '#3f8600' }"
          >
            <template #prefix>
              <UserOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="角色数量"
            :value="stats.roleCount"
            :value-style="{ color: '#1890ff' }"
          >
            <template #prefix>
              <TeamOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="在线节点"
            :value="stats.onlineNodes"
            :value-style="{ color: '#52c41a' }"
          >
            <template #prefix>
              <ClusterOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="菜单数量"
            :value="stats.menuCount"
            :value-style="{ color: '#722ed1' }"
          >
            <template #prefix>
              <MenuOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 系统资源概览 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="系统资源">
          <a-row :gutter="[16, 16]">
            <a-col :span="12">
              <div class="resource-item">
                <div class="resource-label">CPU 使用率</div>
                <a-progress
                  type="circle"
                  :percent="Math.round(stats.cpuUsage)"
                  :size="100"
                  :stroke-color="getCpuColor(stats.cpuUsage)"
                />
              </div>
            </a-col>
            <a-col :span="12">
              <div class="resource-item">
                <div class="resource-label">内存使用率</div>
                <a-progress
                  type="circle"
                  :percent="Math.round(stats.memoryUsage)"
                  :size="100"
                  :stroke-color="getMemoryColor(stats.memoryUsage)"
                />
              </div>
            </a-col>
          </a-row>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="集群状态">
          <div class="cluster-status">
            <div class="status-item">
              <div class="status-icon online">
                <CheckCircleOutlined />
              </div>
              <div class="status-info">
                <div class="status-value">{{ stats.onlineNodes }}</div>
                <div class="status-label">在线节点</div>
              </div>
            </div>
            <div class="status-item">
              <div class="status-icon offline">
                <CloseCircleOutlined />
              </div>
              <div class="status-info">
                <div class="status-value">{{ stats.offlineNodes }}</div>
                <div class="status-label">离线节点</div>
              </div>
            </div>
            <div class="status-item">
              <div class="status-icon maintenance">
                <ToolOutlined />
              </div>
              <div class="status-info">
                <div class="status-value">{{ stats.maintenanceNodes }}</div>
                <div class="status-label">维护中</div>
              </div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 快捷入口 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card title="快捷入口">
          <a-space :size="16" wrap>
            <a-button type="primary" size="large" @click="$router.push('/system/users')">
              <UserOutlined /> 用户管理
            </a-button>
            <a-button type="primary" size="large" @click="$router.push('/system/roles')">
              <TeamOutlined /> 角色管理
            </a-button>
            <a-button type="primary" size="large" @click="$router.push('/system/menus')">
              <MenuOutlined /> 菜单管理
            </a-button>
            <a-button size="large" @click="$router.push('/cluster/nodes')">
              <ClusterOutlined /> 节点管理
            </a-button>
            <a-button size="large" @click="$router.push('/monitor/resources')">
              <MonitorOutlined /> 资源监控
            </a-button>
            <a-button size="large" @click="$router.push('/config/list')">
              <SettingOutlined /> 配置中心
            </a-button>
          </a-space>
        </a-card>
      </a-col>
    </a-row>

    <!-- 最近登录日志 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card title="最近登录记录">
          <a-table
            :columns="logColumns"
            :data-source="loginLogs"
            :pagination="false"
            size="small"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'success'">
                <a-tag :color="record.success ? 'success' : 'error'">
                  {{ record.success ? '成功' : '失败' }}
                </a-tag>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import {
  UserOutlined,
  TeamOutlined,
  ClusterOutlined,
  MenuOutlined,
  MonitorOutlined,
  SettingOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ToolOutlined,
} from '@ant-design/icons-vue'
import { monitorApi } from '@/api/monitor'
import { clusterApi } from '@/api/cluster'
import { userApi, roleApi, loginLogApi } from '@/api/auth'
import { menuApi } from '@/api/menu'
import { formatDateTime } from '@/utils/format'
import type { LoginLog } from '@/types'

// 统计数据
const stats = reactive({
  userCount: 0,
  roleCount: 0,
  menuCount: 0,
  onlineNodes: 0,
  offlineNodes: 0,
  maintenanceNodes: 0,
  cpuUsage: 0,
  memoryUsage: 0,
})

// 登录日志
const loginLogs = ref<LoginLog[]>([])

const logColumns = [
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: 'IP 地址', dataIndex: 'ip', key: 'ip' },
  { title: '状态', key: 'success' },
  { title: '消息', dataIndex: 'message', key: 'message', ellipsis: true },
  { title: '时间', dataIndex: 'created_at', key: 'created_at', customRender: ({ text }: any) => formatDateTime(text) },
]

// 定时器
let timer: number | null = null

// 加载统计数据
const loadStats = async () => {
  try {
    // 获取系统资源
    const resources = await monitorApi.getSystemResources()
    stats.cpuUsage = resources.cpu.percent
    stats.memoryUsage = resources.memory.percent

    // 获取集群节点
    const clusterOverview = await clusterApi.getOverview()
    stats.onlineNodes = clusterOverview.online
    stats.offlineNodes = clusterOverview.offline
    stats.maintenanceNodes = clusterOverview.nodes.filter(n => n.status === 'maintenance').length

    // 获取用户数量
    const users = await userApi.getUsers({ page: 1 })
    stats.userCount = users.count

    // 获取角色数量
    const roles = await roleApi.getRoles({ page: 1 })
    stats.roleCount = roles.count

    // 获取菜单数量
    const menus = await menuApi.getMenus()
    stats.menuCount = menus.length

    // 获取登录日志
    const logs = await loginLogApi.getLoginLogs({ page: 1 })
    loginLogs.value = logs.results.slice(0, 5)
  } catch (e) {
    console.error('加载统计数据失败', e)
  }
}

// 获取颜色
const getCpuColor = (percent: number): string => {
  if (percent > 90) return '#cf1322'
  if (percent > 70) return '#faad14'
  return '#52c41a'
}

const getMemoryColor = (percent: number): string => {
  if (percent > 90) return '#cf1322'
  if (percent > 80) return '#faad14'
  return '#52c41a'
}

onMounted(() => {
  loadStats()
  // 每30秒刷新一次
  timer = window.setInterval(loadStats, 30000)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<style scoped>
.resource-item {
  text-align: center;
  padding: 16px;
}

.resource-label {
  margin-bottom: 16px;
  font-size: 14px;
  color: #666;
}

.cluster-status {
  display: flex;
  justify-content: space-around;
  padding: 24px 0;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.status-icon.online {
  background: #f6ffed;
  color: #52c41a;
}

.status-icon.offline {
  background: #fff2f0;
  color: #ff4d4f;
}

.status-icon.maintenance {
  background: #fff7e6;
  color: #faad14;
}

.status-info {
  text-align: center;
}

.status-value {
  font-size: 28px;
  font-weight: bold;
  color: #333;
}

.status-label {
  font-size: 14px;
  color: #666;
  margin-top: 4px;
}
</style>
