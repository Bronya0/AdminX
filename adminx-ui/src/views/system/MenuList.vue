<template>
  <div class="page-container">
    <!-- 操作栏 -->
    <a-card class="tree-card">
      <div class="tree-toolbar">
        <a-space>
          <a-button type="primary" @click="handleAdd">
            <PlusOutlined /> 新增菜单
          </a-button>
          <a-button @click="expandAll">
            <ExpandOutlined /> 展开全部
          </a-button>
          <a-button @click="collapseAll">
            <CompressOutlined /> 收起全部
          </a-button>
        </a-space>
      </div>

      <!-- 菜单树 -->
      <a-tree
        v-model:expandedKeys="expandedKeys"
        :tree-data="treeData"
        :loading="loading"
        :draggable="true"
        :block-node="true"
        :show-line="true"
        @drop="handleDrop"
      >
        <template #title="{ key, title, icon, menu_type, is_active, is_visible, code, ...record }">
          <div class="tree-node-content">
            <component :is="getIcon(icon)" v-if="icon" class="node-icon" />
            <span class="node-name">{{ title }}</span>
            <span class="node-code">{{ code }}</span>
            <a-tag :color="getTypeColor(menu_type)" size="small">{{ getTypeText(menu_type) }}</a-tag>
            <a-tag v-if="!is_active" color="error" size="small">禁用</a-tag>
            <a-tag v-if="!is_visible" color="default" size="small">隐藏</a-tag>
            <span class="node-actions">
              <a-button type="link" size="small" @click.stop="handleAddChild(record)">
                <PlusOutlined />
              </a-button>
              <a-button type="link" size="small" @click.stop="handleEdit(key)">
                <EditOutlined />
              </a-button>
              <a-popconfirm title="确定要删除该菜单吗？" @confirm.stop="handleDelete(key)">
                <a-button type="link" danger size="small">
                  <DeleteOutlined />
                </a-button>
              </a-popconfirm>
            </span>
          </div>
        </template>
      </a-tree>
    </a-card>

    <!-- 新增/编辑弹窗 -->
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
            <a-form-item label="上级菜单" name="parent">
              <a-tree-select
                v-model:value="formState.parent"
                :tree-data="treeSelectData"
                placeholder="请选择上级菜单（不选为顶级菜单）"
                allow-clear
                tree-default-expand-all
              />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="图标" name="icon">
              <a-input v-model:value="formState.icon" placeholder="请输入图标名称，如: HomeOutlined" />
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
              <a-input v-model:value="formState.path" placeholder="请输入路由路径，如: /system/users" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="组件路径" name="component">
              <a-input v-model:value="formState.component" placeholder="请输入组件路径，如: system/UserList" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="权限编码" name="permission_code">
          <a-input v-model:value="formState.permission_code" placeholder="请输入权限编码，如: accounts:user:list" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="状态" name="is_active">
              <a-switch v-model:checked="formState.is_active" checked-children="启用" un-checked-children="禁用" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="是否可见" name="is_visible">
              <a-switch v-model:checked="formState.is_visible" checked-children="可见" un-checked-children="隐藏" />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ExpandOutlined,
  CompressOutlined,
  DashboardOutlined,
  SettingOutlined,
  UserOutlined,
  TeamOutlined,
  MenuOutlined,
  AppstoreOutlined,
  MonitorOutlined,
  ClusterOutlined,
  FileOutlined,
} from '@ant-design/icons-vue'
import { resolveIcon } from '@/utils/iconResolver'
import { menusToTreeData, flatMenusToTreeByDepth } from '@/utils/menuTree'
import { menuApi } from '@/api/menu'

const getIcon = resolveIcon

// 状态
const loading = ref(false)
const treeData = ref<any[]>([])
const treeSelectData = ref<any[]>([])
const expandedKeys = ref<string[]>([])

// 弹窗状态
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增菜单')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()

const formState = ref({
  name: '',
  code: '',
  icon: '',
  path: '',
  component: '',
  permission_code: '',
  menu_type: 'menu',
  is_active: true,
  is_visible: true,
  sort_order: 0,
  parent: undefined as string | undefined,
})

const formRules = {
  name: [{ required: true, message: '请输入菜单名称' }],
  code: [{ required: true, message: '请输入菜单编码' }],
}

// 类型展示
const getTypeColor = (type: string) => {
  const colors: Record<string, string> = { menu: 'blue', button: 'green', iframe: 'orange' }
  return colors[type] || 'default'
}
const getTypeText = (type: string) => {
  const texts: Record<string, string> = { menu: '菜单', button: '按钮', iframe: 'Iframe' }
  return texts[type] || type
}

// 展开/收起
const expandAll = () => {
  const collect = (nodes: any[]): string[] => {
    const keys: string[] = []
    for (const n of nodes) {
      keys.push(n.key)
      if (n.children) keys.push(...collect(n.children))
    }
    return keys
  }
  expandedKeys.value = collect(treeData.value)
}
const collapseAll = () => {
  expandedKeys.value = []
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const flatMenus = await menuApi.getMenus()
    const fullTree = flatMenusToTreeByDepth(flatMenus)
    treeData.value = menusToTreeData(fullTree, 'id')
    treeSelectData.value = menusToTreeData(fullTree, 'id')
    expandAll()
  } finally {
    loading.value = false
  }
}

// 新增
const handleAdd = () => {
  isEdit.value = false
  modalTitle.value = '新增菜单'
  currentId.value = ''
  formState.value = {
    name: '', code: '', icon: '', path: '', component: '',
    permission_code: '', menu_type: 'menu', is_active: true,
    is_visible: true, sort_order: 0, parent: undefined,
  }
  modalVisible.value = true
}

// 添加子菜单
const handleAddChild = (record: any) => {
  isEdit.value = false
  modalTitle.value = '添加子菜单'
  currentId.value = ''
  formState.value = {
    name: '', code: '', icon: '', path: '', component: '',
    permission_code: '', menu_type: 'menu', is_active: true,
    is_visible: true, sort_order: 0, parent: record.key,
  }
  modalVisible.value = true
}

// 编辑 — 从 treeData 中查找节点
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

const handleEdit = (key: string) => {
  const node = findNode(treeData.value, key)
  if (!node) return

  isEdit.value = true
  modalTitle.value = '编辑菜单'
  currentId.value = key
  formState.value = {
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
  }
  modalVisible.value = true
}

// 删除
const handleDelete = async (key: string) => {
  try {
    await menuApi.deleteMenu(key)
    message.success('删除成功')
    loadData()
  } catch {
    message.error('删除失败')
  }
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true

    const data: any = { ...formState.value }
    if (data.parent === undefined) delete data.parent

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

// 拖拽处理
const handleDrop = async (info: any) => {
  const { dragNode, node, dropPosition, dropToGap } = info
  const dragKey = dragNode.key
  const dropKey = node.key

  let position: 'first-child' | 'left' | 'right'
  if (dropToGap) {
    position = dropPosition === -1 ? 'left' : 'right'
  } else {
    position = 'first-child'
  }

  try {
    await menuApi.moveMenu({ id: dragKey, target_id: dropKey, position })
    message.success('移动成功')
    loadData()
  } catch {
    message.error('移动失败')
  }
}

onMounted(loadData)
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.tree-card {
  margin-bottom: 16px;
}

.tree-toolbar {
  margin-bottom: 16px;
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

/* 树节点 hover 时显示操作按钮 */
:deep(.ant-tree-treenode:hover) .node-actions {
  visibility: visible;
}

:deep(.ant-tree-node-content-wrapper) {
  overflow: hidden;
}
</style>