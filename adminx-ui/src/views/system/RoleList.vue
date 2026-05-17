<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="角色名称">
          <a-input
            v-model:value="searchForm.search"
            placeholder="名称"
            allow-clear
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item label="描述">
          <a-input
            v-model:value="searchForm.desc"
            placeholder="描述"
            allow-clear
            @pressEnter="handleSearch"
          />
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
            <PlusOutlined /> 新增角色
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
            <a-form-item label="角色名称" name="name">
              <a-input v-model:value="formState.name" placeholder="请输入角色名称" :disabled="viewOnly" />
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
              checkable :tree-data="filteredMenuTreeData" :default-expand-all="false" :block-node="true"
              :check-strictly="viewOnly">
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
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
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
import type { Role } from '@/types'

const route = useRoute()

// 表格列定义
const columns = [
  { title: '角色名称', dataIndex: 'name', key: 'name' },
  { title: '描述', dataIndex: 'desc', key: 'desc', ellipsis: true },
  { title: '权限数量', key: 'menus_count', width: 100, align: 'center' as const, customRender: ({ record }: any) => record.menus?.length ?? 0 },
  { title: '状态', key: 'is_active' },
  { title: '类型', key: 'is_system', width: 100, align: 'center' as const },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', customRender: ({ text }: any) => formatDateTime(text) },
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
const searchForm = reactive({ search: '', desc: '', is_active: undefined as boolean | undefined })

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
  name: '', desc: '', is_active: true, menus: [] as string[],
})
const formRules = {
  name: [{ required: true, message: '请输入角色名称' }],
}

// 菜单树
const menuSearch = ref('')
const menuExpandedKeys = ref<string[]>([])
const originalMenuTreeData = ref<any[]>([])
const filteredMenuTreeData = ref<any[]>([])
const selectedMenus = ref<string[]>([])

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await roleApi.getRoles({ page: pagination.current, size: pagination.pageSize, search: searchForm.search, desc: searchForm.desc || undefined, is_active: searchForm.is_active })
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
const resetSearch = () => { searchForm.search = ''; searchForm.desc = ''; searchForm.is_active = undefined; handleSearch() }
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
  Object.assign(formState, { name: '', desc: '', is_active: true, menus: [] })
  selectedMenus.value = []
  menuExpandedKeys.value = []
  menuSearch.value = ''
  loadMenus()
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: Role, readOnly = false) => {
  isEdit.value = true
  viewOnly.value = readOnly
  modalTitle.value = readOnly ? `角色详情 - ${record.name}` : '编辑角色'
  currentId.value = record.id
  modalActiveTab.value = 'basic'
  Object.assign(formState, { name: record.name, desc: record.desc, is_active: record.is_active, menus: record.menus || [] })
  selectedMenus.value = record.menus || []
  menuExpandedKeys.value = []
  menuSearch.value = ''
  loadMenus()
  modalVisible.value = true
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
    formState.menus = selectedMenus.value
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

onMounted(() => {
  const search = route.query.search as string | undefined
  if (search) {
    searchForm.search = search
  }
  loadData()
})
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
