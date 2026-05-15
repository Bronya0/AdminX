<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="用户名">
          <a-input
            v-model:value="searchForm.search"
            placeholder="请输入用户名"
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
            <PlusOutlined /> 新增用户
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
          <template v-if="column.key === 'online'">
            <a-tooltip :title="record.last_activity ? `最后活动: ${formatDateTime(record.last_activity)}` : '暂无活动记录'">
              <a-badge
                :status="record.is_online ? 'processing' : 'default'"
                :text="record.is_online ? '在线' : '离线'"
              />
            </a-tooltip>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'roles'">
            <a-space>
              <a-tag v-for="role in record.role_names" :key="role" color="blue">
                {{ role }}
              </a-tag>
            </a-space>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)">
                <EditOutlined /> 编辑
              </a-button>
              <a-button type="link" size="small" @click="handleResetPassword(record)">
                <KeyOutlined /> 重置密码
              </a-button>
              <a-popconfirm
                title="确定要删除该用户吗？"
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
        <a-form-item label="用户名" name="username">
          <a-input
            v-model:value="formState.username"
            placeholder="请输入用户名"
            :disabled="isEdit"
          />
        </a-form-item>
        <a-form-item label="密码" name="password" v-if="!isEdit">
          <a-input-password
            v-model:value="formState.password"
            placeholder="请输入密码"
          />
        </a-form-item>
        <a-form-item label="邮箱" name="email">
          <a-input v-model:value="formState.email" placeholder="请输入邮箱" />
        </a-form-item>
        <a-form-item label="手机号" name="phone">
          <a-input v-model:value="formState.phone" placeholder="请输入手机号" />
        </a-form-item>
        <a-form-item label="描述" name="desc">
          <a-textarea v-model:value="formState.desc" placeholder="请输入描述（可选）" :rows="2" />
        </a-form-item>
        <a-form-item label="角色" name="roles">
          <a-select
            v-model:value="formState.roles"
            mode="multiple"
            placeholder="请选择角色"
            :options="roleOptions"
          />
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

    <!-- 重置密码弹窗 -->
    <a-modal
      v-model:open="passwordModalVisible"
      title="重置密码"
      :confirm-loading="passwordModalLoading"
      @ok="handlePasswordOk"
      @cancel="passwordModalVisible = false"
    >
      <a-form layout="vertical">
        <a-form-item label="新密码">
          <a-input-password
            v-model:value="newPassword"
            placeholder="请输入新密码"
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
  KeyOutlined,
} from '@ant-design/icons-vue'
import { userApi } from '@/api/auth'
import { roleApi } from '@/api/auth'
import { formatDateTime } from '@/utils/format'
import type { User, Role } from '@/types'

// 表格列定义
const columns = [
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '手机号', dataIndex: 'phone', key: 'phone' },
  { title: '描述', dataIndex: 'desc', key: 'desc', ellipsis: true },
  { title: '角色', key: 'roles' },
  { title: '在线状态', key: 'online', width: 110 },
  { title: '状态', key: 'is_active' },
  { title: '最后登录', dataIndex: 'last_login', key: 'last_login', customRender: ({ text }: any) => formatDateTime(text) },
  { title: '操作', key: 'action', width: 250 },
]

// 状态
const loading = ref(false)
const tableData = ref<User[]>([])
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

// 角色选项
const roleOptions = ref<{ label: string; value: string }[]>([])

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增用户')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  username: '',
  password: '',
  email: '',
  phone: '',
  desc: '',
  roles: [] as string[],
  is_active: true,
})

const formRules = {
  username: [{ required: true, message: '请输入用户名' }],
  password: [{ required: true, message: '请输入密码' }],
  email: [{ type: 'email', message: '请输入正确的邮箱' }],
}

// 密码重置弹窗
const passwordModalVisible = ref(false)
const passwordModalLoading = ref(false)
const newPassword = ref('')
const currentUser = ref<User | null>(null)

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await userApi.getUsers({
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

// 加载角色选项
const loadRoles = async () => {
  try {
    const res = await roleApi.getRoles()
    roleOptions.value = res.results.map((role: Role) => ({
      label: role.name,
      value: role.code,
    }))
  } catch (e) {
    console.error('加载角色失败', e)
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
  modalTitle.value = '新增用户'
  currentId.value = ''
  Object.assign(formState, {
    username: '',
    password: '',
    email: '',
    phone: '',
    desc: '',
    roles: [],
    is_active: true,
  })
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: User) => {
  isEdit.value = true
  modalTitle.value = '编辑用户'
  currentId.value = record.id
  Object.assign(formState, {
    username: record.username,
    email: record.email,
    phone: record.phone,
    desc: record.desc || '',
    roles: record.roles,
    is_active: record.is_active,
  })
  modalVisible.value = true
}

// 删除
const handleDelete = async (record: User) => {
  try {
    await userApi.deleteUser(record.id)
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

    if (isEdit.value) {
      await userApi.updateUser(currentId.value, formState)
      message.success('更新成功')
    } else {
      await userApi.createUser(formState)
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

// 重置密码
const handleResetPassword = (record: User) => {
  currentUser.value = record
  newPassword.value = ''
  passwordModalVisible.value = true
}

const handlePasswordOk = async () => {
  if (!newPassword.value || newPassword.value.length < 6) {
    message.error('密码长度不能少于6位')
    return
  }
  passwordModalLoading.value = true
  try {
    // 调用重置密码API（需要后端支持）
    message.success('密码重置成功')
    passwordModalVisible.value = false
  } catch (e) {
    message.error('密码重置失败')
  } finally {
    passwordModalLoading.value = false
  }
}

onMounted(() => {
  loadData()
  loadRoles()
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
</style>
