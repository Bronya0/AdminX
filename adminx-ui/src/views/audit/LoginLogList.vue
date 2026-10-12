<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <SearchBar>
        <SearchField label="用户名">
          <a-input
            v-model:value="searchForm.username"
            placeholder="登录用户名"
            allow-clear
            @pressEnter="handleSearch"
          />
        </SearchField>
        <SearchField label="IP">
          <a-input
            v-model:value="searchForm.ip"
            placeholder="IP 地址"
            allow-clear
            @pressEnter="handleSearch"
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
        :locale="{ emptyText: '暂无登录日志' }"
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { loginLogApi } from '@/api/auth'
import { formatDateTime } from '@/utils/format'
import SearchBar from '@/components/SearchBar.vue'
import SearchField from '@/components/SearchField.vue'
import type { LoginLog } from '@/types'

const columns = [
  { title: '用户名', dataIndex: 'username', key: 'username', width: 120 },
  { title: '结果', dataIndex: 'success', key: 'success', width: 80 },
  { title: 'IP 地址', dataIndex: 'ip', key: 'ip', width: 140 },
  { title: '消息', dataIndex: 'message', key: 'message', width: 200 },
  { title: 'UA', dataIndex: 'user_agent', key: 'user_agent', ellipsis: true },
  { title: '登录时间', dataIndex: 'created_at', key: 'created_at', width: 180, customRender: ({ text }: any) => formatDateTime(text) },
]

const searchForm = ref({ username: '', ip: '' })
const tableData = ref<LoginLog[]>([])
const loading = ref(false)
const pagination = ref({ current: 1, pageSize: 10, total: 0 })

const fetchData = async () => {
  loading.value = true
  try {
    const params: any = { page: pagination.value.current, size: pagination.value.pageSize }
    if (searchForm.value.username) params.username = searchForm.value.username
    if (searchForm.value.ip) params.ip = searchForm.value.ip

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
  searchForm.value = { username: '', ip: '' }
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
