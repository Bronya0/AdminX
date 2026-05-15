<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="配置键">
          <a-input
            v-model:value="searchForm.search"
            placeholder="请输入配置键"
            allow-clear
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item label="分组">
          <a-select
            v-model:value="searchForm.group"
            placeholder="请选择分组"
            allow-clear
            style="width: 150px"
          >
            <a-select-option v-for="g in groupOptions" :key="g" :value="g">{{ g }}</a-select-option>
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

    <!-- 操作栏 -->
    <a-card class="table-card">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <a-button type="primary" @click="handleAdd">
            <PlusOutlined /> 新增配置
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
          <template v-if="column.key === 'value_type'">
            <a-tag :color="getTypeColor(record.value_type)">
              {{ getTypeText(record.value_type) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'display_value'">
            <span v-if="record.value_type === 'encrypted'">
              <a-tag color="orange">********</a-tag>
            </span>
            <span v-else-if="record.value_type === 'bool'">
              <a-tag :color="record.value === 'true' ? 'success' : 'error'">
                {{ record.value === 'true' ? '是' : '否' }}
              </a-tag>
            </span>
            <span v-else-if="record.value_type === 'json' || record.value_type === 'options'">
              <a-tag color="blue">JSON</a-tag>
            </span>
            <span v-else>{{ record.display_value }}</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)">
                <EditOutlined /> 编辑
              </a-button>
              <a-popconfirm
                title="确定要删除该配置吗？"
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
        <a-form-item label="配置键" name="key">
          <a-input
            v-model:value="formState.key"
            placeholder="请输入配置键"
            :disabled="isEdit"
          />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="值类型" name="value_type">
              <a-select v-model:value="formState.value_type" placeholder="请选择值类型">
                <a-select-option value="string">字符串</a-select-option>
                <a-select-option value="int">整数</a-select-option>
                <a-select-option value="bool">布尔</a-select-option>
                <a-select-option value="json">JSON</a-select-option>
                <a-select-option value="encrypted">加密</a-select-option>
                <a-select-option value="options">选项列表</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="分组" name="group">
              <a-input v-model:value="formState.group" placeholder="请输入分组" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="配置值" name="value" v-if="formState.value_type !== 'encrypted'">
          <a-textarea
            v-model:value="formState.value"
            placeholder="请输入配置值"
            :rows="formState.value_type === 'json' || formState.value_type === 'options' ? 6 : 3"
          />
        </a-form-item>

        <a-form-item label="加密值" name="encrypted_value" v-if="formState.value_type === 'encrypted'">
          <a-input-password v-model:value="formState.encrypted_value" placeholder="请输入加密值" />
        </a-form-item>

        <a-form-item label="描述" name="desc">
          <a-textarea v-model:value="formState.desc" placeholder="请输入描述" :rows="2" />
        </a-form-item>

        <a-form-item label="状态" name="is_active">
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
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons-vue'
import { configApi } from '@/api/config'
import type { Config } from '@/types'

// 表格列定义
const columns = [
  { title: '配置键', dataIndex: 'key', key: 'key', width: 200 },
  { title: '类型', key: 'value_type', width: 100 },
  { title: '配置值', key: 'display_value', ellipsis: true },
  { title: '分组', dataIndex: 'group', key: 'group', width: 120 },
  { title: '描述', dataIndex: 'desc', key: 'desc', ellipsis: true },
  { title: '状态', key: 'is_active', width: 100 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' },
]

// 状态
const loading = ref(false)
const tableData = ref<Config[]>([])
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})
const searchForm = reactive({
  search: '',
  group: '',
})

// 分组选项（动态加载）
const groupOptions = ref<string[]>([])

const loadGroups = async () => {
  try {
    const res = await configApi.getGroups()
    groupOptions.value = res
  } catch {
    console.warn('加载分组列表失败')
  }
}

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增配置')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  key: '',
  value: '',
  value_type: 'string',
  encrypted_value: '',
  desc: '',
  group: 'default',
  is_active: true,
})

const formRules = {
  key: [{ required: true, message: '请输入配置键' }],
  value_type: [{ required: true, message: '请选择值类型' }],
}

// 获取类型颜色
const getTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    string: 'blue',
    int: 'green',
    bool: 'purple',
    json: 'orange',
    encrypted: 'red',
    options: 'cyan',
  }
  return colors[type] || 'default'
}

// 获取类型文本
const getTypeText = (type: string) => {
  const texts: Record<string, string> = {
    string: '字符串',
    int: '整数',
    bool: '布尔',
    json: 'JSON',
    encrypted: '加密',
    options: '选项',
  }
  return texts[type] || type
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await configApi.getConfigs({
      page: pagination.current,
      size: pagination.pageSize,
      search: searchForm.search,
      group: searchForm.group,
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
  searchForm.group = ''
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
  modalTitle.value = '新增配置'
  currentId.value = ''
  Object.assign(formState, {
    key: '',
    value: '',
    value_type: 'string',
    encrypted_value: '',
    desc: '',
    group: 'default',
    is_active: true,
  })
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: Config) => {
  isEdit.value = true
  modalTitle.value = '编辑配置'
  currentId.value = record.id
  Object.assign(formState, {
    key: record.key,
    value: record.value,
    value_type: record.value_type,
    encrypted_value: '',
    desc: record.desc,
    group: record.group,
    is_active: record.is_active,
  })
  modalVisible.value = true
}

// 删除
const handleDelete = async (record: Config) => {
  try {
    await configApi.deleteConfig(record.id)
    message.success('删除成功')
    loadData()
  } catch (e) {
    message.error('删除失败')
  }
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true

    const data: any = { ...formState }
    if (data.value_type === 'encrypted' && data.encrypted_value) {
      data.value = data.encrypted_value
    }
    delete data.encrypted_value

    if (isEdit.value) {
      await configApi.updateConfig(currentId.value, data)
      message.success('更新成功')
    } else {
      await configApi.createConfig(data)
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
  loadGroups()
})
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.search-card {
  margin-bottom: 16px;
}

.table-card {
  margin-bottom: 16px;
}

.table-toolbar {
  margin-bottom: 16px;
}
</style>
