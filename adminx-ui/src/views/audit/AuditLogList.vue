<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="关键字">
          <a-input
            v-model:value="searchForm.search"
            placeholder="操作人/对象/摘要"
            allow-clear
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item label="操作类型">
          <a-select
            v-model:value="searchForm.action"
            placeholder="全部"
            allow-clear
            style="width: 100px"
          >
            <a-select-option value="create">创建</a-select-option>
            <a-select-option value="update">更新</a-select-option>
            <a-select-option value="delete">删除</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="模型">
          <a-input
            v-model:value="searchForm.model_name"
            placeholder="模型名称"
            allow-clear
            @pressEnter="handleSearch"
            style="width: 140px"
          />
        </a-form-item>
        <a-form-item label="起止">
          <a-range-picker
            v-model:value="dateRange"
            :placeholder="['开始日期', '结束日期']"
            @change="handleSearch"
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

    <!-- 表格 -->
    <a-card class="table-card">
      <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
        :locale="{ emptyText: '暂无审计日志' }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <a-tag :color="actionColor(record.action)">{{ actionLabel(record.action) }}</a-tag>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { auditApi } from '@/api/audit'
import { formatDateTime } from '@/utils/format'
import type { AuditLog } from '@/types'

const columns = [
  { title: '操作人', dataIndex: 'operator', key: 'operator', width: 120 },
  { title: '操作类型', dataIndex: 'action', key: 'action', width: 100 },
  { title: '模型', dataIndex: 'model_name', key: 'model_name', width: 150 },
  { title: '对象', dataIndex: 'object_repr', key: 'object_repr', width: 200 },
  { title: '变更摘要', dataIndex: 'diff_summary', key: 'diff_summary', ellipsis: true },
  { title: '操作时间', dataIndex: 'created_at', key: 'created_at', width: 180, customRender: ({ text }: any) => formatDateTime(text) },
]

const actionLabel = (action: string) =>
  ({ create: '创建', update: '更新', delete: '删除' })[action] || action

const actionColor = (action: string) =>
  ({ create: 'green', update: 'blue', delete: 'red' })[action] || 'default'

const searchForm = ref({ search: '', action: undefined as string | undefined, model_name: '' })
const dateRange = ref<[any, any] | null>(null)
const tableData = ref<AuditLog[]>([])
const loading = ref(false)
const pagination = ref({ current: 1, pageSize: 10, total: 0 })

const fetchData = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.current,
      size: pagination.value.pageSize,
    }
    if (searchForm.value.search) params.search = searchForm.value.search
    if (searchForm.value.action) params.action = searchForm.value.action
    if (searchForm.value.model_name) params.model_name = searchForm.value.model_name
    if (dateRange.value && dateRange.value[0]) params.created_at__gte = dateRange.value[0].format('YYYY-MM-DD')
    if (dateRange.value && dateRange.value[1]) params.created_at__lte = dateRange.value[1].format('YYYY-MM-DD') + ' 23:59:59'

    const res = await auditApi.getAuditLogs(params)
    tableData.value = res.results
    pagination.value.total = res.count
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.value.current = 1
  fetchData()
}

const resetSearch = () => {
  searchForm.value = { search: '', action: undefined, model_name: '' }
  dateRange.value = null
  pagination.value.current = 1
  fetchData()
}

const handleTableChange = (pag: any) => {
  pagination.value.current = pag.current
  pagination.value.pageSize = pag.pageSize
  fetchData()
}

onMounted(fetchData)
</script>
