<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="关键字">
          <a-input
            v-model:value="searchForm.search"
            placeholder="用户名/IP/消息"
            allow-clear
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item label="IP">
          <a-input
            v-model:value="searchForm.ip"
            placeholder="IP 地址"
            allow-clear
            @pressEnter="handleSearch"
            style="width: 140px"
          />
        </a-form-item>
        <a-form-item label="结果">
          <a-select
            v-model:value="searchForm.success"
            placeholder="全部"
            allow-clear
            style="width: 100px"
          >
            <a-select-option :value="true">成功</a-select-option>
            <a-select-option :value="false">失败</a-select-option>
          </a-select>
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
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'success'">
            <a-tag :color="record.success ? 'success' : 'error'">
              {{ record.success ? '成功' : '失败' }}
            </a-tag>
          </template>
        </template>
        <template #emptyText>
          <a-empty description="暂无登录日志" />
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { loginLogApi } from '@/api/auth'
import { formatDateTime } from '@/utils/format'
import type { LoginLog } from '@/types'

const columns = [
  { title: '用户名', dataIndex: 'username', key: 'username', width: 120 },
  { title: '结果', dataIndex: 'success', key: 'success', width: 80 },
  { title: 'IP 地址', dataIndex: 'ip', key: 'ip', width: 140 },
  { title: '消息', dataIndex: 'message', key: 'message', width: 200 },
  { title: 'UA', dataIndex: 'user_agent', key: 'user_agent', ellipsis: true },
  { title: '登录时间', dataIndex: 'created_at', key: 'created_at', width: 180, customRender: ({ text }: any) => formatDateTime(text) },
]

const searchForm = ref({ search: '', ip: '', success: undefined as boolean | undefined })
const dateRange = ref<[any, any] | null>(null)
const tableData = ref<LoginLog[]>([])
const loading = ref(false)
const pagination = ref({ current: 1, pageSize: 10, total: 0 })

const fetchData = async () => {
  loading.value = true
  try {
    const params: any = { page: pagination.value.current, size: pagination.value.pageSize }
    if (searchForm.value.search) params.search = searchForm.value.search
    if (searchForm.value.ip) params.ip = searchForm.value.ip
    if (searchForm.value.success !== undefined) params.success = searchForm.value.success
    if (dateRange.value && dateRange.value[0]) params.created_at__gte = dateRange.value[0].format('YYYY-MM-DD')
    if (dateRange.value && dateRange.value[1]) params.created_at__lte = dateRange.value[1].format('YYYY-MM-DD') + ' 23:59:59'

    const res = await loginLogApi.getLoginLogs(params)
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
  searchForm.value = { search: '', ip: '', success: undefined }
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
