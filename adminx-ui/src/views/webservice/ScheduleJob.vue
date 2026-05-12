<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="任务名称">
          <a-input
            v-model:value="searchForm.search"
            placeholder="请输入任务名称"
            allow-clear
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="handleSearch">
            <SearchOutlined /> 搜索
          </a-button>
          <a-button style="margin-left: 8px" @click="resetSearch">
            <ReloadOutlined /> 重置
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 操作栏 -->
    <a-card class="table-card">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <a-button type="primary" @click="handleAdd">
            <PlusOutlined /> 新增任务
          </a-button>
          <a-button @click="loadSchedulerStatus">
            <InfoCircleOutlined /> 调度器状态
          </a-button>
          <a-button @click="handleReload">
            <ReloadOutlined /> 重载任务
          </a-button>
        </div>
      </div>

      <!-- 表格 -->
      <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'trigger_type'">
            <a-tag :color="triggerTypeColor(record.trigger_type)">
              {{ triggerTypeText(record.trigger_type) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'webservice'">
            <span>{{ record.webservice_name || '-' }}</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)">
                <EditOutlined /> 编辑
              </a-button>
              <a-button type="link" size="small" @click="handleRunOnce(record)">
                <PlayCircleOutlined /> 执行一次
              </a-button>
              <a-popconfirm title="确定要删除该任务吗？" @confirm="handleDelete(record)">
                <a-button type="link" danger size="small">
                  <DeleteOutlined /> 删除
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新增/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      :confirm-loading="modalLoading"
      @ok="handleModalOk"
      @cancel="handleModalCancel"
      width="700px"
    >
      <a-form ref="formRef" :model="formState" :rules="formRules" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="任务名称" name="name">
              <a-input v-model:value="formState.name" placeholder="请输入任务名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="处理函数" name="handler">
              <a-input v-model:value="formState.handler" placeholder="如: djangoadminx.webservice.tasks.sync_data" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="触发类型" name="trigger_type">
              <a-select v-model:value="formState.trigger_type" placeholder="请选择触发类型">
                <a-select-option value="cron">Cron</a-select-option>
                <a-select-option value="interval">间隔</a-select-option>
                <a-select-option value="date">指定时间</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="关联 WebService" name="webservice">
              <a-select
                v-model:value="formState.webservice"
                placeholder="请选择关联服务（可选）"
                allow-clear
                :options="webserviceOptions"
              />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="触发配置" name="trigger_config">
          <a-textarea
            v-model:value="formState.trigger_config"
            :placeholder="triggerConfigPlaceholder"
            :rows="3"
          />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="参数 (JSON 数组)" name="args">
              <a-textarea v-model:value="formState.args" placeholder='["arg1", "arg2"]' :rows="2" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="关键字参数 (JSON 对象)" name="kwargs">
              <a-textarea v-model:value="formState.kwargs" placeholder='{"key": "value"}' :rows="2" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="状态" name="is_active">
          <a-switch v-model:checked="formState.is_active" checked-children="启用" un-checked-children="禁用" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 调度器状态弹窗 -->
    <a-modal
      v-model:open="statusModalVisible"
      title="调度器状态"
      :footer="null"
      width="600px"
    >
      <div v-if="schedulerStatus">
        <p><strong>运行状态：</strong><a-tag :color="schedulerStatus.running ? 'success' : 'error'">{{ schedulerStatus.running ? '运行中' : '已停止' }}</a-tag></p>
        <p><strong>任务数量：</strong>{{ schedulerStatus.job_count }}</p>
        <a-table
          :columns="[{ title: '任务ID', dataIndex: 'id' }, { title: '名称', dataIndex: 'name' }, { title: '下次执行', dataIndex: 'next_run' }]"
          :data-source="schedulerStatus.jobs"
          size="small"
          :pagination="false"
        />
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons-vue'
import { scheduleJobApi } from '@/api/webservice'
import { webserviceApi } from '@/api/webservice'
import type { ScheduleJob, WebService } from '@/types'

// 表格列定义
const columns = [
  { title: '任务名称', dataIndex: 'name', key: 'name' },
  { title: '处理函数', dataIndex: 'handler', key: 'handler', ellipsis: true },
  { title: '触发类型', key: 'trigger_type', width: 100 },
  { title: '触发配置', dataIndex: 'trigger_config', key: 'trigger_config', ellipsis: true },
  { title: '关联服务', key: 'webservice' },
  { title: '状态', key: 'is_active', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'action', width: 250 },
]

// 状态
const loading = ref(false)
const tableData = ref<ScheduleJob[]>([])
const pagination = reactive({
  current: 1, pageSize: 10, total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})
const searchForm = reactive({ search: '' })

// WebService 选项
const webserviceOptions = ref<{ label: string; value: string }[]>([])

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增任务')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  name: '',
  handler: '',
  trigger_type: 'interval' as 'cron' | 'interval' | 'date',
  trigger_config: '',
  args: '',
  kwargs: '',
  webservice: undefined as string | undefined,
  is_active: true,
})

const formRules = {
  name: [{ required: true, message: '请输入任务名称' }],
  handler: [{ required: true, message: '请输入处理函数' }],
  trigger_type: [{ required: true, message: '请选择触发类型' }],
}

const triggerConfigPlaceholder = computed(() => {
  const map: Record<string, string> = {
    cron: '{"minute": "0", "hour": "*", "day": "*", "month": "*", "day_of_week": "*"}',
    interval: '{"minutes": 5}',
    date: '{"run_date": "2024-01-01 12:00:00"}',
  }
  return map[formState.trigger_type] || ''
})

const triggerTypeColor = (type: string) => {
  const map: Record<string, string> = { cron: 'purple', interval: 'blue', date: 'orange' }
  return map[type] || 'default'
}

const triggerTypeText = (type: string) => {
  const map: Record<string, string> = { cron: 'Cron', interval: '间隔', date: '指定时间' }
  return map[type] || type
}

// 调度器状态
const statusModalVisible = ref(false)
const schedulerStatus = ref<{ running: boolean; job_count: number; jobs: any[] } | null>(null)

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await scheduleJobApi.getJobs({
      page: pagination.current,
      search: searchForm.search,
    })
    tableData.value = res.results
    pagination.total = res.count
  } finally {
    loading.value = false
  }
}

// 加载 WebService 选项
const loadWebserviceOptions = async () => {
  try {
    const res = await webserviceApi.getServices({ page: 1 })
    webserviceOptions.value = res.results.map((ws: WebService) => ({
      label: ws.name,
      value: ws.id,
    }))
  } catch (e) {
    console.error('加载 WebService 失败', e)
  }
}

// 搜索
const handleSearch = () => { pagination.current = 1; loadData() }
const resetSearch = () => { searchForm.search = ''; handleSearch() }
const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadData()
}

// 新增
const handleAdd = () => {
  isEdit.value = false
  modalTitle.value = '新增任务'
  currentId.value = ''
  Object.assign(formState, {
    name: '', handler: '', trigger_type: 'interval',
    trigger_config: '', args: '', kwargs: '',
    webservice: undefined, is_active: true,
  })
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: ScheduleJob) => {
  isEdit.value = true
  modalTitle.value = '编辑任务'
  currentId.value = record.id
  Object.assign(formState, {
    name: record.name,
    handler: record.handler,
    trigger_type: record.trigger_type,
    trigger_config: record.trigger_config,
    args: record.args,
    kwargs: record.kwargs,
    webservice: record.webservice,
    is_active: record.is_active,
  })
  modalVisible.value = true
}

// 删除
const handleDelete = async (record: ScheduleJob) => {
  try {
    await scheduleJobApi.deleteJob(record.id)
    message.success('删除成功')
    loadData()
  } catch { message.error('删除失败') }
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true
    const data: any = { ...formState }
    if (!data.webservice) delete data.webservice
    if (isEdit.value) {
      await scheduleJobApi.updateJob(currentId.value, data)
      message.success('更新成功')
    } else {
      await scheduleJobApi.createJob(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadData()
  } catch (e) { console.error(e) }
  finally { modalLoading.value = false }
}

const handleModalCancel = () => { modalVisible.value = false; formRef.value?.resetFields() }

// 执行一次
const handleRunOnce = async (record: ScheduleJob) => {
  try {
    const res = await scheduleJobApi.runOnce(record.id)
    message.success(`执行成功: ${res.result}`)
  } catch { message.error('执行失败') }
}

// 调度器状态
const loadSchedulerStatus = async () => {
  try {
    const res = await scheduleJobApi.getStatus()
    schedulerStatus.value = res
    statusModalVisible.value = true
  } catch { message.error('获取状态失败') }
}

// 重载任务
const handleReload = async () => {
  try {
    await scheduleJobApi.reloadJobs()
    message.success('任务重载成功')
    loadData()
  } catch { message.error('重载失败') }
}

onMounted(() => {
  loadData()
  loadWebserviceOptions()
})
</script>

<style scoped>
.page-container { padding: 24px; }
.search-card { margin-bottom: 16px; }
.table-card { margin-bottom: 16px; }
.table-toolbar { margin-bottom: 16px; }
</style>