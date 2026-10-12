<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <SearchBar>
        <SearchField label="操作类型">
          <a-select v-model:value="searchForm.action" placeholder="全部" allow-clear>
            <a-select-option value="create">创建</a-select-option>
            <a-select-option value="update">更新</a-select-option>
            <a-select-option value="delete">删除</a-select-option>
            <a-select-option value="execute">执行</a-select-option>
          </a-select>
        </SearchField>
        <SearchField label="模型">
          <a-input
            v-model:value="searchForm.model"
            placeholder="模型名称"
            allow-clear
            @pressEnter="handleSearch"
          />
        </SearchField>
        <SearchField label="起止">
          <a-range-picker
            v-model:value="dateRange"
            :placeholder="['开始日期', '结束日期']"
            @change="handleSearch"
          />
        </SearchField>

        <template #actions>
          <a-button type="primary" @click="handleSearch">
            <SearchOutlined /> 搜索
          </a-button>
          <a-button @click="resetSearch">
            <ReloadOutlined /> 重置
          </a-button>
        </template>
      </SearchBar>
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
import SearchBar from '@/components/SearchBar.vue'
import SearchField from '@/components/SearchField.vue'
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
  ({ create: '创建', update: '更新', delete: '删除', execute: '执行' })[action] || action

const actionColor = (action: string) =>
  ({ create: 'green', update: 'blue', delete: 'red', execute: 'orange' })[action] || 'default'

const searchForm = ref({ action: undefined as string | undefined, model: '' })
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
    if (searchForm.value.action) params.action = searchForm.value.action
    if (searchForm.value.model) params.model = searchForm.value.model
    // 时间范围必须是 RFC3339：后端 time.Parse(RFC3339) 失败会静默丢弃该条件
    if (dateRange.value?.[0]) params.start = dateRange.value[0].startOf('day').toISOString()
    if (dateRange.value?.[1]) params.end = dateRange.value[1].endOf('day').toISOString()

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
  searchForm.value = { action: undefined, model: '' }
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

<style scoped>
.search-card {
  margin-bottom: 16px;
}
</style>
