<template>
  <div class="page-container">
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
          <div class="resource-detail">核心数: {{ resources.cpu?.count || 0 }}</div>
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
          <div class="resource-detail">分区数: {{ resources.disk?.length || 0 }}</div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <a-statistic title="系统负载" :value="getLoadAvg()" :precision="2">
            <template #prefix>
              <LineChartOutlined />
            </template>
          </a-statistic>
          <div class="resource-detail">更新时间: {{ resources.timestamp ? formatTime(resources.timestamp) : '-' }}</div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card>
          <template #title>资源趋势</template>
          <template #extra>
            <a-space :size="12" wrap>
              <a-radio-group v-model:value="historyRange" button-style="solid" size="small">
                <a-radio-button v-for="option in historyRangeOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </a-radio-button>
              </a-radio-group>
              <span class="history-meta">聚合间隔: {{ historyIntervalLabel }}</span>
              <span class="history-meta">历史采样: 1 分钟</span>
            </a-space>
          </template>

          <a-row :gutter="[16, 16]">
            <a-col :xs="24" :xl="12">
              <a-card size="small" title="CPU 使用率趋势" :loading="historyLoading">
                <MetricTrendChart :labels="historyLabels" :series="cpuTrendSeries" unit="%" :max-value="100" />
              </a-card>
            </a-col>
            <a-col :xs="24" :xl="12">
              <a-card size="small" title="内存使用率趋势" :loading="historyLoading">
                <MetricTrendChart :labels="historyLabels" :series="memoryTrendSeries" unit="%" :max-value="100" />
              </a-card>
            </a-col>
            <a-col :xs="24" :xl="12">
              <a-card size="small" title="磁盘 IO 吞吐趋势" :loading="historyLoading">
                <MetricTrendChart :labels="historyLabels" :series="diskIoTrendSeries" unit=" MB/s" :precision="2" />
              </a-card>
            </a-col>
            <a-col :xs="24" :xl="12">
              <a-card size="small" title="磁盘使用率趋势" :loading="historyLoading">
                <MetricTrendChart :labels="historyLabels" :series="diskUsageTrendSeries" unit="%" :max-value="100" />
              </a-card>
            </a-col>
          </a-row>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card title="磁盘使用情况">
          <a-table :columns="diskColumns" :data-source="resources.disk || []" :pagination="false" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'percent'">
                <a-progress :percent="record.percent" :status="record.percent > 90 ? 'exception' : 'normal'" size="small" />
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

    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="网络连接统计">
          <a-descriptions bordered :column="2" size="small">
            <a-descriptions-item label="监听中"><a-tag color="blue">{{ netstat.LISTEN }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="已建立"><a-tag color="green">{{ netstat.ESTABLISHED }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="等待关闭"><a-tag color="orange">{{ netstat.TIME_WAIT }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="关闭等待"><a-tag color="red">{{ netstat.CLOSE_WAIT }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="其他"><a-tag>{{ netstat.OTHER }}</a-tag></a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="网络 IO">
          <a-descriptions bordered :column="2" size="small">
            <a-descriptions-item label="发送字节">{{ formatBytes(resources.network_io?.bytes_sent || 0) }}</a-descriptions-item>
            <a-descriptions-item label="接收字节">{{ formatBytes(resources.network_io?.bytes_recv || 0) }}</a-descriptions-item>
            <a-descriptions-item label="发送包">{{ formatNumber(resources.network_io?.packets_sent || 0) }}</a-descriptions-item>
            <a-descriptions-item label="接收包">{{ formatNumber(resources.network_io?.packets_recv || 0) }}</a-descriptions-item>
            <a-descriptions-item label="错误入">{{ resources.network_io?.errin || 0 }}</a-descriptions-item>
            <a-descriptions-item label="错误出">{{ resources.network_io?.errout || 0 }}</a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card>
          <div class="refresh-control">
            <a-space>
              <a-switch v-model:checked="autoRefresh" checked-children="自动刷新" un-checked-children="手动刷新" />
              <span v-if="autoRefresh">实时刷新: {{ refreshInterval }}秒，历史趋势: 60秒</span>
              <a-button type="primary" @click="handleManualRefresh" :loading="loading || historyLoading">
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
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import {
  DashboardOutlined,
  DatabaseOutlined,
  HddOutlined,
  LineChartOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue'
import MetricTrendChart from '@/components/monitor/MetricTrendChart.vue'
import { monitorApi } from '@/api/monitor'
import type { NetstatInfo, SystemResourceHistory, SystemResources } from '@/types'

type HistoryRange = '1h' | '6h' | '24h' | '7d'

const HISTORY_REFRESH_MS = 60 * 1000

const loading = ref(false)
const historyLoading = ref(false)
const resources = reactive<Partial<SystemResources>>({})
const netstat = reactive<NetstatInfo>({
  LISTEN: 0,
  ESTABLISHED: 0,
  TIME_WAIT: 0,
  CLOSE_WAIT: 0,
  OTHER: 0,
})
const resourceHistory = ref<SystemResourceHistory | null>(null)
const autoRefresh = ref(true)
const refreshInterval = ref(5)
const historyRange = ref<HistoryRange>('1h')
const lastHistoryLoadedAt = ref(0)
let timer: number | null = null

const historyRangeOptions: Array<{ label: string; value: HistoryRange }> = [
  { label: '近1小时', value: '1h' },
  { label: '近6小时', value: '6h' },
  { label: '近24小时', value: '24h' },
  { label: '近7天', value: '7d' },
]

const diskColumns = [
  { title: '设备', dataIndex: 'device', key: 'device' },
  { title: '挂载点', dataIndex: 'mountpoint', key: 'mountpoint' },
  { title: '文件系统', dataIndex: 'fstype', key: 'fstype' },
  { title: '总容量', key: 'total' },
  { title: '已使用', key: 'used' },
  { title: '可用', key: 'free' },
  { title: '使用率', key: 'percent' },
]

const historyPoints = computed(() => resourceHistory.value?.points || [])

const historyLabels = computed(() => {
  return historyPoints.value.map(point => formatHistoryLabel(point.timestamp, historyRange.value))
})

const historyIntervalLabel = computed(() => {
  const interval = resourceHistory.value?.interval || '1m'
  return ({ '1m': '1 分钟', '5m': '5 分钟', '15m': '15 分钟', '1h': '1 小时' } as Record<string, string>)[interval] || interval
})

const cpuTrendSeries = computed(() => [{
  name: 'CPU',
  color: '#1677ff',
  values: historyPoints.value.map(point => point.cpu_percent),
}])

const memoryTrendSeries = computed(() => [{
  name: '内存',
  color: '#52c41a',
  values: historyPoints.value.map(point => point.memory_percent),
}])

const diskIoTrendSeries = computed(() => [
  {
    name: '读取',
    color: '#722ed1',
    values: historyPoints.value.map(point => point.disk_read_mbps),
  },
  {
    name: '写入',
    color: '#fa8c16',
    values: historyPoints.value.map(point => point.disk_write_mbps),
  },
])

const diskUsageTrendSeries = computed(() => [{
  name: '磁盘',
  color: '#eb2f96',
  values: historyPoints.value.map(point => point.disk_percent),
}])

const loadRealtime = async () => {
  const [resourceRes, netstatRes] = await Promise.all([
    monitorApi.getSystemResources(),
    monitorApi.getNetstat(),
  ])

  Object.assign(resources, resourceRes)
  Object.assign(netstat, netstatRes)
}

const loadHistory = async (force = false) => {
  if (!force && Date.now() - lastHistoryLoadedAt.value < HISTORY_REFRESH_MS) {
    return
  }

  historyLoading.value = true
  try {
    resourceHistory.value = await monitorApi.getSystemResourceHistory({ range: historyRange.value })
    lastHistoryLoadedAt.value = Date.now()
  } catch {
    // 接口失败时也更新时间戳，避免 5s 自动刷新每轮都重试失败请求（错误退避）。
    // 全局 axios 拦截器已展示错误提示，这里不再重复弹窗。
    lastHistoryLoadedAt.value = Date.now()
  } finally {
    historyLoading.value = false
  }
}

const loadData = async (options?: { includeHistory?: boolean; forceHistory?: boolean }) => {
  loading.value = true
  try {
    await loadRealtime()
    if (options?.includeHistory) {
      await loadHistory(options.forceHistory)
    }
  } catch {
  } finally {
    loading.value = false
  }
}

const handleManualRefresh = () => loadData({ includeHistory: true, forceHistory: true })

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatNumber = (num: number): string => num.toLocaleString()

const formatTime = (timestamp: number): string => {
  if (!timestamp) return '-'
  return new Date(timestamp * 1000).toLocaleString()
}

const formatHistoryLabel = (timestamp: number, range: HistoryRange): string => {
  const date = new Date(timestamp * 1000)
  const timeText = `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  if (range === '7d') {
    return `${date.getMonth() + 1}-${date.getDate()} ${timeText}`
  }
  return timeText
}

const getDiskUsagePercent = (): number => {
  if (!resources.disk || resources.disk.length === 0) return 0
  const total = resources.disk.reduce((sum, item) => sum + item.total, 0)
  const used = resources.disk.reduce((sum, item) => sum + item.used, 0)
  return total > 0 ? Math.round((used / total) * 100) : 0
}

const getLoadAvg = (): number => {
  if (!resources.load_avg || resources.load_avg.length === 0) return 0
  return resources.load_avg[0] || 0
}

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

const startAutoRefresh = () => {
  if (timer) clearInterval(timer)
  timer = window.setInterval(() => {
    loadData({ includeHistory: true })
  }, refreshInterval.value * 1000)
}

const stopAutoRefresh = () => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

watch(autoRefresh, value => {
  if (value) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

watch(historyRange, async () => {
  lastHistoryLoadedAt.value = 0
  await loadHistory(true)
})

onMounted(async () => {
  await loadData({ includeHistory: true, forceHistory: true })
  if (autoRefresh.value) {
    startAutoRefresh()
  }
})

onUnmounted(() => {
  stopAutoRefresh()
})
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

.history-meta {
  color: #666;
  font-size: 12px;
}

.refresh-control {
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>
