<template>
  <div class="page-container">
    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline">
        <a-form-item label="搜索">
          <a-input
            v-model:value="searchText"
            placeholder="搜索权限名称或编码"
            allow-clear
            @pressEnter="filterData"
          />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="filterData">
            <SearchOutlined /> 搜索
          </a-button>
          <a-button style="margin-left: 8px" @click="resetSearch">
            <ReloadOutlined /> 重置
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 主体内容 -->
    <a-card class="table-card">
      <a-tabs v-model:activeKey="activeTab" type="card">
        <!-- ===== 接口权限 ===== -->
        <a-tab-pane key="api" tab="接口权限">
          <div class="table-toolbar">
            <span style="color: #999; font-size: 13px;">
              共 {{ filteredApiPerms.length }} 条，由 Django 自动生成，按模型分类
            </span>
          </div>

          <a-table
            :columns="apiColumns"
            :data-source="filteredApiPerms"
            :pagination="{ pageSize: 20, showSizeChanger: true, showTotal: (t: number) => `共 ${t} 条` }"
            row-key="codename"
            size="middle"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'codename'">
                <a-tag>{{ record.codename }}</a-tag>
              </template>
              <template v-if="column.key === 'type'">
                <a-tag color="purple">API</a-tag>
              </template>
            </template>
          </a-table>
        </a-tab-pane>

        <!-- ===== 菜单权限 ===== -->
        <a-tab-pane key="menu" tab="菜单权限">
          <div class="table-toolbar">
            <div class="table-toolbar-left">
              <a-button type="primary" @click="handleAdd">
                <PlusOutlined /> 新增菜单
              </a-button>
            </div>
            <span style="color: #999; font-size: 13px;">
              共 {{ filteredMenuPerms.length }} 条
            </span>
          </div>

          <a-table
            :columns="menuColumns"
            :data-source="filteredMenuPerms"
            :pagination="{ pageSize: 20, showSizeChanger: true, showTotal: (t: number) => `共 ${t} 条` }"
            row-key="id"
            size="middle"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a-space>
                  <component :is="resolveIcon(record.icon)" v-if="record.icon" />
                  <span>{{ record.name }}</span>
                </a-space>
              </template>
              <template v-if="column.key === 'menu_type'">
                <a-tag :color="getTypeColor(record.menu_type)">{{ getTypeText(record.menu_type) }}</a-tag>
              </template>
              <template v-if="column.key === 'permission_code'">
                <a-tag v-if="record.permission_code" color="blue">{{ record.permission_code }}</a-tag>
                <span v-else style="color: #ccc;">-</span>
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
                  <a-popconfirm title="确定删除？子菜单也会一并删除" @confirm="handleDelete(record)">
                    <a-button type="link" danger size="small">
                      <DeleteOutlined /> 删除
                    </a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-card>

    <!-- 菜单编辑弹窗 -->
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
            <a-form-item label="菜单名称" name="name">
              <a-input v-model:value="formState.name" placeholder="请输入菜单名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="菜单编码" name="code">
              <a-input v-model:value="formState.code" placeholder="请输入菜单编码" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="菜单类型" name="menu_type">
              <a-select v-model:value="formState.menu_type" placeholder="请选择菜单类型">
                <a-select-option value="menu">菜单</a-select-option>
                <a-select-option value="button">按钮</a-select-option>
                <a-select-option value="iframe">Iframe</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="关联接口权限" name="permission_code">
              <a-select
                v-model:value="formState.permission_code"
                placeholder="选择关联的权限码（可选）"
                allow-clear
                show-search
                :filter-option="filterPermOption"
              >
                <a-select-option v-for="p in allPermissions" :key="p.codename" :value="p.codename">
                  {{ p.codename }} — {{ p.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="图标" name="icon">
              <a-input v-model:value="formState.icon" placeholder="如: SettingOutlined" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="排序" name="sort_order">
              <a-input-number v-model:value="formState.sort_order" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="路由路径" name="path">
              <a-input v-model:value="formState.path" placeholder="如: /system/users" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="组件路径" name="component">
              <a-input v-model:value="formState.component" placeholder="如: system/user/index" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="上级菜单" name="parent">
              <a-tree-select
                v-model:value="formState.parent"
                :tree-data="parentTreeData"
                placeholder="不选则为顶级菜单"
                allow-clear
                tree-default-expand-all
              />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="启用">
              <a-switch v-model:checked="formState.is_active" checked-children="是" un-checked-children="否" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="可见">
              <a-switch v-model:checked="formState.is_visible" checked-children="是" un-checked-children="否" />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons-vue'
import { resolveIcon } from '@/utils/iconResolver'
import { menuApi } from '@/api/menu'
import { permissionApi } from '@/api/auth'
import { menusToTreeData, flatMenusToTreeByDepth } from '@/utils/menuTree'
import type { Menu, Permission } from '@/types'

// ─── 状态 ───
const activeTab = ref('api')
const searchText = ref('')

// 接口权限
const allPermissions = ref<Permission[]>([])
const filteredApiPerms = ref<Permission[]>([])

// 菜单权限
const allMenus = ref<Menu[]>([])
const filteredMenuPerms = ref<Menu[]>([])
const parentTreeData = ref<any[]>([])
const loading = ref(false)

// ─── 表格列 ───
const apiColumns = [
  { title: '权限编码', key: 'codename', width: 200 },
  { title: '中文名称', dataIndex: 'name', key: 'name' },
  { title: '所属模块', dataIndex: 'content_type_name', key: 'module', width: 180 },
  { title: '类型', key: 'type', width: 80 },
]

const menuColumns = [
  { title: '菜单名称', key: 'name', width: 200 },
  { title: '编码', dataIndex: 'code', key: 'code', width: 180 },
  { title: '路径', dataIndex: 'path', key: 'path', width: 150, ellipsis: true },
  { title: '类型', key: 'menu_type', width: 80 },
  { title: '关联权限', key: 'permission_code', width: 160 },
  { title: '状态', key: 'is_active', width: 80 },
  { title: '操作', key: 'action', width: 140 },
]

// ─── 过滤 ───
const filterData = () => {
  const kw = searchText.value.toLowerCase()
  filteredApiPerms.value = kw
    ? allPermissions.value.filter(p => p.name.includes(kw) || p.codename.includes(kw))
    : allPermissions.value

  filteredMenuPerms.value = kw
    ? allMenus.value.filter(m =>
        m.name.toLowerCase().includes(kw) ||
        m.code.toLowerCase().includes(kw) ||
        (m.path && m.path.toLowerCase().includes(kw)) ||
        (m.permission_code && m.permission_code.toLowerCase().includes(kw))
      )
    : allMenus.value
}

const resetSearch = () => {
  searchText.value = ''
  filterData()
}

const filterPermOption = (input: string, option: any) =>
  option.value.toLowerCase().includes(input.toLowerCase())

// ─── 加载数据 ───
const loadData = async () => {
  loading.value = true
  try {
    const [perms, menus] = await Promise.all([
      permissionApi.getPermissions(),
      menuApi.getMenus(),
    ])
    allPermissions.value = perms
    allMenus.value = menus

    // 构建上级菜单的树选择器数据
    const fullTree = flatMenusToTreeByDepth(menus)
    parentTreeData.value = menusToTreeData(fullTree, 'id')

    filterData()
  } finally {
    loading.value = false
  }
}

// ─── 菜单类型 ───
const getTypeColor = (type: string) => {
  const colors: Record<string, string> = { menu: 'blue', button: 'green', iframe: 'orange' }
  return colors[type] || 'default'
}
const getTypeText = (type: string) => {
  const texts: Record<string, string> = { menu: '菜单', button: '按钮', iframe: 'Iframe' }
  return texts[type] || type
}

// ─── 弹窗状态 ───
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增菜单')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = reactive({
  name: '',
  code: '',
  icon: '',
  path: '',
  component: '',
  permission_code: '',
  menu_type: 'menu' as string,
  is_active: true,
  is_visible: true,
  sort_order: 0,
  parent: undefined as string | undefined,
})

const formRules = {
  name: [{ required: true, message: '请输入菜单名称' }],
  code: [{ required: true, message: '请输入菜单编码' }],
}

const resetForm = () => {
  Object.assign(formState, {
    name: '', code: '', icon: '', path: '', component: '',
    permission_code: '', menu_type: 'menu', is_active: true,
    is_visible: true, sort_order: 0, parent: undefined,
  })
}

// ─── CRUD ───
const handleAdd = () => {
  isEdit.value = false
  modalTitle.value = '新增菜单'
  currentId.value = ''
  resetForm()
  modalVisible.value = true
}

const handleEdit = (record: any) => {
  isEdit.value = true
  modalTitle.value = '编辑菜单'
  currentId.value = record.id
  Object.assign(formState, {
    name: record.name,
    code: record.code,
    icon: record.icon || '',
    path: record.path || '',
    component: record.component || '',
    permission_code: record.permission_code || '',
    menu_type: record.menu_type,
    is_active: record.is_active,
    is_visible: record.is_visible,
    sort_order: record.sort_order || 0,
    parent: record.parent || undefined,
  })
  modalVisible.value = true
}

const handleDelete = async (record: any) => {
  try {
    await menuApi.deleteMenu(record.id)
    message.success('删除成功')
    loadData()
  } catch {
    message.error('删除失败')
  }
}

const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true

    const data: any = { ...formState }
    if (!data.parent) delete data.parent

    if (isEdit.value) {
      await menuApi.updateMenu(currentId.value, data)
      message.success('更新成功')
    } else {
      await menuApi.createMenu(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadData()
  } catch (e) {
    // 校验错误忽略
  } finally {
    modalLoading.value = false
  }
}

const handleModalCancel = () => {
  modalVisible.value = false
  formRef.value?.resetFields()
}

// ─── 初始化 ───
onMounted(loadData)
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
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.table-toolbar-left {
  display: flex;
  gap: 8px;
}
</style>
