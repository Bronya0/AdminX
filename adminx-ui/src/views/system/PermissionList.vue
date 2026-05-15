<template>
  <div class="page-container">
    <a-card class="table-card">
      <a-tabs v-model:activeKey="activeTab" type="card">
        <!-- ===== 接口权限 ===== -->
        <a-tab-pane key="api" tab="接口权限">
          <div class="table-toolbar">
            <a-form layout="inline" style="flex-wrap: wrap; gap: 4px;">
              <a-form-item label="名称">
                <a-input v-model:value="searchName" placeholder="权限名称" allow-clear @pressEnter="filterApiPerms" />
              </a-form-item>
              <a-form-item label="编码">
                <a-input v-model:value="searchCodeName" placeholder="权限编码" allow-clear @pressEnter="filterApiPerms" />
              </a-form-item>
              <a-form-item label="模块">
                <a-select v-model:value="moduleFilter" placeholder="全部" allow-clear style="width: 140px" @change="filterApiPerms">
                  <a-select-option v-for="m in moduleOptions" :key="m" :value="m">{{ m }}</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item>
                <a-button type="primary" @click="filterApiPerms"><SearchOutlined /> 搜索</a-button>
              </a-form-item>
            </a-form>
            <span style="color: #999; font-size: 13px;">
              共 {{ filteredApiPerms.length }} 条
            </span>
          </div>

          <a-table
            :columns="apiColumns"
            :data-source="filteredApiPerms"
            :pagination="{ pageSize: 10, showSizeChanger: true, showTotal: (t: number) => `共 ${t} 条` }"
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
              <a-button @click="expandAllMenuTree">
                <ExpandOutlined /> 展开全部
              </a-button>
              <a-button @click="collapseAllMenuTree">
                <CompressOutlined /> 收起全部
              </a-button>
            </div>
            <a-input-search
              v-model:value="menuSearchText"
              placeholder="搜索菜单名称..."
              allow-clear
              style="width: 240px"
              @change="filterMenuTree"
            />
          </div>

          <a-tree
            v-model:expandedKeys="menuExpandedKeys"
            :tree-data="filteredMenuTreeData"
            :loading="loading"
            :draggable="true"
            :block-node="true"
            :show-line="true"
            @drop="handleMenuDrop"
          >
            <template #title="{ key, title, icon, menu_type, is_active, is_visible, code }">
              <div class="tree-node-content">
                <component :is="getIcon(icon)" v-if="icon" class="node-icon" />
                <span class="node-name">{{ title }}</span>
                <span class="node-code">{{ code }}</span>
                <a-tag :color="getTypeColor(menu_type)" size="small">{{ getTypeText(menu_type) }}</a-tag>
                <a-tag v-if="!is_active" color="error" size="small">禁用</a-tag>
                <a-tag v-if="!is_visible" color="default" size="small">隐藏</a-tag>
                <span class="node-actions">
                  <a-button type="link" size="small" @click.stop="handleAddChild(key)">
                    <PlusOutlined />
                  </a-button>
                  <a-button type="link" size="small" @click.stop="handleEdit(key)">
                    <EditOutlined />
                  </a-button>
                  <a-popconfirm title="确定要删除该菜单吗？子菜单也会一并删除" @confirm.stop="handleDelete(key)">
                    <a-button type="link" danger size="small">
                      <DeleteOutlined />
                    </a-button>
                  </a-popconfirm>
                </span>
              </div>
            </template>
          </a-tree>
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
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  SearchOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ExpandOutlined,
  CompressOutlined,
} from '@ant-design/icons-vue'
import { resolveIcon } from '@/utils/iconResolver'
import { menuApi } from '@/api/menu'
import { permissionApi } from '@/api/auth'
import { menusToTreeData, flatMenusToTreeByDepth, filterTreeBySearch } from '@/utils/menuTree'
import type { Menu, Permission } from '@/types'

const getIcon = resolveIcon

// ─── 状态 ───
const activeTab = ref('api')
const loading = ref(false)

// 接口权限
const allPermissions = ref<Permission[]>([])
const searchName = ref('')
const searchCodeName = ref('')
const filteredApiPerms = ref<Permission[]>([])
const moduleFilter = ref<string | undefined>(undefined)
const moduleOptions = ref<string[]>([])

// 菜单权限
const allMenus = ref<Menu[]>([])
const menuSearchText = ref('')
const menuExpandedKeys = ref<string[]>([])
const originalMenuTreeData = ref<any[]>([])
const filteredMenuTreeData = ref<any[]>([])
const parentTreeData = ref<any[]>([])

// ─── 表格列 ───
const apiColumns = [
  { title: '权限编码', key: 'codename', width: 200 },
  { title: '中文名称', dataIndex: 'name', key: 'name' },
  { title: '所属模块', dataIndex: 'content_type_name', key: 'module', width: 180 },
  { title: '类型', key: 'type', width: 80 },
]

// ─── 过滤 ───
const filterApiPerms = () => {
  let list = allPermissions.value
  if (moduleFilter.value) {
    list = list.filter(p => p.content_type_name === moduleFilter.value)
  }
  const name = searchName.value.toLowerCase()
  if (name) {
    list = list.filter(p => p.name.toLowerCase().includes(name))
  }
  const codename = searchCodeName.value.toLowerCase()
  if (codename) {
    list = list.filter(p => p.codename.toLowerCase().includes(codename))
  }
  filteredApiPerms.value = list
}

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
    moduleOptions.value = [...new Set(perms.map(p => p.content_type_name).filter(Boolean))].sort() as string[]
    filterApiPerms()

    // 构建菜单树
    const fullTree = flatMenusToTreeByDepth(menus)
    originalMenuTreeData.value = menusToTreeData(fullTree, 'id')
    filteredMenuTreeData.value = [...originalMenuTreeData.value]
    parentTreeData.value = menusToTreeData(fullTree, 'id')
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

// ─── 菜单树操作 ───
const expandAllMenuTree = () => {
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
const collapseAllMenuTree = () => { menuExpandedKeys.value = [] }

const filterMenuTree = () => {
  filteredMenuTreeData.value = filterTreeBySearch(originalMenuTreeData.value, menuSearchText.value)
  if (menuSearchText.value) expandAllMenuTree()
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

const filterPermOption = (input: string, option: any) =>
  option.value.toLowerCase().includes(input.toLowerCase())

// ─── 从树中查找节点 ───
const findNode = (nodes: any[], key: string): any | null => {
  for (const n of nodes) {
    if (n.key === key) return n
    if (n.children) {
      const found = findNode(n.children, key)
      if (found) return found
    }
  }
  return null
}

// ─── CRUD ───
const handleAdd = () => {
  isEdit.value = false
  modalTitle.value = '新增菜单'
  currentId.value = ''
  resetForm()
  modalVisible.value = true
}

const handleAddChild = (parentKey: string) => {
  isEdit.value = false
  modalTitle.value = '添加子菜单'
  currentId.value = ''
  resetForm()
  formState.parent = parentKey
  modalVisible.value = true
}

const handleEdit = (key: string) => {
  const node = findNode(filteredMenuTreeData.value, key)
  if (!node) return

  isEdit.value = true
  modalTitle.value = '编辑菜单'
  currentId.value = key
  Object.assign(formState, {
    name: node.title,
    code: node.code,
    icon: node.icon || '',
    path: node.path || '',
    component: node.component || '',
    permission_code: node.permission_code || '',
    menu_type: node.menu_type || 'menu',
    is_active: node.is_active ?? true,
    is_visible: node.is_visible ?? true,
    sort_order: node.sort_order ?? 0,
    parent: undefined,
  })
  modalVisible.value = true
}

const handleDelete = async (key: string) => {
  try {
    await menuApi.deleteMenu(key)
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

// ─── 拖拽移动 ───
const handleMenuDrop = async (info: any) => {
  const { dragNode, node, dropPosition, dropToGap } = info
  let position: 'first-child' | 'left' | 'right'
  if (dropToGap) {
    position = dropPosition === -1 ? 'left' : 'right'
  } else {
    position = 'first-child'
  }
  try {
    await menuApi.moveMenu({ id: dragNode.key, target_id: node.key, position })
    message.success('移动成功')
    loadData()
  } catch {
    message.error('移动失败')
  }
}

// ─── 初始化 ───
onMounted(loadData)
</script>

<style scoped>
.page-container { padding: 24px; }
.table-card { margin-bottom: 16px; }
.table-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 8px;
}
.table-toolbar-left {
  display: flex;
  gap: 8px;
}

.tree-node-content {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 0;
  width: 100%;
}
.node-icon {
  font-size: 14px;
  color: #1890ff;
  flex-shrink: 0;
}
.node-name {
  font-weight: 500;
  font-size: 14px;
  min-width: 80px;
}
.node-code {
  font-size: 12px;
  color: #999;
  min-width: 100px;
}
.node-actions {
  margin-left: auto;
  flex-shrink: 0;
  visibility: hidden;
  white-space: nowrap;
}
:deep(.ant-tree-treenode:hover) .node-actions {
  visibility: visible;
}
:deep(.ant-tree-node-content-wrapper) {
  overflow: hidden;
}
</style>
