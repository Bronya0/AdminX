<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="角色名称">
          <a-input
            v-model:value="searchForm.search"
            placeholder="请输入角色名称"
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
            <PlusOutlined /> 新增角色
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
              <a-button type="link" size="small" @click="handlePermission(record)">
                <SafetyOutlined /> 权限
              </a-button>
              <a-popconfirm title="确定要删除该角色吗？" @confirm="handleDelete(record)">
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
      <a-form ref="formRef" :model="formState" :rules="formRules" layout="vertical">
        <a-form-item label="角色名称" name="name">
          <a-input v-model:value="formState.name" placeholder="请输入角色名称" />
        </a-form-item>
        <a-form-item label="角色编码" name="code">
          <a-input v-model:value="formState.code" placeholder="请输入角色编码" :disabled="isEdit" />
        </a-form-item>
        <a-form-item label="描述" name="desc">
          <a-textarea v-model:value="formState.desc" placeholder="请输入描述" :rows="3" />
        </a-form-item>
        <a-form-item label="状态" name="is_active">
          <a-switch v-model:checked="formState.is_active" checked-children="启用" un-checked-children="禁用" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 权限分配弹窗 -->
    <a-modal
      v-model:open="permissionModalVisible"
      title="分配权限"
      :confirm-loading="permissionModalLoading"
      @ok="handlePermissionOk"
      @cancel="permissionModalVisible = false"
      width="800px"
    >
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="permissions" tab="功能权限">
          <div class="permission-toolbar">
            <a-input-search
              v-model:value="permissionSearch"
              placeholder="搜索权限名称..."
              allow-clear
              style="width: 240px"
              @change="filterPermissionTree"
            />
            <a-space>
              <a-button size="small" @click="expandAllPermissions">展开全部</a-button>
              <a-button size="small" @click="collapseAllPermissions">收起全部</a-button>
              <a-button size="small" @click="selectAllPermissions">全选</a-button>
              <a-button size="small" @click="deselectAllPermissions">取消全选</a-button>
            </a-space>
            <span class="selected-count">已选 {{ selectedPermissions.length }} 项</span>
          </div>
          <div class="tree-scroll">
            <a-tree
              v-model:checkedKeys="selectedPermissions"
              v-model:expandedKeys="permissionExpandedKeys"
              checkable
              :tree-data="filteredPermissionTreeData"
              :default-expand-all="false"
            />
          </div>
        </a-tab-pane>
        <a-tab-pane key="menus" tab="菜单权限">
          <div class="permission-toolbar">
            <a-input-search
              v-model:value="menuSearch"
              placeholder="搜索菜单名称..."
              allow-clear
              style="width: 240px"
              @change="filterMenuTree"
            />
            <a-space>
              <a-button size="small" @click="expandAllMenus">展开全部</a-button>
              <a-button size="small" @click="collapseAllMenus">收起全部</a-button>
              <a-button size="small" @click="selectAllMenus">全选</a-button>
              <a-button size="small" @click="deselectAllMenus">取消全选</a-button>
            </a-space>
            <span class="selected-count">已选 {{ selectedMenus.length }} 项</span>
          </div>
          <div class="tree-scroll">
            <a-tree
              v-model:checkedKeys="selectedMenus"
              v-model:expandedKeys="menuExpandedKeys"
              checkable
              :tree-data="filteredMenuTreeData"
              :default-expand-all="false"
            />
          </div>
        </a-tab-pane>
      </a-tabs>
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
  SafetyOutlined,
} from '@ant-design/icons-vue'
import { roleApi, permissionApi } from '@/api/auth'
import { menuApi } from '@/api/menu'
import { menusToTreeData } from '@/utils/menuTree'
import { filterTreeBySearch } from '@/utils/menuTree'
import type { Role, Permission } from '@/types'

// 表格列定义
const columns = [
  { title: '角色名称', dataIndex: 'name', key: 'name' },
  { title: '角色编码', dataIndex: 'code', key: 'code' },
  { title: '描述', dataIndex: 'desc', key: 'desc', ellipsis: true },
  { title: '状态', key: 'is_active' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'action', width: 250 },
]

// 状态
const loading = ref(false)
const tableData = ref<Role[]>([])
const pagination = reactive({
  current: 1, pageSize: 10, total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})
const searchForm = reactive({ search: '' })

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增角色')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  name: '', code: '', desc: '', is_active: true,
})
const formRules = {
  name: [{ required: true, message: '请输入角色名称' }],
  code: [{ required: true, message: '请输入角色编码' }],
}

// 权限弹窗状态
const permissionModalVisible = ref(false)
const permissionModalLoading = ref(false)
const activeTab = ref('permissions')
const currentRole = ref<Role | null>(null)
const selectedPermissions = ref<string[]>([])
const selectedMenus = ref<string[]>([])

// 权限树
const permissionSearch = ref('')
const permissionExpandedKeys = ref<string[]>([])
const originalPermissionTreeData = ref<any[]>([])
const filteredPermissionTreeData = ref<any[]>([])

// 菜单树
const menuSearch = ref('')
const menuExpandedKeys = ref<string[]>([])
const originalMenuTreeData = ref<any[]>([])
const filteredMenuTreeData = ref<any[]>([])

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await roleApi.getRoles({ page: pagination.current, search: searchForm.search })
    tableData.value = res.results
    pagination.total = res.count
  } finally {
    loading.value = false
  }
}

// 加载权限树
const loadPermissions = async () => {
  try {
    const perms = await permissionApi.getPermissions()
    // 按 content_type 分组
    const grouped: Record<string, any[]> = {}
    perms.forEach((perm: Permission) => {
      const group = perm.content_type_name || '其他'
      if (!grouped[group]) grouped[group] = []
      grouped[group].push({ title: perm.name, key: perm.codename, value: perm.codename })
    })
    originalPermissionTreeData.value = Object.entries(grouped).map(([key, children]) => ({
      title: key,
      key: `group-${key}`,
      children,
    }))
    filteredPermissionTreeData.value = [...originalPermissionTreeData.value]
  } catch (e) {
    console.error('加载权限失败', e)
  }
}

// 加载菜单树
const loadMenus = async () => {
  try {
    const tree = await menuApi.getMenuTree()
    originalMenuTreeData.value = menusToTreeData(tree, 'code')
    filteredMenuTreeData.value = [...originalMenuTreeData.value]
  } catch (e) {
    console.error('加载菜单失败', e)
  }
}

// 权限树过滤
const filterPermissionTree = () => {
  filteredPermissionTreeData.value = filterTreeBySearch(originalPermissionTreeData.value, permissionSearch.value)
  if (permissionSearch.value) expandAllPermissions()
}
const expandAllPermissions = () => {
  const collect = (nodes: any[]): string[] => {
    const keys: string[] = []
    for (const n of nodes) {
      keys.push(n.key)
      if (n.children) keys.push(...collect(n.children))
    }
    return keys
  }
  permissionExpandedKeys.value = collect(filteredPermissionTreeData.value)
}
const collapseAllPermissions = () => { permissionExpandedKeys.value = [] }
const selectAllPermissions = () => {
  const all = originalPermissionTreeData.value.flatMap(g => g.children.map((c: any) => c.key))
  selectedPermissions.value = [...new Set([...selectedPermissions.value, ...all])]
}
const deselectAllPermissions = () => { selectedPermissions.value = [] }

// 菜单树过滤
const filterMenuTree = () => {
  filteredMenuTreeData.value = filterTreeBySearch(originalMenuTreeData.value, menuSearch.value)
  if (menuSearch.value) expandAllMenus()
}
const expandAllMenus = () => {
  const collect = (nodes: any[]): string[] => {
    const keys: string[] = []
    for (const n of nodes) {
      keys.push(n.key)
      if (n.children) keys.push(...collect(n.children))
    }
    return keys
  }
  menuExpandedKeys.value = collect(filteredMenuTreeData.value)
}
const collapseAllMenus = () => { menuExpandedKeys.value = [] }
const selectAllMenus = () => {
  const collect = (nodes: any[]): string[] => {
    const keys: string[] = []
    for (const n of nodes) {
      keys.push(n.key)
      if (n.children) keys.push(...collect(n.children))
    }
    return keys
  }
  const all = collect(originalMenuTreeData.value)
  selectedMenus.value = [...new Set([...selectedMenus.value, ...all])]
}
const deselectAllMenus = () => { selectedMenus.value = [] }

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
  modalTitle.value = '新增角色'
  currentId.value = ''
  Object.assign(formState, { name: '', code: '', desc: '', is_active: true })
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: Role) => {
  isEdit.value = true
  modalTitle.value = '编辑角色'
  currentId.value = record.id
  Object.assign(formState, { name: record.name, code: record.code, desc: record.desc, is_active: record.is_active })
  modalVisible.value = true
}

// 删除
const handleDelete = async (record: Role) => {
  try {
    await roleApi.deleteRole(record.id)
    message.success('删除成功')
    loadData()
  } catch { message.error('删除失败') }
}

// 权限分配
const handlePermission = async (record: Role) => {
  currentRole.value = record
  selectedPermissions.value = record.permissions || []
  selectedMenus.value = record.menus || []
  permissionSearch.value = ''
  menuSearch.value = ''
  await Promise.all([loadPermissions(), loadMenus()])
  permissionModalVisible.value = true
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true
    if (isEdit.value) {
      await roleApi.updateRole(currentId.value, formState)
      message.success('更新成功')
    } else {
      await roleApi.createRole(formState)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadData()
  } catch (e) { console.error(e) }
  finally { modalLoading.value = false }
}
const handleModalCancel = () => { modalVisible.value = false; formRef.value?.resetFields() }

// 权限确认
const handlePermissionOk = async () => {
  if (!currentRole.value) return
  permissionModalLoading.value = true
  try {
    await roleApi.updateRole(currentRole.value.id, {
      permissions: selectedPermissions.value,
      menus: selectedMenus.value,
    })
    message.success('权限分配成功')
    permissionModalVisible.value = false
    loadData()
  } catch { message.error('权限分配失败') }
  finally { permissionModalLoading.value = false }
}

onMounted(loadData)
</script>

<style scoped>
.page-container { padding: 24px; }
.search-card { margin-bottom: 16px; }
.table-card { margin-bottom: 16px; }

.permission-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.selected-count {
  font-size: 12px;
  color: #999;
  margin-left: auto;
}

.tree-scroll {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  padding: 8px;
}
</style>