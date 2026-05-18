<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="任务名称">
          <a-input v-model:value="searchForm.search" placeholder="名称" allow-clear @pressEnter="handleSearch" />
        </a-form-item>
        <a-form-item label="类型">
          <a-select v-model:value="searchForm.command_type" placeholder="全部" allow-clear style="width: 120px">
            <a-select-option value="python">Python 函数</a-select-option>
            <a-select-option value="shell">Shell 命令</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="触发">
          <a-select v-model:value="searchForm.trigger_type" placeholder="全部" allow-clear style="width: 120px">
            <a-select-option value="cron">Cron</a-select-option>
            <a-select-option value="interval">间隔</a-select-option>
            <a-select-option value="date">指定时间</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="searchForm.is_active" placeholder="全部" allow-clear style="width: 100px">
            <a-select-option :value="true">启用</a-select-option>
            <a-select-option :value="false">禁用</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="handleSearch"><SearchOutlined /> 搜索</a-button>
          <a-button style="margin-left: 8px" @click="resetSearch"><ReloadOutlined /> 重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 操作栏 -->
    <a-card class="table-card">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <a-button type="primary" @click="handleAdd"><PlusOutlined /> 新建任务</a-button>
          <a-button @click="handleReload"><ReloadOutlined /> 重载调度器</a-button>
        </div>
        <div class="table-toolbar-right">
          <a-tag :color="schedulerRunning ? 'green' : 'red'">{{ schedulerRunning ? '调度器运行中' : '调度器未启动' }}</a-tag>
          <a-tag v-if="schedulerJobCount !== null">APS 任务: {{ schedulerJobCount }}</a-tag>
        </div>
      </div>

      <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
        :expand-row-keys="expandedRows"
        @expand="handleExpand"
      >
        <template #expandIconColumnTitle>
          <span>日志</span>
        </template>
        <template #expandedRowRender="{ record }">
          <div style="padding: 8px 0;">
            <a-table
              :data-source="logMap[record.id] || []"
              :columns="logColumns"
              :pagination="false"
              row-key="id"
              size="small"
            >
              <template #bodyCell="{ column, record: log }">
                <template v-if="column.key === 'status'">
                  <a-tag :color="({ running: 'processing', success: 'success', failed: 'error' } as Record<string, string>)[log.status]">
                    {{ ({ running: '运行中', success: '成功', failed: '失败' } as Record<string, string>)[log.status] }}
                  </a-tag>
                </template>
                <template v-if="column.key === 'duration'">
                  {{ log.started_at && log.finished_at ? formatDuration(log.started_at, log.finished_at) : '-' }}
                </template>
                <template v-if="column.key === 'result'">
                  <a-tooltip>
                    <template #title><pre style="max-height: 300px; overflow-y: auto; white-space: pre-wrap;">{{ log.result || '无输出' }}</pre></template>
                    <a>{{ (log.result || '').substring(0, 80) || '-' }}</a>
                  </a-tooltip>
                </template>
              </template>
            </a-table>
          </div>
        </template>

        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'command_type'">
            <a-tag :color="record.command_type === 'shell' ? 'purple' : 'blue'">{{ record.command_type === 'shell' ? 'Shell' : 'Python' }}</a-tag>
          </template>
          <template v-if="column.key === 'content'">
            <a-typography-paragraph
              :code="true"
              :ellipsis="{ rows: 1, expandable: false }"
              style="margin: 0; max-width: 300px;"
              :title="displayContent(record)"
              :content="displayContent(record)"
            />
          </template>
          <template v-if="column.key === 'trigger'">
            <span style="font-size: 12px;">{{ formatTrigger(record) }}</span>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-switch :checked="record.is_active" size="small" @change="(c: boolean) => handleToggleActive(record, c)" />
          </template>
          <template v-if="column.key === 'last_result'">
            <template v-if="record.last_run">
              <div style="display: flex; align-items: center; gap: 4px;">
                <a-tag :color="({ success: 'success', failed: 'error', running: 'processing' } as Record<string, string>)[record.last_run.status]" style="font-size: 12px;">
                  {{ ({ success: '成功', failed: '失败', running: '运行中' } as Record<string, string>)[record.last_run.status] || record.last_run.status }}
                </a-tag>
                <a-tooltip>
                  <template #title><pre style="max-height: 200px; overflow-y: auto; white-space: pre-wrap;">{{ record.last_run.result || '无输出' }}</pre></template>
                  <span style="font-size: 12px; color: #666; cursor: help; max-width: 200px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                    {{ (record.last_run.result || '').substring(0, 60) || '-' }}
                  </span>
                </a-tooltip>
              </div>
            </template>
            <span v-else style="color: #ccc;">-</span>
          </template>
          <template v-if="column.key === 'last_time'">
            <span style="font-size: 12px; color: #999;">{{ record.last_run ? formatDateTime(record.last_run.started_at) : '-' }}</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)"><EditOutlined /> 编辑</a-button>
              <a-button type="link" size="small" @click="handleRunOnce(record)"><PlayCircleOutlined /> 执行</a-button>
              <a-popconfirm title="确定删除？" @confirm="handleDelete(record)">
                <a-button type="link" danger size="small"><DeleteOutlined /> 删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新建/编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="modalTitle" @ok="handleModalOk" :confirm-loading="modalLoading" width="640px" destroy-on-close>
      <a-form ref="formRef" :model="formState" :rules="formRules" layout="vertical">
        <a-form-item label="任务名称" name="name">
          <a-input v-model:value="formState.name" placeholder="给任务起个名字" />
        </a-form-item>
        <a-form-item label="命令类型" name="command_type">
          <a-radio-group v-model:value="formState.command_type">
            <a-radio value="python">Python 函数</a-radio>
            <a-radio value="shell">Shell 命令</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="formState.command_type === 'python'" label="处理函数" name="handler">
          <a-input v-model:value="formState.handler" placeholder="如 djangoadminx.webservice.tasks.ntp_sync" />
        </a-form-item>
        <a-form-item v-if="formState.command_type === 'shell'" label="Shell 命令" name="command">
          <a-textarea v-model:value="formState.command" :rows="3" placeholder="要执行的命令或脚本路径" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="触发类型" name="trigger_type">
              <a-select v-model:value="formState.trigger_type">
                <a-select-option value="interval">间隔执行</a-select-option>
                <a-select-option value="cron">Cron 表达式</a-select-option>
                <a-select-option value="date">指定时间</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <!-- 间隔执行 -->
        <a-form-item v-if="formState.trigger_type === 'interval'" label="间隔">
          <div style="display: flex; gap: 8px; align-items: center; flex-wrap: wrap;">
            <a-input-number v-model:value="formInterval.days" :min="0" :max="365" style="width: 70px" /><span>天</span>
            <a-input-number v-model:value="formInterval.hours" :min="0" :max="23" style="width: 70px" /><span>时</span>
            <a-input-number v-model:value="formInterval.minutes" :min="0" :max="59" style="width: 70px" /><span>分</span>
            <a-input-number v-model:value="formInterval.seconds" :min="0" :max="59" style="width: 70px" /><span>秒</span>
            <a-select v-model:value="intervalPreset" :options="intervalOptions" placeholder="常用间隔" allow-clear style="width: 120px" @change="onIntervalPreset" />
          </div>
        </a-form-item>
        <!-- Cron 表达式 -->
        <a-form-item v-else-if="formState.trigger_type === 'cron'" label="Cron 表达式" name="trigger_config">
          <div style="display: flex; gap: 8px;">
            <a-input v-model:value="formState.trigger_config" placeholder="*/5 * * * *" style="flex:1" />
            <a-select v-model:value="formState.trigger_config" :options="cronOptions" placeholder="常用表达式" allow-clear style="width: 150px" />
          </div>
        </a-form-item>
        <!-- 指定时间 -->
        <a-form-item v-else label="执行时间" name="trigger_config">
          <a-date-picker v-model:value="formDate" show-time value-format="YYYY-MM-DD HH:mm:ss" style="width: 100%" />
        </a-form-item>
        <a-form-item label="状态">
          <a-switch v-model:checked="formState.is_active" checked-children="启用" un-checked-children="禁用" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import {
  SearchOutlined, ReloadOutlined, PlusOutlined,
  EditOutlined, DeleteOutlined, PlayCircleOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { scheduleJobApi, jobLogApi } from '@/api/webservice'
import { formatDateTime } from '@/utils/format'
import type { ScheduleJob, JobLog } from '@/types'

// ── 搜索 ──
const searchForm = ref({ search: '', command_type: undefined as string | undefined, trigger_type: undefined as string | undefined, is_active: undefined as boolean | undefined })

// ── 表格 ──
const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 140 },
  { title: '类型', dataIndex: 'command_type', key: 'command_type', width: 70 },
  { title: '内容', key: 'content', width: 220 },
  { title: '触发', dataIndex: 'trigger_type', key: 'trigger', width: 160 },
  { title: '状态', dataIndex: 'is_active', key: 'is_active', width: 65 },
  { title: '上次结果', key: 'last_result', width: 200 },
  { title: '执行时间', key: 'last_time', width: 150 },
  { title: '操作', key: 'action', width: 200, fixed: 'right' },
]

const logColumns = [
  { title: '状态', dataIndex: 'status', key: 'status', width: 65 },
  { title: '耗时', key: 'duration', width: 70 },
  { title: '结果', key: 'result' },
  { title: '时间', dataIndex: 'started_at', key: 'started_at', width: 155, customRender: ({ text }: any) => formatDateTime(text) },
]

const tableData = ref<ScheduleJob[]>([])
const loading = ref(false)
const pagination = ref({ current: 1, pageSize: 10, total: 0, showTotal: (total: number) => `共 ${total} 条` })
const schedulerRunning = ref(false)
const schedulerJobCount = ref<number | null>(null)

// 展开行日志
const expandedRows = ref<string[]>([])
const logMap = ref<Record<string, JobLog[]>>({})

// ── 弹窗 ──
const modalVisible = ref(false)
const modalLoading = ref(false)
const editingId = ref<string | null>(null)
const modalTitle = computed(() => editingId.value ? '编辑任务' : '新建任务')
const formState = ref({
  name: '', command_type: 'python' as 'python' | 'shell',
  handler: '', command: '',
  trigger_type: 'interval' as 'cron' | 'interval' | 'date', trigger_config: '{"hours": 1}', is_active: true,
})
const formRules: Record<string, any> = { name: [{ required: true, message: '请输入任务名称' }] }

// 间隔配置拆分为独立字段
const formInterval = reactive({ days: 0, hours: 1, minutes: 0, seconds: 0 })
const formDate = ref<string>('')

// 常用间隔选项
const intervalOptions = [
  { label: '每 5 分钟', value: '5' },
  { label: '每 10 分钟', value: '10' },
  { label: '每 30 分钟', value: '30' },
  { label: '每 1 小时', value: '60' },
  { label: '每 2 小时', value: '120' },
  { label: '每 6 小时', value: '360' },
  { label: '每 1 天', value: '1440' },
]
const intervalPreset = ref<string>('60')
const onIntervalPreset = (val: string) => {
  const total = Number(val)
  if (!total) return
  formInterval.days = Math.floor(total / 1440)
  formInterval.hours = Math.floor((total % 1440) / 60)
  formInterval.minutes = total % 60
  formInterval.seconds = 0
}

// 常用 Cron 表达式
const cronOptions = [
  { label: '每 5 分钟', value: '*/5 * * * *' },
  { label: '每 10 分钟', value: '*/10 * * * *' },
  { label: '每 30 分钟', value: '*/30 * * * *' },
  { label: '每小时', value: '0 * * * *' },
  { label: '每天 2:00', value: '0 2 * * *' },
  { label: '每天 12:00', value: '0 12 * * *' },
  { label: '每周一 0:00', value: '0 0 * * 1' },
  { label: '每月 1 号 0:00', value: '0 0 1 * *' },
]

// 间隔字段 → JSON
const intervalToConfig = () => JSON.stringify({
  ...(formInterval.days ? { days: formInterval.days } : {}),
  ...(formInterval.hours ? { hours: formInterval.hours } : {}),
  ...(formInterval.minutes ? { minutes: formInterval.minutes } : {}),
  ...(formInterval.seconds ? { seconds: formInterval.seconds } : {}),
})
// JSON → 间隔字段
const configToInterval = (json: string) => {
  try {
    const c = JSON.parse(json)
    formInterval.days = c.days ?? 0
    formInterval.hours = c.hours ?? 0
    formInterval.minutes = c.minutes ?? 0
    formInterval.seconds = c.seconds ?? 0
  } catch {
    formInterval.days = formInterval.hours = formInterval.minutes = formInterval.seconds = 0
  }
}

// ── 取数据 ──
const fetchData = async () => {
  loading.value = true
  try {
    const params: any = { page: pagination.value.current, size: pagination.value.pageSize }
    if (searchForm.value.search) params.search = searchForm.value.search
    if (searchForm.value.command_type) params.command_type = searchForm.value.command_type
    if (searchForm.value.trigger_type) params.trigger_type = searchForm.value.trigger_type
    if (searchForm.value.is_active !== undefined) params.is_active = searchForm.value.is_active
    const res = await scheduleJobApi.getJobs(params)
    tableData.value = res.results
    pagination.value.total = res.count
  } finally { loading.value = false }
}

const fetchSchedulerStatus = async () => {
  try {
    const s = await scheduleJobApi.getStatus()
    schedulerRunning.value = s.running
    schedulerJobCount.value = s.job_count
  } catch { /* ignore */ }
}

// ── 展开日志 ──
const handleExpand = async (expanded: boolean, record: ScheduleJob) => {
  if (expanded) {
    try {
      const res = await jobLogApi.list({ page: 1, size: 10, job: record.id })
      logMap.value[record.id] = res.results
    } catch { logMap.value[record.id] = [] }
  }
}

// ── 操作 ──
const handleSearch = () => { pagination.value.current = 1; fetchData() }
const resetSearch = () => { searchForm.value = { search: '', command_type: undefined, trigger_type: undefined, is_active: undefined }; pagination.value.current = 1; fetchData() }
const handleTableChange = (pag: any) => { pagination.value.current = pag.current; pagination.value.pageSize = pag.pageSize; fetchData() }

const displayContent = (job: ScheduleJob) => job.command_type === 'shell' ? (job.command || '-') : (job.handler || '-')

const handleAdd = () => {
  editingId.value = null
  formState.value = { name: '', command_type: 'python', handler: '', command: '', trigger_type: 'interval', trigger_config: '{"hours": 1}', is_active: true }
  formInterval.days = 0; formInterval.hours = 1; formInterval.minutes = 0; formInterval.seconds = 0
  formDate.value = ''
  modalVisible.value = true
}

const handleEdit = (job: ScheduleJob) => {
  editingId.value = job.id
  formState.value = { name: job.name, command_type: job.command_type, handler: job.handler || '', command: job.command || '', trigger_type: job.trigger_type, trigger_config: job.trigger_config, is_active: job.is_active }
  formDate.value = ''
  if (job.trigger_type === 'interval') {
    configToInterval(job.trigger_config)
  }
  if (job.trigger_type === 'date') {
    try {
      const c = JSON.parse(job.trigger_config)
      formDate.value = c.run_date || ''
    } catch { formDate.value = '' }
  }
  modalVisible.value = true
}

const handleModalOk = async () => {
  modalLoading.value = true
  try {
    // 序列化触发配置
    const data = { ...formState.value }
    if (data.trigger_type === 'interval') {
      data.trigger_config = intervalToConfig()
    } else if (data.trigger_type === 'date' && formDate.value) {
      data.trigger_config = JSON.stringify({ run_date: formDate.value })
    }
    if (editingId.value) {
      await scheduleJobApi.update(editingId.value, data)
      message.success('任务已更新')
    } else {
      await scheduleJobApi.create(data)
      message.success('任务已创建')
    }
    modalVisible.value = false
    fetchData()
  } finally { modalLoading.value = false }
}

const handleDelete = async (job: ScheduleJob) => {
  try {
    await scheduleJobApi.remove(job.id)
    message.success('已删除')
    fetchData()
  } catch { /* interceptor handles error */ }
}

const handleRunOnce = async (job: ScheduleJob) => {
  try {
    const res = await scheduleJobApi.runOnce(job.id)
    const result = (res?.result || '').substring(0, 100)
    message.success(`执行完成: ${result || '无输出'}`)
    fetchData()
    fetchSchedulerStatus()
  } catch {
    fetchData()
  }
}

const handleToggleActive = async (job: ScheduleJob, checked: boolean) => {
  try {
    await scheduleJobApi.update(job.id, { is_active: checked } as any)
    job.is_active = checked
    message.success(checked ? '任务已启用' : '任务已禁用')
  } catch { /* revert on failure */ }
}

const handleReload = async () => {
  try {
    await scheduleJobApi.reload()
    message.success('调度器已重载')
    fetchSchedulerStatus()
  } catch { /* interceptor handles error */ }
}

// ── 格式化 ──
const formatTrigger = (job: ScheduleJob) => {
  if (job.trigger_type === 'interval') {
    try {
      const c = JSON.parse(job.trigger_config)
      const p: string[] = []
      if (c.days) p.push(`${c.days}天`)
      if (c.hours) p.push(`${c.hours}小时`)
      if (c.minutes) p.push(`${c.minutes}分钟`)
      if (c.seconds) p.push(`${c.seconds}秒`)
      return `每 ${p.join(' ')}`
    } catch { return job.trigger_config }
  }
  if (job.trigger_type === 'cron') return job.trigger_config
  return job.trigger_config
}

const formatDuration = (start: string, end: string) => {
  const ms = new Date(end).getTime() - new Date(start).getTime()
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`
}

onMounted(() => { fetchData(); fetchSchedulerStatus() })
</script>
