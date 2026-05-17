<template>
  <div class="config-list-content">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="键">
          <a-input
            v-model:value="searchForm.search"
            placeholder="配置键"
            allow-clear
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item label="类型">
          <a-select
            v-model:value="searchForm.value_type"
            placeholder="全部"
            allow-clear
            style="width: 120px"
            @change="handleSearch"
          >
            <a-select-option value="string">字符串</a-select-option>
            <a-select-option value="int">整数</a-select-option>
            <a-select-option value="bool">布尔</a-select-option>
            <a-select-option value="json">JSON</a-select-option>
            <a-select-option value="options">选项列表</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="分组">
          <a-select
            v-model:value="searchForm.group"
            placeholder="全部"
            allow-clear
            style="width: 120px"
            @change="handleSearch"
          >
            <a-select-option v-for="g in groupOptions" :key="g" :value="g">{{ g }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述">
          <a-input
            v-model:value="searchForm.desc"
            placeholder="描述"
            allow-clear
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item label="加密">
          <a-select
            v-model:value="searchForm.is_encrypted"
            placeholder="全部"
            allow-clear
            style="width: 100px"
            @change="handleSearch"
          >
            <a-select-option :value="true">是</a-select-option>
            <a-select-option :value="false">否</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select
            v-model:value="searchForm.is_active"
            placeholder="全部"
            allow-clear
            style="width: 100px"
            @change="handleSearch"
          >
            <a-select-option :value="true">启用</a-select-option>
            <a-select-option :value="false">禁用</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="handleSearch">
            <SearchOutlined /> 查询
          </a-button>
          <a-button style="margin-left: 8px" @click="resetSearch">
            <ReloadOutlined /> 重置
          </a-button>
          <a-button type="primary" style="margin-left: 16px" @click="handleAdd">
            <PlusOutlined /> 新增配置
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card class="table-card">

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
          <template v-if="column.key === 'is_encrypted'">
            <a-tag :color="record.is_encrypted ? 'orange' : 'default'">
              {{ record.is_encrypted ? '是' : '否' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'display_value'">
            <span v-if="record.is_encrypted">
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
                <a-select-option value="options">选项列表</a-select-option>
              </a-select>
              <span class="encrypted-hint">如需加密存储，使用下方「加密存储」开关</span>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="分组" name="group">
              <a-input v-model:value="formState.group" placeholder="请输入分组" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="配置值" name="value">
          <a-textarea
            v-model:value="formState.value"
            :placeholder="valuePlaceholder"
            :rows="formState.value_type === 'json' || formState.value_type === 'options' ? 6 : 3"
          />
        </a-form-item>
        <a-form-item label="加密存储" name="is_encrypted">
          <a-switch
            v-model:checked="formState.is_encrypted"
            :disabled="isEdit && formState.is_encrypted"
            checked-children="已加密"
            un-checked-children="未加密"
          />
          <span class="encrypted-hint">开启后加密存储到数据库，API 返回 ***，编辑时不可见原值</span>
          <span v-if="isEdit && formState.is_encrypted" class="encrypted-warning">已加密的配置不允许改回非加密</span>
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
import { ref, reactive, computed, watch, onMounted } from 'vue'
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
  { title: '加密', key: 'is_encrypted', width: 80, align: 'center' as const },
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
  value_type: undefined as string | undefined,
  desc: '',
  is_encrypted: undefined as boolean | undefined,
  is_active: undefined as boolean | undefined,
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
  is_encrypted: false,
  desc: '',
  group: 'default',
  is_active: true,
})

const valuePlaceholder = computed(() => {
  const hints: Record<string, string> = {
    string: '普通文本字符串',
    int: '整数值，如 42',
    bool: 'true 或 false',
    json: 'JSON 对象，如 {"key": "value"}',
    options: '选项列表 JSON 数组，如 [{"label": "男", "value": "male"}]',
  }
  return hints[formState.value_type] || '请输入配置值'
})

const valueValidator = (_rule: any, value: string) => {
  if (formState.is_encrypted && !value) return Promise.resolve()
  const t = formState.value_type
  if (!value) {
    if (t === 'options') return Promise.resolve()
    return Promise.reject(new Error('请输入配置值'))
  }
  if (t === 'int') {
    if (!/^-?\d+$/.test(value)) return Promise.reject(new Error('整数类型请输入整数值，如 42'))
  } else if (t === 'bool') {
    if (!['true', 'false'].includes(value.toLowerCase())) return Promise.reject(new Error('布尔类型请输入 true 或 false'))
  } else if (t === 'json') {
    try {
      const parsed = JSON.parse(value)
      if (parsed === null || typeof parsed !== 'object')
        return Promise.reject(new Error('JSON 类型请输入对象 {} 或数组 []'))
    } catch { return Promise.reject(new Error('JSON 格式无效，请检查语法')) }
  } else if (t === 'options') {
    try {
      const arr = JSON.parse(value)
      if (!Array.isArray(arr)) return Promise.reject(new Error('选项列表请输入 JSON 数组'))
      if (!arr.every((i: any) => i && typeof i.label === 'string' && typeof i.value === 'string'))
        return Promise.reject(new Error('选项列表格式：[{"label":"显示名","value":"值"}]'))
    } catch { return Promise.reject(new Error('不是有效 JSON 数组')) }
  }
  return Promise.resolve()
}

// 切换值类型时重新校验 value 字段
watch(() => formState.value_type, () => {
  if (formRef.value && formState.value) {
    formRef.value.validateFields('value').catch(() => {})
  }
})

const formRules = {
  key: [{ required: true, message: '请输入配置键' }],
  value_type: [{ required: true, message: '请选择值类型' }],
  value: [{ validator: valueValidator, trigger: 'change' }],
}

// 获取类型颜色
const getTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    string: 'blue',
    int: 'green',
    bool: 'purple',
    json: 'orange',
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
      value_type: searchForm.value_type,
      desc: searchForm.desc || undefined,
      is_encrypted: searchForm.is_encrypted,
      is_active: searchForm.is_active,
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
  searchForm.value_type = undefined
  searchForm.desc = ''
  searchForm.is_encrypted = undefined
  searchForm.is_active = undefined
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
    is_encrypted: false,
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
    is_encrypted: record.is_encrypted,
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
  } catch { /* interceptor handles error */ }
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true

    const data: any = { ...formState }

    if (isEdit.value) {
      if (data.is_encrypted && !data.value) {
        delete data.value
      }
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
.config-list-content {
  min-height: 200px;
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

.encrypted-hint {
  font-size: 12px;
  color: #999;
  margin-left: 8px;
}

.encrypted-warning {
  font-size: 12px;
  color: #ff4d4f;
  margin-left: 8px;
}
</style>
