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
            style="width: 120px"
          >
            <a-select-option value="create">创建</a-select-option>
            <a-select-option value="update">更新</a-select-option>
            <a-select-option value="delete">删除</a-select-option>
          </a-select>
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
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <a-tag :color="actionColor(record.action)">{{ actionLabel(record.action) }}</a-tag>
          </template>
        </template>
        <template #emptyText>
          <a-empty description="暂无审计日志" />
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { auditApi } from '@/api/audit'
import type { AuditLog } from '@/types'

const columns = [
  { title: '操作人', dataIndex: 'operator', key: 'operator', width: 120 },
  { title: '操作类型', dataIndex: 'action', key: 'action', width: 100 },
  { title: '模型', dataIndex: 'model_name', key: 'model_name', width: 150 },
  { title: '对象', dataIndex: 'object_repr', key: 'object_repr', width: 200 },
  { title: '变更摘要', dataIndex: 'diff_summary', key: 'diff_summary', ellipsis: true },
  { title: '操作时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
]

const actionLabel = (action: string) =>
  ({ create: '创建', update: '更新', delete: '删除' })[action] || action

const actionColor = (action: string) =>
  ({ create: 'green', update: 'blue', delete: 'red' })[action] || 'default'

const searchForm = ref({ search: '', action: undefined as string | undefined })
const tableData = ref<AuditLog[]>([])
const loading = ref(false)
const pagination = ref({ current: 1, pageSize: 20, total: 0 })

const fetchData = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.current,
      size: pagination.value.pageSize,
    }
    if (searchForm.value.search) params.search = searchForm.value.search

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
  searchForm.value = { search: '', action: undefined }
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
