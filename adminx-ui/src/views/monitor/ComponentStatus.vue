<template>
  <div>
    <!-- 状态概览 -->
    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card hoverable @click="activeKey = 'system'">
          <template #title>
            <a-space>
              <DashboardOutlined style="color: #1890ff" />
              <span>系统状态</span>
            </a-space>
          </template>
          <a-skeleton :loading="loading" active>
            <div class="component-status">
              <a-badge :status="health.status === 'ok' ? 'success' : 'error'" />
              <span>{{ health.status === 'ok' ? '运行正常' : '异常' }}</span>
            </div>
            <div class="component-meta">
              <div>CPU: {{ health.cpu_percent ?? '-' }}%</div>
              <div>内存: {{ formatBytes(health.memory?.used) }}/{{ formatBytes(health.memory?.total) }}</div>
            </div>
          </a-skeleton>
        </a-card>
      </a-col>

      <a-col :xs="24" :sm="12" :lg="6">
        <a-card hoverable @click="activeKey = 'cache'">
          <template #title>
            <a-space>
              <DatabaseOutlined style="color: #52c41a" />
              <span>缓存服务</span>
            </a-space>
          </template>
          <a-skeleton :loading="loading" active>
            <div class="component-status">
              <a-badge :status="cacheStats.backend === 'redis' ? 'success' : 'warning'" />
              <span>{{ cacheStats.backend === 'redis' ? 'Redis' : cacheStats.backend || '未知' }}</span>
            </div>
            <div class="component-meta" v-if="cacheStats.backend === 'redis'">
              <div>内存: {{ cacheStats.used_memory }}</div>
              <div>Key 数量: {{ cacheStats.keys }}</div>
            </div>
            <div class="component-meta" v-else>
              <div>{{ cacheStats.msg }}</div>
            </div>
          </a-skeleton>
        </a-card>
      </a-col>

      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <template #title>
            <a-space>
              <ClusterOutlined style="color: #722ed1" />
              <span>集群节点</span>
            </a-space>
          </template>
          <a-skeleton :loading="loading" active>
            <div class="component-status">
              <a-badge status="success" />
              <span>在线: {{ clusterStats.online }} / {{ clusterStats.total }}</span>
            </div>
            <div class="component-meta">
              <a-button type="link" size="small" @click="$router.push('/cluster/nodes')">管理节点</a-button>
            </div>
          </a-skeleton>
        </a-card>
      </a-col>

      <a-col :xs="24" :sm="12" :lg="6">
        <a-card>
          <template #title>
            <a-space>
              <ApiOutlined style="color: #fa8c16" />
              <span>WebService</span>
            </a-space>
          </template>
          <a-skeleton :loading="loading" active>
            <div class="component-status">
              <a-badge status="default" />
              <span>{{ wsCount }} 个服务</span>
            </div>
          </a-skeleton>
        </a-card>
      </a-col>
    </a-row>

    <!-- 详情区域 -->
    <a-card style="margin-top: 16px">
      <a-tabs v-model:activeKey="activeKey">
        <a-tab-pane key="system" tab="系统组件">
          <a-descriptions bordered :column="2">
            <a-descriptions-item label="操作系统">{{ osInfo }}</a-descriptions-item>
            <a-descriptions-item label="Python 版本">{{ pythonVersion }}</a-descriptions-item>
            <a-descriptions-item label="Django 版本">{{ djangoVersion }}</a-descriptions-item>
            <a-descriptions-item label="Node 状态" :span="2">
              <a-badge :status="health.status === 'ok' ? 'success' : 'error'" text="运行正常" />
            </a-descriptions-item>
          </a-descriptions>
        </a-tab-pane>

        <a-tab-pane key="cache" tab="缓存管理">
          <a-form layout="inline" style="margin-bottom: 16px">
            <a-form-item label="缓存后端">
              <a-tag :color="cacheStats.backend === 'redis' ? 'green' : 'orange'">
                {{ cacheStats.backend === 'redis' ? 'Redis' : cacheStats.backend || '未知' }}
              </a-tag>
            </a-form-item>
            <a-form-item v-if="cacheStats.backend === 'redis'">
              <template #label>
                <span>内存占用</span>
              </template>
              <span>{{ cacheStats.used_memory }}</span>
            </a-form-item>
            <a-form-item>
              <a-button type="primary" danger :loading="clearing" @click="handleClearCache">
                <DeleteOutlined /> 清理缓存
              </a-button>
            </a-form-item>
          </a-form>

          <a-descriptions bordered :column="2" v-if="cacheStats.backend === 'redis'">
            <a-descriptions-item label="已用内存">{{ cacheStats.used_memory }}</a-descriptions-item>
            <a-descriptions-item label="Key 数量">{{ cacheStats.keys }}</a-descriptions-item>
            <a-descriptions-item label="运行天数">{{ cacheStats.uptime_days }}</a-descriptions-item>
            <a-descriptions-item label="命中率">{{ cacheStats.hit_rate }}</a-descriptions-item>
          </a-descriptions>
          <a-alert v-else type="warning" message="当前使用本地内存缓存，不支持详细统计" show-icon />
        </a-tab-pane>

        <a-tab-pane key="database" tab="数据库">
          <a-empty description="数据库连接信息" />
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  DashboardOutlined,
  DatabaseOutlined,
  ClusterOutlined,
  ApiOutlined,
  DeleteOutlined,
} from '@ant-design/icons-vue'
import { commonApi } from '@/api/common'
import { clusterApi } from '@/api/cluster'
import { webserviceApi } from '@/api/monitor'

const loading = ref(true)
const clearing = ref(false)
const activeKey = ref('system')

const health = reactive<any>({})
const cacheStats = reactive<any>({})
const clusterStats = reactive({ total: 0, online: 0 })
const wsCount = ref(0)

const osInfo = navigator.platform || 'Unknown'
const pythonVersion = '3.13'
const djangoVersion = '5.x'

const loadData = async () => {
  loading.value = true
  try {
    const [h, c, cl, ws] = await Promise.all([
      commonApi.health(),
      commonApi.cacheStats(),
      clusterApi.getOverview(),
      webserviceApi.getServices({ page: 1 }),
    ])
    Object.assign(health, h)
    Object.assign(cacheStats, c)
    clusterStats.total = cl.total
    clusterStats.online = cl.online
    wsCount.value = ws.count
  } catch (e) {
    console.error('加载组件状态失败', e)
  } finally {
    loading.value = false
  }
}

const handleClearCache = async () => {
  clearing.value = true
  try {
    const res = await commonApi.cacheClear()
    message.success(`缓存已清理 (${res.cleared})`)
    await commonApi.cacheStats().then((c) => Object.assign(cacheStats, c))
  } catch (e) {
    message.error('清理缓存失败')
  } finally {
    clearing.value = false
  }
}

const formatBytes = (bytes?: number) => {
  if (!bytes) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

onMounted(loadData)
</script>

<style scoped>
.component-status {
  font-size: 16px;
  font-weight: 500;
  margin-bottom: 8px;
}

.component-meta {
  font-size: 12px;
  color: #666;
}
</style>