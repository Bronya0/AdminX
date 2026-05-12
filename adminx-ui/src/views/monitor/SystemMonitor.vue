<template>
  <div class="page-container">
    <!-- 资源概览卡片 -->
    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="CPU 使用率"
            :value="resources.cpu?.percent || 0"
            suffix="%"
            :precision="1"
            :value-style="{ color: getCpuColor(resources.cpu?.percent || 0) }"
          >
            <template #prefix>
              <DashboardOutlined />
            </template>
          </a-statistic>
          <div class="resource-detail">
            核心数: {{ resources.cpu?.count || 0 }}
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="内存使用率"
            :value="resources.memory?.percent || 0"
            suffix="%"
            :precision="1"
            :value-style="{ color: getMemoryColor(resources.memory?.percent || 0) }"
          >
            <template #prefix>
              <DatabaseOutlined />
            </template>
          </a-statistic>
          <div class="resource-detail">
            已用: {{ formatBytes(resources.memory?.used || 0) }} / {{ formatBytes(resources.memory?.total || 0) }}
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="磁盘使用率"
            :value="getDiskUsagePercent()"
            suffix="%"
            :precision="1"
            :value-style="{ color: getDiskColor(getDiskUsagePercent()) }"
          >
            <template #prefix>
              <HddOutlined />
            </template>
          </a-statistic>
          <div class="resource-detail">
            分区数: {{ resources.disk?.length || 0 }}
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic
            title="系统负载"
            :value="getLoadAvg()"
            :precision="2"
          >
            <template #prefix>
              <LineChartOutlined />
            </template>
          </a-statistic>
          <div class="resource-detail">
            更新时间: {{ resources.timestamp ? formatTime(resources.timestamp) : '-' }}
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 磁盘详情 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card title="磁盘使用情况">
          <a-table
            :columns="diskColumns"
            :data-source="resources.disk || []"
            :pagination="false"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'percent'">
                <a-progress
                  :percent="record.percent"
                  :status="record.percent > 90 ? 'exception' : 'normal'"
                  size="small"
                />
              </template>
              <template v-if="column.key === 'used'">
                {{ formatBytes(record.used) }}
              </template>
              <template v-if="column.key === 'total'">
                {{ formatBytes(record.total) }}
              </template>
              <template v-if="column.key === 'free'">
                {{ formatBytes(record.free) }}
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>

    <!-- 网络连接 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="网络连接统计">
          <a-descriptions bordered :column="2" size="small">
            <a-descriptions-item label="监听中">
              <a-tag color="blue">{{ netstat.LISTEN }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="已建立">
              <a-tag color="green">{{ netstat.ESTABLISHED }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="等待关闭">
              <a-tag color="orange">{{ netstat.TIME_WAIT }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="关闭等待">
              <a-tag color="red">{{ netstat.CLOSE_WAIT }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="其他">
              <a-tag>{{ netstat.OTHER }}</a-tag>
            </a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="网络 IO">
          <a-descriptions bordered :column="2" size="small">
            <a-descriptions-item label="发送字节">
              {{ formatBytes(resources.network_io?.bytes_sent || 0) }}
            </a-descriptions-item>
            <a-descriptions-item label="接收字节">
              {{ formatBytes(resources.network_io?.bytes_recv || 0) }}
            </a-descriptions-item>
            <a-descriptions-item label="发送包">
              {{ formatNumber(resources.network_io?.packets_sent || 0) }}
            </a-descriptions-item>
            <a-descriptions-item label="接收包">
              {{ formatNumber(resources.network_io?.packets_recv || 0) }}
            </a-descriptions-item>
            <a-descriptions-item label="错误入">
              {{ resources.network_io?.errin || 0 }}
            </a-descriptions-item>
            <a-descriptions-item label="错误出">
              {{ resources.network_io?.errout || 0 }}
            </a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>
    </a-row>

    <!-- 自动刷新 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card>
          <div class="refresh-control">
            <a-space>
              <a-switch
                v-model:checked="autoRefresh"
                checked-children="自动刷新"
                un-checked-children="手动刷新"
              />
              <span v-if="autoRefresh">刷新间隔: {{ refreshInterval }}秒</span>
              <a-button type="primary" @click="loadData" :loading="loading">
                <ReloadOutlined /> 立即刷新
              </a-button>
            </a-space>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  DashboardOutlined,
  DatabaseOutlined,
  HddOutlined,
  LineChartOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue'
import { monitorApi } from '@/api/monitor'
import type { SystemResources, NetstatInfo } from '@/types'

// 状态
const loading = ref(false)
const resources = reactive<Partial<SystemResources>>({})
const netstat = reactive<NetstatInfo>({
  LISTEN: 0,
  ESTABLISHED: 0,
  TIME_WAIT: 0,
  CLOSE_WAIT: 0,
  OTHER: 0,
})
const autoRefresh = ref(true)
const refreshInterval = ref(5)
let timer: number | null = null

// 磁盘表格列
const diskColumns = [
  { title: '设备', dataIndex: 'device', key: 'device' },
  { title: '挂载点', dataIndex: 'mountpoint', key: 'mountpoint' },
  { title: '文件系统', dataIndex: 'fstype', key: 'fstype' },
  { title: '总容量', key: 'total' },
  { title: '已使用', key: 'used' },
  { title: '可用', key: 'free' },
  { title: '使用率', key: 'percent' },
]

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await monitorApi.getSystemResources()
    Object.assign(resources, res)

    const netstatRes = await monitorApi.getNetstat()
    Object.assign(netstat, netstatRes)
  } catch (e) {
    message.error('加载监控数据失败')
  } finally {
    loading.value = false
  }
}

// 格式化字节
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 格式化数字
const formatNumber = (num: number): string => {
  return num.toLocaleString()
}

// 格式化时间
const formatTime = (timestamp: number): string => {
  if (!timestamp) return '-'
  return new Date(timestamp * 1000).toLocaleString()
}

// 获取磁盘使用率
const getDiskUsagePercent = (): number => {
  if (!resources.disk || resources.disk.length === 0) return 0
  const total = resources.disk.reduce((sum, d) => sum + d.total, 0)
  const used = resources.disk.reduce((sum, d) => sum + d.used, 0)
  return total > 0 ? Math.round((used / total) * 100) : 0
}

// 获取负载
const getLoadAvg = (): number => {
  if (!resources.load_avg || resources.load_avg.length === 0) return 0
  return resources.load_avg[0] || 0
}

// 获取颜色
const getCpuColor = (percent: number): string => {
  if (percent > 90) return '#cf1322'
  if (percent > 70) return '#faad14'
  return '#3f8600'
}

const getMemoryColor = (percent: number): string => {
  if (percent > 90) return '#cf1322'
  if (percent > 80) return '#faad14'
  return '#3f8600'
}

const getDiskColor = (percent: number): string => {
  if (percent > 90) return '#cf1322'
  if (percent > 80) return '#faad14'
  return '#3f8600'
}

// 启动自动刷新
const startAutoRefresh = () => {
  if (timer) clearInterval(timer)
  timer = window.setInterval(() => {
    loadData()
  }, refreshInterval.value * 1000)
}

// 停止自动刷新
const stopAutoRefresh = () => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

// 监听自动刷新
watch(autoRefresh, (val) => {
  if (val) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

onMounted(() => {
  loadData()
  if (autoRefresh.value) {
    startAutoRefresh()
  }
})

onUnmounted(() => {
  stopAutoRefresh()
})

import { watch } from 'vue'
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.resource-detail {
  margin-top: 8px;
  color: #666;
  font-size: 12px;
}

.refresh-control {
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>
