<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <SearchBar>
        <SearchField label="关键词" style="flex: 1 1 260px">
          <a-input
            v-model:value="searchForm.search"
            placeholder="显示名称/角色标识/描述"
            allow-clear
            @pressEnter="handleSearch"
          />
        </SearchField>
        <SearchField label="状态" style="flex: 0 0 130px">
          <a-select
            v-model:value="searchForm.is_active"
            placeholder="全部"
            allow-clear
          >
            <a-select-option :value="true">启用</a-select-option>
            <a-select-option :value="false">禁用</a-select-option>
          </a-select>
        </SearchField>
        <SearchField label="类型" style="flex: 0 0 150px">
          <a-select
            v-model:value="searchForm.is_system"
            placeholder="全部"
            allow-clear
          >
            <a-select-option :value="true">系统内置</a-select-option>
            <a-select-option :value="false">自定义</a-select-option>
          </a-select>
        </SearchField>
        <SearchField label="权限" style="flex: 0 0 200px">
          <a-select
            v-model:value="searchForm.menu"
            placeholder="全部"
            show-search
            allow-clear
            option-filter-prop="label"
            :options="menuOptions"
          />
        </SearchField>
        <template #actions>
          <span class="list-count">共 {{ pagination.total }} 条</span>
          <a-button type="primary" @click="handleSearch">
            <SearchOutlined /> 查询
          </a-button>
          <a-button @click="resetSearch">
            <ReloadOutlined /> 重置
          </a-button>
          <a-button type="primary" @click="handleAdd">
            <PlusOutlined /> 新增角色
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
        table-layout="fixed"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'menus_count'">
            <a-tooltip v-if="record.menus?.length" placement="left">
              <template #title>
                <div class="menus-tip">
                  <div v-for="m in record.menus" :key="m.id">{{ m.name }}</div>
                </div>
              </template>
              <a-tag :bordered="false" class="count-tag">{{ record.menus.length }}</a-tag>
            </a-tooltip>
            <span v-else>0</span>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'is_system'">
            <a-tag v-if="record.is_system" color="orange">系统内置</a-tag>
            <a-tag v-else color="default">自定义</a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="record.is_system ? handleEdit(record, true) : handleEdit(record)">
                <EditOutlined /> 编辑
              </a-button>
              <a-popconfirm v-if="!record.is_system" title="确定要删除该角色吗？" @confirm="handleDelete(record)">
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
      :footer="viewOnly ? null : undefined"
      @ok="handleModalOk"
      @cancel="handleModalCancel"
      width="800px"
    >
      <a-tabs v-model:activeKey="modalActiveTab">
        <a-tab-pane key="basic" tab="基本信息">
          <a-form ref="formRef" :model="formState" :rules="viewOnly ? {} : formRules" layout="vertical">
            <a-form-item label="角色标识" name="name">
              <a-input v-model:value="formState.name" placeholder="英文标识，权限与绑定用它，如 ops_admin" :disabled="viewOnly" />
            </a-form-item>
            <a-form-item label="显示名称" name="display_name">
              <a-input v-model:value="formState.display_name" placeholder="界面展示用（如 运维管理员），留空则显示角色标识" :disabled="viewOnly" />
            </a-form-item>
            <a-form-item label="描述" name="desc">
              <a-textarea v-model:value="formState.desc" placeholder="请输入描述" :rows="3" :disabled="viewOnly" />
            </a-form-item>
            <a-form-item label="状态" name="is_active">
              <a-switch v-model:checked="formState.is_active" checked-children="启用" un-checked-children="禁用" :disabled="viewOnly" />
            </a-form-item>
          </a-form>
        </a-tab-pane>
        <a-tab-pane key="menus" tab="菜单权限">
          <div class="permission-toolbar">
            <a-input-search v-model:value="menuSearch" placeholder="搜索菜单名称..." allow-clear style="width: 240px" @change="filterMenuTree" />
            <a-space>
              <a-button size="small" @click="expandAllMenus">展开全部</a-button>
              <a-button size="small" @click="collapseAllMenus">收起全部</a-button>
              <a-button v-if="!viewOnly" size="small" @click="selectAllMenus">全选</a-button>
              <a-button v-if="!viewOnly" size="small" @click="deselectAllMenus">取消全选</a-button>
            </a-space>
            <span class="selected-count">已选 {{ selectedMenus.length }} 项</span>
          </div>
          <div class="tree-scroll">
            <a-tree v-model:checkedKeys="selectedMenus" v-model:expandedKeys="menuExpandedKeys"
              checkable :tree-data="filteredMenuTreeData" :default-expand-all="false" :block-node="true">
              <template #title="{ title, path }">
                <span class="menu-node">
                  <span class="menu-name">{{ title }}</span>
                  <span v-if="path" class="menu-path">{{ path }}</span>
                </span>
              </template>
            </a-tree>
          </div>
        </a-tab-pane>
      </a-tabs>
      <a-alert
        v-if="viewOnly"
        message="系统内置角色不可编辑"
        type="warning"
        show-icon
        style="margin-top: 16px"
      />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons-vue'
import { roleApi } from '@/api/auth'
import { menusToTreeData, filterTreeBySearch } from '@/utils/menuTree'
import { formatDateTime } from '@/utils/format'
import SearchBar from '@/components/SearchBar.vue'
import SearchField from '@/components/SearchField.vue'
import type { Menu, Role } from '@/types'

const route = useRoute()

// 表格列定义
const columns = [
  { title: '显示名称', key: 'display_name', width: 160, customRender: ({ record }: any) => record.display_name || record.name },
  { title: '角色标识', dataIndex: 'name', key: 'name', width: 160 },
  { title: '描述', dataIndex: 'desc', key: 'desc', width: 260, ellipsis: true },
  { title: '权限数量', key: 'menus_count', width: 100, align: 'center' as const },
  { title: '状态', key: 'is_active', width: 90 },
  { title: '类型', key: 'is_system', width: 100, align: 'center' as const },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170, customRender: ({ text }: any) => formatDateTime(text) },
  { title: '操作', key: 'action', width: 160 },
]

// 状态
const loading = ref(false)
const tableData = ref<Role[]>([])
const pagination = reactive({
  current: 1, pageSize: 10, total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})
const searchForm = reactive({
  search: '',
  is_active: undefined as boolean | undefined,
  is_system: undefined as boolean | undefined,
  menu: undefined as string | undefined,
})

// 权限下拉选项：按菜单过滤角色，层级用缩进体现，选中值是菜单 code
const menuOptions = ref<{ label: string; value: string }[]>([])
const flattenMenuOptions = (items: Menu[], depth = 0): { label: string; value: string }[] =>
  items.flatMap(item => [
    { label: `${'　'.repeat(depth)}${item.name}`, value: item.code },
    ...flattenMenuOptions(item.children || [], depth + 1),
  ])
const loadMenuOptions = async () => {
  try {
    menuOptions.value = flattenMenuOptions(await roleApi.getMenuTree())
  } catch (e) {
    console.error('加载菜单选项失败', e)
  }
}

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增角色')
const modalActiveTab = ref('basic')
const isEdit = ref(false)
const viewOnly = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  name: '', display_name: '', desc: '', is_active: true,
})
const formRules = {
  name: [{ required: true, message: '请输入角色标识' }],
}

// 菜单树
const menuSearch = ref('')
const menuExpandedKeys = ref<string[]>([])
const originalMenuTreeData = ref<any[]>([])
const filteredMenuTreeData = ref<any[]>([])
const selectedMenus = ref<string[]>([])
// code → 菜单 ID：树上勾的是 code，提交给后端的是 menu_ids
const menuIdByCode = ref<Record<string, number>>({})

const indexMenuIds = (items: Menu[]) => {
  for (const m of items) {
    menuIdByCode.value[m.code] = m.id
    if (m.children?.length) indexMenuIds(m.children)
  }
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await roleApi.getRoles({
      page: pagination.current,
      size: pagination.pageSize,
      search: searchForm.search,
      is_active: searchForm.is_active,
      is_system: searchForm.is_system,
      menu: searchForm.menu,
    })
    tableData.value = res.results
    pagination.total = res.count
  } finally {
    loading.value = false
  }
}

// 加载菜单树
const loadMenus = async () => {
  try {
    const tree = await roleApi.getMenuTree()
    indexMenuIds(tree)
    originalMenuTreeData.value = menusToTreeData(tree, 'code')
    filteredMenuTreeData.value = [...originalMenuTreeData.value]
  } catch (e) {
    console.error('加载菜单失败', e)
  }
}

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
const resetSearch = () => {
  searchForm.search = ''
  searchForm.is_active = undefined
  searchForm.is_system = undefined
  searchForm.menu = undefined
  handleSearch()
}
const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadData()
}

// 新增
const handleAdd = () => {
  isEdit.value = false
  viewOnly.value = false
  modalTitle.value = '新增角色'
  currentId.value = ''
  modalActiveTab.value = 'basic'
  Object.assign(formState, { name: '', display_name: '', desc: '', is_active: true })
  selectedMenus.value = []
  menuExpandedKeys.value = []
  menuSearch.value = ''
  loadMenus()
  modalVisible.value = true
  // 清除上次校验残留
  formRef.value?.clearValidate?.()
}

// 编辑
const handleEdit = (record: Role, readOnly = false) => {
  isEdit.value = true
  viewOnly.value = readOnly
  modalTitle.value = readOnly ? `角色详情 - ${record.display_name || record.name}` : '编辑角色'
  currentId.value = record.id
  modalActiveTab.value = 'basic'
  Object.assign(formState, { name: record.name, display_name: record.display_name || '', desc: record.desc, is_active: record.is_active })
  // 列表返回的是菜单对象数组，而树勾的是菜单 code：不转就一个都勾不上
  selectedMenus.value = (record.menus || [])
    .map(m => (typeof m === 'string' ? m : m.code))
    .filter(Boolean)
  menuExpandedKeys.value = []
  menuSearch.value = ''
  loadMenus()
  modalVisible.value = true
  // 清除上次校验残留
  formRef.value?.clearValidate?.()
}

// 删除
const handleDelete = async (record: Role) => {
  try {
    await roleApi.deleteRole(record.id)
    message.success('删除成功')
    loadData()
  } catch { /* interceptor handles error */ }
}

// 弹窗确认
const handleModalOk = async () => {
  if (viewOnly.value) {
    modalVisible.value = false
    return
  }
  try {
    await formRef.value.validate()
    modalLoading.value = true
    // 后端只认 menu_ids（数字），树上的 code 必须先换成菜单 ID，否则权限根本没保存
    const menuIds = selectedMenus.value
      .map(code => menuIdByCode.value[code])
      .filter((id): id is number => typeof id === 'number')
    const payload = { ...formState, menu_ids: menuIds }
    if (isEdit.value) {
      await roleApi.updateRole(currentId.value, payload)
      message.success('更新成功')
    } else {
      await roleApi.createRole(payload)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadData()
  } catch (e) { console.error(e) }
  finally { modalLoading.value = false }
}
const handleModalCancel = () => { modalVisible.value = false; formRef.value?.resetFields() }

onMounted(() => {
  if (route.query.search) searchForm.search = route.query.search as string
  loadMenuOptions()
  loadData()
})

// 组件在同路由下复用（如从用户列表点另一个角色标签），query 变化不会重跑 onMounted，
// 必须以 URL 为唯一事实源跟随变化，否则 URL 换了数据还是旧的
watch(() => route.query.search, (val) => {
  searchForm.search = (val as string) || ''
  pagination.current = 1
  loadData()
})
</script>

<style scoped>
.page-container { padding: 24px; }
.search-card { margin-bottom: 16px; }
.table-card { margin-bottom: 16px; }

/* 权限数量：数字做成标签，悬浮列出该角色绑定的菜单 */
.count-tag {
  cursor: default;
  margin-inline-end: 0;
}

.menus-tip {
  max-height: 260px;
  overflow-y: auto;
  font-size: 12px;
  line-height: 1.9;
}

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

.menu-node {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-right: 12px;
}

.menu-name {
  font-weight: 500;
  font-size: 13px;
}

.menu-path {
  color: #999;
  font-family: monospace;
  font-size: 11px;
}

:deep(.ant-tree-node-content-wrapper) {
  flex: 1;
  width: 0;
}
</style>
