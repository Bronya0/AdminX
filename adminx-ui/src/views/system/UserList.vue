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
        <a-form-item label="角色">
          <a-select
            v-model:value="searchForm.role"
            placeholder="全部"
            allow-clear
            style="width: 150px"
            @change="handleSearch"
          >
            <a-select-option v-for="r in roleOptions" :key="r.value" :value="r.value">{{ r.label }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="在线">
          <a-select
            v-model:value="searchForm.is_online"
            placeholder="全部"
            allow-clear
            style="width: 120px"
            @change="handleSearch"
          >
            <a-select-option value="true">在线</a-select-option>
            <a-select-option value="false">离线</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select
            v-model:value="searchForm.is_active"
            placeholder="全部"
            allow-clear
            style="width: 120px"
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
            <PlusOutlined /> 新增用户
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
              <a-tag v-for="role in record.role_names" :key="role" color="blue"
                style="cursor: pointer" @click="router.push({ name: 'roles', query: { search: role } })">
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
                v-if="!isCurrentUser(record)"
                title="确定要删除该用户吗？"
                @confirm="handleDelete(record)"
              >
                <a-button type="link" danger size="small">
                  <DeleteOutlined /> 删除
                </a-button>
              </a-popconfirm>
              <a-tooltip v-else title="不能删除当前登录用户">
                <a-button type="link" danger size="small" disabled>
                  <DeleteOutlined /> 删除
                </a-button>
              </a-tooltip>
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
        <a-form-item label="首页">
          <a-tree-select
            v-model:value="formState.home_page"
            :tree-data="menuTreeOptions"
            placeholder="默认"
            allow-clear
            style="width: 100%"
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
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  KeyOutlined,
} from '@ant-design/icons-vue'
import { userApi, roleApi, authApi } from '@/api/auth'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/format'
import type { Menu, User } from '@/types'

const router = useRouter()
const userStore = useUserStore()

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
  is_active: undefined as boolean | undefined,
  role: undefined as string | undefined,
  is_online: undefined as string | undefined,
})

// 角色选项
const roleOptions = ref<{ label: string; value: string }[]>([])

// 完整菜单树（用于用户首页树形选择）
const fullMenuTree = ref<Menu[]>([])
type TreeSelectOption = {
  title: string,
  value: string,
  key: string,
  selectable?: boolean,
  children?: TreeSelectOption[],
}
const menuTreeOptions = ref<TreeSelectOption[]>([])

const buildMenuTreeOptions = (items: Menu[], allowedPaths: Set<string>): TreeSelectOption[] => {
  return items.flatMap((item) => {
    const children = buildMenuTreeOptions(item.children || [], allowedPaths)
    // 仅叶子菜单（无子菜单）可选为首页，父级菜单没有实际页面组件，选中会导致空白页
    const hasChildren = Boolean(item.children && item.children.length > 0)
    const isSelectable = !hasChildren && Boolean(item.path && item.path !== '/dashboard' && allowedPaths.has(item.path))

    if (!isSelectable && !children.length) {
      return []
    }

    return [{
      title: item.name,
      value: isSelectable ? item.path : `__group__:${item.code}`,
      key: item.code,
      selectable: isSelectable,
      children: children.length ? children : undefined,
    }]
  })
}

const loadFullMenuTree = async () => {
  try {
    fullMenuTree.value = await roleApi.getMenuTree()
  } catch (e) {
    console.error('加载菜单树失败', e)
  }
}

// 菜单首页选项（根据选中角色过滤）
const loadMenuOptions = async (roles: string[]) => {
  if (!roles.length) {
    menuTreeOptions.value = []
    formState.home_page = ''
    return
  }
  if (!fullMenuTree.value.length) {
    await loadFullMenuTree()
  }
  try {
    const menus = await roleApi.accessibleMenus(roles)
    const allowedPaths = new Set(
      menus
        .filter(m => m.path && m.path !== '/dashboard')
        .map(m => m.path)
    )
    menuTreeOptions.value = buildMenuTreeOptions(fullMenuTree.value, allowedPaths)

    if (formState.home_page && !allowedPaths.has(formState.home_page)) {
      formState.home_page = ''
    }
  } catch (e) {
    console.error('加载菜单失败', e)
  }
}

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
  home_page: '',
})

watch(() => formState.roles, (val) => {
  loadMenuOptions(val)
})

// 密码策略驱动的校验（后端 GET /policy/policy/），替换此前写死的 6 位下限
const policyRules = ref<Array<{ pattern: RegExp; message: string }>>([])

const applyPasswordPolicy = (p: {
  min_length: number
  require_upper: boolean
  require_lower: boolean
  require_digit: boolean
  require_special: boolean
  is_active: boolean
}) => {
  if (!p || !p.is_active) return
  const rules: Array<{ pattern: RegExp; message: string }> = []
  if (p.min_length > 0) {
    rules.push({ pattern: new RegExp(`.{${p.min_length},}`), message: `密码长度不能少于${p.min_length}位` })
  }
  if (p.require_upper) rules.push({ pattern: /[A-Z]/, message: '密码必须包含大写字母' })
  if (p.require_lower) rules.push({ pattern: /[a-z]/, message: '密码必须包含小写字母' })
  if (p.require_digit) rules.push({ pattern: /\d/, message: '密码必须包含数字' })
  if (p.require_special) rules.push({ pattern: /[^A-Za-z0-9]/, message: '密码必须包含特殊字符' })
  policyRules.value = rules
}

const validatePassword = (pwd: string): string | null => {
  for (const r of policyRules.value) {
    if (!r.pattern.test(pwd)) return r.message
  }
  return null
}

const formRules = computed(() => ({
  username: [{ required: true, message: '请输入用户名' }],
  password: [
    { required: true, message: '请输入密码' },
    ...policyRules.value.map((r) => ({ pattern: r.pattern, message: r.message, trigger: 'blur' })),
  ],
  email: [{ type: 'email', message: '请输入正确的邮箱' }],
}))

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
      is_active: searchForm.is_active,
      role: searchForm.role,
      is_online: searchForm.is_online,
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
    const res = await userApi.getRoleOptions()
    roleOptions.value = res.map((role: { name: string }) => ({
      label: role.name,
      value: role.name,
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
  searchForm.is_active = undefined
  searchForm.role = undefined
  searchForm.is_online = undefined
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
    home_page: '',
  })
  modalVisible.value = true
  // 清除上次的校验错误状态（红色提示），避免打开弹窗时残留
  formRef.value?.clearValidate?.()
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
    home_page: record.home_page || '',
  })
  modalVisible.value = true
  // 清除上次的校验错误状态
  formRef.value?.clearValidate?.()
}

// 删除
const handleDelete = async (record: User) => {
  try {
    await userApi.deleteUser(record.id)
    message.success('删除成功')
    loadData()
  } catch { /* interceptor handles error */ }
}

const isCurrentUser = (record: User) => record.id === userStore.user?.id

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
  const policyError = validatePassword(newPassword.value)
  if (policyError) {
    message.error(policyError)
    return
  }
  passwordModalLoading.value = true
  try {
    if (!currentUser.value) return
    await userApi.updateUser(currentUser.value.id, { password: newPassword.value })
    message.success('密码重置成功')
    passwordModalVisible.value = false
    // 清空密码，避免弹窗残留导致下次打开可见上次输入
    newPassword.value = ''
  } finally {
    passwordModalLoading.value = false
  }
}

onMounted(() => {
  loadData()
  loadRoles()
  loadFullMenuTree()
  // 拉取密码策略驱动前端校验（失败时静默，由后端校验兜底）
  authApi.getPasswordPolicy().then(applyPasswordPolicy).catch(() => {})
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
