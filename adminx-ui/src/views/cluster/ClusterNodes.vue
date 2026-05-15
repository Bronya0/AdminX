<template>
  <div class="page-container">
    <!-- 概览卡片 -->
    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :sm="8">
        <a-card>
          <a-statistic
            title="总节点数"
            :value="overview.total"
            :value-style="{ color: '#1890ff' }"
          >
            <template #prefix>
              <ClusterOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="8">
        <a-card>
          <a-statistic
            title="在线节点"
            :value="overview.online"
            :value-style="{ color: '#3f8600' }"
          >
            <template #prefix>
              <CheckCircleOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="8">
        <a-card>
          <a-statistic
            title="离线节点"
            :value="overview.offline"
            :value-style="{ color: '#cf1322' }"
          >
            <template #prefix>
              <CloseCircleOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 搜索和操作栏 -->
    <a-card style="margin-top: 16px">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="节点名称">
          <a-input
            v-model:value="searchForm.search"
            placeholder="请输入节点名称"
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
          <a-button type="primary" style="margin-left: 8px" @click="handleAdd">
            <PlusOutlined /> 新增节点
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 节点表格 -->
    <a-card style="margin-top: 16px">
      <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-badge
              :status="getStatusType(record.status)"
              :text="getStatusText(record.status)"
            />
          </template>
          <template v-if="column.key === 'role'">
            <a-tag :color="record.role === 'master' ? 'red' : 'blue'">
              {{ record.role === 'master' ? '主节点' : '从节点' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'last_heartbeat'">
            {{ record.last_heartbeat ? formatTime(record.last_heartbeat) : '从未' }}
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)">
                <EditOutlined /> 编辑
              </a-button>
              <a-button
                type="link"
                size="small"
                :disabled="record.status !== 'online'"
                @click="handleHeartbeat(record)"
              >
                <HeartOutlined /> 心跳
              </a-button>
              <a-popconfirm
                title="确定要删除该节点吗？"
                @confirm="handleDelete(record)"
              >
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
      width="600px"
    >
      <a-form
        ref="formRef"
        :model="formState"
        :rules="formRules"
        layout="vertical"
      >
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="节点名称" name="name">
              <a-input v-model:value="formState.name" placeholder="请输入节点名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="主机地址" name="host">
              <a-input v-model:value="formState.host" placeholder="请输入主机地址" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="端口" name="port">
              <a-input-number
                v-model:value="formState.port"
                :min="1"
                :max="65535"
                style="width: 100%"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="角色" name="role">
              <a-select v-model:value="formState.role" placeholder="请选择角色">
                <a-select-option value="master">主节点</a-select-option>
                <a-select-option value="slave">从节点</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="状态" name="status">
              <a-select v-model:value="formState.status" placeholder="请选择状态">
                <a-select-option value="online">在线</a-select-option>
                <a-select-option value="offline">离线</a-select-option>
                <a-select-option value="maintenance">维护</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="版本" name="version">
              <a-input v-model:value="formState.version" placeholder="请输入版本号" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="启用" name="is_active">
          <a-switch
            v-model:checked="formState.is_active"
            checked-children="启用"
            un-checked-children="禁用"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  ClusterOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  HeartOutlined,
} from '@ant-design/icons-vue'
import { clusterApi } from '@/api/cluster'
import type { ClusterNode, ClusterOverview } from '@/types'

// 表格列定义
const columns = [
  { title: '节点名称', dataIndex: 'name', key: 'name' },
  { title: '主机地址', dataIndex: 'host', key: 'host' },
  { title: '端口', dataIndex: 'port', key: 'port', width: 100 },
  { title: '角色', key: 'role', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '版本', dataIndex: 'version', key: 'version', width: 120 },
  { title: '最后心跳', key: 'last_heartbeat', width: 180 },
  { title: '启用', key: 'is_active', width: 100 },
  { title: '操作', key: 'action', width: 200, fixed: 'right' },
]

// 状态
const loading = ref(false)
const tableData = ref<ClusterNode[]>([])
const overview = reactive<ClusterOverview>({
  total: 0,
  online: 0,
  offline: 0,
  nodes: [],
})
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})
const searchForm = reactive({
  search: '',
})

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增节点')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  name: '',
  host: '',
  port: 8000,
  role: 'slave' as 'master' | 'slave',
  status: 'offline' as 'online' | 'offline' | 'maintenance',
  version: '',
  is_active: true,
})

const formRules = {
  name: [{ required: true, message: '请输入节点名称' }],
  host: [{ required: true, message: '请输入主机地址' }],
  port: [{ required: true, message: '请输入端口' }],
}

// 获取状态类型
const getStatusType = (status: string) => {
  const types: Record<string, any> = {
    online: 'success',
    offline: 'error',
    maintenance: 'warning',
  }
  return types[status] || 'default'
}

// 获取状态文本
const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    online: '在线',
    offline: '离线',
    maintenance: '维护',
  }
  return texts[status] || status
}

// 格式化时间
const formatTime = (time: string) => {
  return new Date(time).toLocaleString()
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    // 加载概览
    const overviewRes = await clusterApi.getOverview()
    Object.assign(overview, overviewRes)

    // 加载列表
    const res = await clusterApi.getNodes({
      page: pagination.current,
      size: pagination.pageSize,
      search: searchForm.search,
    })
    tableData.value = res.results
    pagination.total = res.count
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.current = 1
  loadData()
}

// 重置搜索
const resetSearch = () => {
  searchForm.search = ''
  handleSearch()
}

// 表格变化
const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadData()
}

// 新增
const handleAdd = () => {
  isEdit.value = false
  modalTitle.value = '新增节点'
  currentId.value = ''
  Object.assign(formState, {
    name: '',
    host: '',
    port: 8000,
    role: 'slave',
    status: 'offline',
    version: '',
    is_active: true,
  })
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: ClusterNode) => {
  isEdit.value = true
  modalTitle.value = '编辑节点'
  currentId.value = record.id
  Object.assign(formState, {
    name: record.name,
    host: record.host,
    port: record.port,
    role: record.role,
    status: record.status,
    version: record.version,
    is_active: record.is_active,
  })
  modalVisible.value = true
}

// 删除
const handleDelete = async (record: ClusterNode) => {
  try {
    await clusterApi.deleteNode(record.id)
    message.success('删除成功')
    loadData()
  } catch (e) {
    message.error('删除失败')
  }
}

// 心跳
const handleHeartbeat = async (record: ClusterNode) => {
  message.info(`向节点 ${record.name} 发送心跳检测...`)
  // 实际实现需要后端支持心跳检测接口
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true

    if (isEdit.value) {
      await clusterApi.updateNode(currentId.value, formState)
      message.success('更新成功')
    } else {
      await clusterApi.createNode(formState)
      message.success('创建成功')
    }

    modalVisible.value = false
    loadData()
  } catch (e) {
    console.error(e)
  } finally {
    modalLoading.value = false
  }
}

// 弹窗取消
const handleModalCancel = () => {
  modalVisible.value = false
  formRef.value?.resetFields()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.page-container {
  padding: 24px;
}
</style>
