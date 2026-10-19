<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <SearchBar>
        <SearchField label="用户名">
          <a-input
            v-model:value="searchForm.search"
            placeholder="请输入用户名"
            allow-clear
            @pressEnter="handleSearch"
          />
        </SearchField>
        <SearchField label="角色">
          <a-select
            v-model:value="searchForm.role"
            placeholder="全部"
            allow-clear
            @change="handleSearch"
          >
            <a-select-option v-for="r in roleOptions" :key="r.value" :value="r.value">{{ r.label }}</a-select-option>
          </a-select>
        </SearchField>
        <SearchField label="状态">
          <a-select
            v-model:value="searchForm.is_active"
            placeholder="全部"
            allow-clear
            @change="handleSearch"
          >
            <a-select-option :value="true">启用</a-select-option>
            <a-select-option :value="false">禁用</a-select-option>
          </a-select>
        </SearchField>

        <template #actions>
          <a-button type="primary" @click="handleSearch">
            <SearchOutlined /> 查询
          </a-button>
          <a-button @click="resetSearch">
            <ReloadOutlined /> 重置
          </a-button>
          <a-button type="primary" @click="handleAdd">
            <PlusOutlined /> 新增用户
          </a-button>
        </template>
      </SearchBar>
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
              <a-tag v-for="(role, i) in record.roles" :key="role" color="blue"
                style="cursor: pointer" @click="router.push({ name: 'roles', query: { search: role } })">
                {{ record.role_labels?.[i] || role }}
              </a-tag>
            </a-space>
          </template>
          <template v-if="column.key === 'action'">
            <div class="action-cell">
              <a-button type="link" size="small" @click="handleEdit(record)">
                <EditOutlined /> 编辑
              </a-button>
              <!-- 次要操作收进下拉：四个按钮平铺会把操作列挤出表格 -->
              <a-dropdown :trigger="['click']">
                <a-button type="link" size="small">
                  更多 <DownOutlined />
                </a-button>
                <template #overlay>
                  <a-menu @click="(info: any) => handleMoreAction(info, record)">
                    <a-menu-item key="reset">
                      <KeyOutlined /> 重置密码
                    </a-menu-item>
                    <a-menu-item v-if="record.is_online && !isCurrentUser(record)" key="logout">
                      <LogoutOutlined /> 强制下线
                    </a-menu-item>
                    <!-- 删除也收进下拉；当前登录用户不出现，避免删掉自己 -->
                    <a-menu-item v-if="!isCurrentUser(record)" key="delete" danger>
                      <DeleteOutlined /> 删除
                    </a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </div>
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
            :disabled="editingSuperuser"
          />
          <div v-if="editingSuperuser" class="form-hint">
            超级管理员的权限来自账号本身，与角色无关，角色仅用于展示
          </div>
        </a-form-item>
        <a-form-item label="首页" name="home_page">
          <a-tree-select
            v-model:value="formState.home_page"
            :tree-data="menuTreeOptions"
            placeholder="默认首页（可选）"
            allow-clear
            style="width: 100%"
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
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  KeyOutlined,
  LogoutOutlined,
  DownOutlined,
} from '@ant-design/icons-vue'
import { userApi, roleApi, authApi } from '@/api/auth'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/format'
import SearchBar from '@/components/SearchBar.vue'
import SearchField from '@/components/SearchField.vue'
import type { Menu, User } from '@/types'

const router = useRouter()
const userStore = useUserStore()

// 表格列定义
const columns = [
  { title: '用户名', dataIndex: 'username', key: 'username', width: 180 },
  { title: '邮箱', dataIndex: 'email', key: 'email', width: 160, ellipsis: true },
  { title: '手机号', dataIndex: 'phone', key: 'phone', width: 120 },
  { title: '描述', dataIndex: 'desc', key: 'desc', width: 240, ellipsis: true },
  // 角色/状态这类窄信息固定窄列，不设宽度会按内容抢空间，把操作列挤出去
  { title: '角色', key: 'roles', width: 130 },
  { title: '在线状态', key: 'online', width: 120 },
  { title: '状态', key: 'is_active', width: 100 },
  { title: '最后登录', dataIndex: 'last_login', key: 'last_login', width: 160, customRender: ({ text }: any) => formatDateTime(text) },
  // 操作列不写死宽度：由内容自适应，写死会被 4 个按钮撑出去
  { title: '操作', key: 'action' },
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
    const isSelectable = !hasChildren && Boolean(item.path && allowedPaths.has(item.path))

    if (!isSelectable && !children.length) {
      return []
    }

    // key 必须与 value 一致：antd 的 TreeNode 两者不同会报警，
    // 选中态靠 value 回显，key 不一致时容易选不中或回显错乱
    const value = isSelectable ? item.path : `__group__:${item.code}`
    return [{
      title: item.name,
      value,
      key: value,
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

// 收集全部菜单路径：未选角色时用它不做过过滤，直接给出所有可选首页
const collectMenuPaths = (items: Menu[]): string[] =>
  items.flatMap(item => [item.path, ...collectMenuPaths(item.children || [])]).filter(Boolean)

// 找第一个可选的叶子菜单：新建用户时用它做首页默认值，
// 免得“必选”变成每次都要手动挑一项。
const firstSelectablePath = (items: TreeSelectOption[]): string => {
  for (const item of items) {
    if (item.selectable && !item.value.startsWith('__group__:')) return item.value
    const child = item.children ? firstSelectablePath(item.children) : ''
    if (child) return child
  }
  return ''
}

// 首页必选：当前值不可用时自动落到第一个可选项
const ensureHomePage = () => {
  const valid = (function has(items: TreeSelectOption[]): boolean {
    return items.some(i => (i.selectable && i.value === formState.home_page) || (i.children ? has(i.children) : false))
  })(menuTreeOptions.value)
  if (!valid) formState.home_page = firstSelectablePath(menuTreeOptions.value)
}

let menuOptionsSeq = 0
// 菜单首页选项（根据选中角色过滤）
const loadMenuOptions = async (roles: string[]) => {
  const seq = ++menuOptionsSeq
  // 未选角色时不做权限过滤：新建用户往往先看首页有哪些可选，
  // 此时给一个空下拉会让人以为功能坏了。选角色后会自动收窄并清掉越界的选项。
  if (!roles.length) {
    if (!fullMenuTree.value.length) {
      await loadFullMenuTree()
    }
    if (seq !== menuOptionsSeq) return
    menuTreeOptions.value = buildMenuTreeOptions(fullMenuTree.value, new Set(collectMenuPaths(fullMenuTree.value)))
    ensureHomePage()
    return
  }
  if (!fullMenuTree.value.length) {
    await loadFullMenuTree()
  }
  try {
    const menus = await roleApi.accessibleMenus(roles)
    // 请求序号守卫：快速切换角色时只让最后一次回包生效，
    // 否则慢回包会覆盖新选项，首页下拉与所选角色错位
    if (seq !== menuOptionsSeq) return
    // 不过滤仪表盘：普通用户这类角色可能只绑了仪表盘，排除它会让首页下拉直接空掉
    const allowedPaths = new Set(
      menus
        .filter(m => m.path)
        .map(m => m.path)
    )
    menuTreeOptions.value = buildMenuTreeOptions(fullMenuTree.value, allowedPaths)

    if (formState.home_page && !allowedPaths.has(formState.home_page)) {
      formState.home_page = ''
    }
    ensureHomePage()
  } catch (e) {
    console.error('加载菜单失败', e)
  }
}

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增用户')
const isEdit = ref(false)
// 正在编辑的是超级管理员：超管的权限来自 is_superuser，角色只影响列表展示，
// 所以角色选择器置灰，避免“改了角色好像就能提权/降权”的误会。
const editingSuperuser = ref(false)
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
  // 普通用户至少要有一个角色：无角色账号登录后看不到任何菜单。
  // 编辑超管时角色只读且不提交，这里不能拦，否则改个描述都保存不了。
  roles: editingSuperuser.value
    ? []
    : [{ required: true, type: 'array' as const, min: 1, message: '请至少选择一个角色', trigger: 'change' }],
  home_page: [{ required: true, message: '请选择登录后的首页', trigger: 'change' }],
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
    roleOptions.value = res.map((role: { name: string; display_name?: string }) => ({
      label: role.display_name || role.name,
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
  editingSuperuser.value = false
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
  // 角色未变时 watch 不会触发，这里显式加载一次首页菜单选项
  loadMenuOptions(formState.roles)
}

// 编辑
const handleEdit = (record: User) => {
  isEdit.value = true
  editingSuperuser.value = !!record.is_superuser
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
  // 角色未变时 watch 不会触发，这里显式加载一次首页菜单选项
  loadMenuOptions(formState.roles)
}

// 删除：收进下拉后失去 popconfirm，改用 Modal 确认
const handleDelete = (record: User) => {
  Modal.confirm({
    title: '删除用户',
    content: `确定删除用户 ${record.username}？删除后该账号不能再登录。`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      await userApi.deleteUser(record.id)
      message.success('删除成功')
      loadData()
    },
  })
}

const isCurrentUser = (record: User) => record.id === userStore.user?.id

// 强制下线：后端写 last_logout，该用户已签发的 token 立即失效
const handleForceLogout = (record: User) => {
  Modal.confirm({
    title: '强制下线',
    content: `确定让 ${record.username} 立即下线？该用户已登录的会话会立即失效。`,
    okText: '确定',
    cancelText: '取消',
    onOk: async () => {
      await userApi.forceLogout(record.id)
      message.success('已强制下线')
      loadData()
    },
  })
}

// 操作列「更多」下拉：次要操作都在这里
const handleMoreAction = (info: { key: string }, record: User) => {
  if (info.key === 'reset') handleResetPassword(record)
  else if (info.key === 'logout') handleForceLogout(record)
  else if (info.key === 'delete') handleDelete(record)
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true

    if (isEdit.value) {
      // 显式构造 payload：编辑绝不能带 password——后端收到非空密码就会改密并吊销该用户全部 token，
      // 而 formState.password 可能残留着上一次「新增」输入的值（取消弹窗不会清空它）。
      await userApi.updateUser(currentId.value, {
        email: formState.email,
        phone: formState.phone,
        desc: formState.desc,
        // 超管的角色不参与授权，后端也会忽略，索性不提交
        ...(editingSuperuser.value ? {} : { roles: formState.roles }),
        is_active: formState.is_active,
        home_page: formState.home_page,
      })
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
  if (!newPassword.value?.trim()) {
    message.error('请输入新密码')
    return
  }
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

.form-hint {
  margin-top: 6px;
  color: inherit;
  opacity: 0.65;
  font-size: 12px;
  line-height: 1.4;
}

/* 操作列自适应：不换行、不被列宽截断，宽度由按钮内容决定 */
.action-cell {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
</style>
