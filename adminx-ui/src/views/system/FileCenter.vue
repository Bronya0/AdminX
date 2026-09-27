<template>
  <div class="page-container">
    <a-card class="table-card">
      <div class="table-toolbar">
        <a-upload
          :show-upload-list="false"
          :custom-request="handleUpload"
          :disabled="uploading"
          :accept="acceptExts"
        >
          <a-button type="primary" :loading="uploading"><UploadOutlined /> 上传文件</a-button>
        </a-upload>
        <span class="upload-hint">
          单个文件最大 100MB，仅支持图片/文档/压缩包；非超级管理员只能看到并管理自己上传的文件
        </span>
      </div>

      <a-table
        :columns="columns"
        :data-source="records"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'size'">{{ formatSize(record.size) }}</template>
          <template v-else-if="column.key === 'created_at'">{{ formatDateTime(record.created_at) }}</template>
          <template v-else-if="column.key === 'action'">
            <a-button type="link" size="small" @click="handleDownload(record)">下载</a-button>
            <a-popconfirm title="确定删除该文件？" @confirm="handleDelete(record)">
              <a-button type="link" danger size="small">删除</a-button>
            </a-popconfirm>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { UploadOutlined } from '@ant-design/icons-vue'
import { fileApi } from '@/api/file'
import { formatDateTime } from '@/utils/format'
import type { FileRecord } from '@/types'

// 与后端 allowedUploadExt 白名单保持一致（后端才是权威校验，这里只是提前拦一次）
const acceptExts = '.jpg,.jpeg,.png,.gif,.webp,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.md,.csv,.zip,.rar,.7z,.tar,.gz'

const loading = ref(false)
const uploading = ref(false)
const records = ref<FileRecord[]>([])

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})

const columns = [
  { title: '文件名', dataIndex: 'original_name', key: 'original_name', ellipsis: true },
  { title: '大小', key: 'size', width: 110 },
  { title: '类型', dataIndex: 'mime_type', key: 'mime_type', width: 180, ellipsis: true },
  { title: '上传人', dataIndex: 'uploaded_by', key: 'uploaded_by', width: 220, ellipsis: true },
  { title: '上传时间', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 140 },
]

const fetchData = async () => {
  loading.value = true
  try {
    const res = await fileApi.list({ page: pagination.current, size: pagination.pageSize })
    records.value = res.results || []
    pagination.total = res.count || 0
  } catch {
    /* 错误提示已由请求拦截器统一处理 */
  } finally {
    loading.value = false
  }
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  fetchData()
}

const handleUpload = async (options: any) => {
  uploading.value = true
  try {
    await fileApi.upload(options.file)
    message.success('上传成功')
    options.onSuccess?.({})
    pagination.current = 1
    await fetchData()
  } catch (e) {
    options.onError?.(e)
  } finally {
    uploading.value = false
  }
}

const handleDownload = async (record: FileRecord) => {
  try {
    await fileApi.download(record)
  } catch {
    /* 拦截器已提示 */
  }
}

const handleDelete = async (record: FileRecord) => {
  try {
    await fileApi.remove(record.id)
    message.success('已删除')
    await fetchData()
  } catch {
    /* 拦截器已提示 */
  }
}

const formatSize = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index += 1
  }
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

onMounted(() => { fetchData() })
</script>

<style scoped>
.page-container { padding: 24px; }
.table-card { margin-bottom: 16px; }
.table-toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.upload-hint { color: #999; font-size: 12px; }
</style>
