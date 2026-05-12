<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="服务名称">
          <a-input
            v-model:value="searchForm.search"
            placeholder="请输入服务名称"
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
            <PlusOutlined /> 新增服务
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
          <template v-if="column.key === 'type'">
            <a-tag :color="record.type === 'publish' ? 'blue' : 'green'">
              {{ record.type === 'publish' ? '发布' : '调用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'auth_type'">
            <a-tag>{{ authTypeText(record.auth_type) }}</a-tag>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)">
                <EditOutlined /> 编辑
              </a-button>
              <a-button type="link" size="small" @click="handleInvoke(record)">
                <PlayCircleOutlined /> 调用
              </a-button>
              <a-popconfirm title="确定要删除该服务吗？" @confirm="handleDelete(record)">
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
            <a-form-item label="服务名称" name="name">
              <a-input v-model:value="formState.name" placeholder="请输入服务名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="类型" name="type">
              <a-select v-model:value="formState.type" placeholder="请选择类型">
                <a-select-option value="publish">发布</a-select-option>
                <a-select-option value="consume">调用</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="WSDL 地址" name="wsdl_url">
              <a-input v-model:value="formState.wsdl_url" placeholder="http://example.com/service?wsdl" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="方法名" name="method">
              <a-input v-model:value="formState.method" placeholder="请输入方法名" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="请求模板" name="request_template">
          <a-textarea
            v-model:value="formState.request_template"
            placeholder='{"param1": "value1", "param2": "value2"}'
            :rows="4"
          />
        </a-form-item>

        <a-form-item label="响应映射" name="response_mapping">
          <a-textarea
            v-model:value="formState.response_mapping"
            placeholder="响应字段映射配置"
            :rows="3"
          />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="认证类型" name="auth_type">
              <a-select v-model:value="formState.auth_type" placeholder="请选择认证类型">
                <a-select-option value="none">无</a-select-option>
                <a-select-option value="basic">Basic</a-select-option>
                <a-select-option value="token">Token</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态" name="is_active">
              <a-switch v-model:checked="formState.is_active" checked-children="启用" un-checked-children="禁用" />
            </a-form-item>
          </a-col>
        </a-row>

        <template v-if="formState.auth_type !== 'none'">
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="认证用户名" name="auth_username">
                <a-input v-model:value="formState.auth_username" placeholder="请输入用户名" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="认证密码" name="auth_password">
                <a-input-password v-model:value="formState.auth_password" placeholder="请输入密码" />
              </a-form-item>
            </a-col>
          </a-row>
        </template>
      </a-form>
    </a-modal>

    <!-- 调用弹窗 -->
    <a-modal
      v-model:open="invokeModalVisible"
      title="调用 WebService"
      :confirm-loading="invokeLoading"
      @ok="handleInvokeOk"
      @cancel="invokeModalVisible = false"
      width="600px"
    >
      <a-form layout="vertical">
        <a-form-item label="请求参数 (JSON)">
          <a-textarea v-model:value="invokeParams" placeholder='{"key": "value"}' :rows="6" />
        </a-form-item>
      </a-form>
      <div v-if="invokeResult" style="margin-top: 16px;">
        <a-divider />
        <p><strong>执行结果：</strong></p>
        <p>耗时：{{ invokeResult.cost_ms }}ms</p>
        <pre style="background: #f5f5f5; padding: 12px; border-radius: 4px; overflow-x: auto;">{{ invokeResult.result }}</pre>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons-vue'
import { webserviceApi } from '@/api/webservice'
import type { WebService } from '@/types'

// 表格列定义
const columns = [
  { title: '服务名称', dataIndex: 'name', key: 'name' },
  { title: '类型', key: 'type', width: 100 },
  { title: 'WSDL 地址', dataIndex: 'wsdl_url', key: 'wsdl_url', ellipsis: true },
  { title: '方法名', dataIndex: 'method', key: 'method' },
  { title: '认证', key: 'auth_type', width: 100 },
  { title: '状态', key: 'is_active', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'action', width: 250 },
]

// 状态
const loading = ref(false)
const tableData = ref<WebService[]>([])
const pagination = reactive({
  current: 1, pageSize: 10, total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})
const searchForm = reactive({ search: '' })

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增服务')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  name: '',
  wsdl_url: '',
  type: 'consume' as 'publish' | 'consume',
  method: '',
  request_template: '',
  response_mapping: '',
  auth_type: 'none' as 'none' | 'basic' | 'token',
  auth_username: '',
  auth_password: '',
  is_active: true,
})

const formRules = {
  name: [{ required: true, message: '请输入服务名称' }],
  wsdl_url: [{ required: true, message: '请输入 WSDL 地址' }],
}

// 调用弹窗
const invokeModalVisible = ref(false)
const invokeLoading = ref(false)
const invokeParams = ref('{}')
const invokeResult = ref<{ result: string; cost_ms: number } | null>(null)
const currentService = ref<WebService | null>(null)

const authTypeText = (type: string) => {
  const map: Record<string, string> = { none: '无', basic: 'Basic', token: 'Token' }
  return map[type] || type
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await webserviceApi.getServices({
      page: pagination.current,
      search: searchForm.search,
    })
    tableData.value = res.results
    pagination.total = res.count
  } finally {
    loading.value = false
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
  modalTitle.value = '新增服务'
  currentId.value = ''
  Object.assign(formState, {
    name: '', wsdl_url: '', type: 'consume', method: '',
    request_template: '', response_mapping: '',
    auth_type: 'none', auth_username: '', auth_password: '', is_active: true,
  })
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: WebService) => {
  isEdit.value = true
  modalTitle.value = '编辑服务'
  currentId.value = record.id
  Object.assign(formState, {
    name: record.name,
    wsdl_url: record.wsdl_url,
    type: record.type,
    method: record.method,
    request_template: record.request_template,
    response_mapping: record.response_mapping,
    auth_type: record.auth_type,
    auth_username: record.auth_username,
    auth_password: '',
    is_active: record.is_active,
  })
  modalVisible.value = true
}

// 删除
const handleDelete = async (record: WebService) => {
  try {
    await webserviceApi.deleteService(record.id)
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
    if (!data.auth_password) delete data.auth_password
    if (isEdit.value) {
      await webserviceApi.updateService(currentId.value, data)
      message.success('更新成功')
    } else {
      await webserviceApi.createService(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadData()
  } catch (e) { console.error(e) }
  finally { modalLoading.value = false }
}

const handleModalCancel = () => { modalVisible.value = false; formRef.value?.resetFields() }

// 调用
const handleInvoke = (record: WebService) => {
  currentService.value = record
  invokeParams.value = record.request_template || '{}'
  invokeResult.value = null
  invokeModalVisible.value = true
}

const handleInvokeOk = async () => {
  if (!currentService.value) return
  invokeLoading.value = true
  try {
    const params = JSON.parse(invokeParams.value || '{}')
    const res = await webserviceApi.invokeService(currentService.value.id, params)
    invokeResult.value = res
    message.success('调用成功')
  } catch (e) {
    message.error('调用失败')
  } finally {
    invokeLoading.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.page-container { padding: 24px; }
.search-card { margin-bottom: 16px; }
.table-card { margin-bottom: 16px; }
.table-toolbar { margin-bottom: 16px; }
</style>