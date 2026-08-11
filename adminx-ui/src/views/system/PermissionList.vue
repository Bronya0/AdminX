<template>
  <div class="page-container">
    <a-card class="table-card">
      <a-tabs v-model:activeKey="activeTab" type="card">
        <!-- ===== 菜单权限 ===== -->
        <a-tab-pane key="menu" tab="菜单权限">
          <div class="table-toolbar">
            <div class="table-toolbar-left">
              <a-button type="primary" @click="handleAdd"><PlusOutlined /> 新增菜单</a-button>
              <a-button @click="expandAllMenuTree"><ExpandOutlined /> 展开全部</a-button>
              <a-button @click="collapseAllMenuTree"><CompressOutlined /> 收起全部</a-button>
              <a-input-search v-model:value="menuSearchText" placeholder="搜索菜单名称..." allow-clear style="width: 240px; margin-left: 16px;" @change="filterMenuTree" />
            </div>
          </div>
          <a-tree v-model:expandedKeys="menuExpandedKeys" :tree-data="filteredMenuTreeData" :loading="loading" :block-node="true">
            <template #title="{ key, title, is_active, path, permission_code }">
              <div class="tree-node-content">
                <span class="node-name">{{ title }}</span>
                <span v-if="path" class="node-path">{{ path }}</span>
                <a-tag v-if="!is_active" color="error" size="small">禁用</a-tag>
                <span class="node-actions">
                  <a-button type="link" size="small" @click.stop="handleAddChild(key)"><PlusOutlined /></a-button>
                  <a-button type="link" size="small" @click.stop="handleEdit(key)"><EditOutlined /></a-button>
                  <a-popconfirm title="确定要删除？" @confirm.stop="handleDelete(key)">
                    <a-button type="link" danger size="small"><DeleteOutlined /></a-button>
                  </a-popconfirm>
                </span>
              </div>
            </template>
          </a-tree>
        </a-tab-pane>
      </a-tabs>
    </a-card>

    <!-- 菜单编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="modalTitle" :confirm-loading="modalLoading" @ok="handleModalOk" @cancel="handleModalCancel" width="700px">
      <a-form ref="formRef" :model="formState" :rules="formRules" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="菜单名称" name="name"><a-input v-model:value="formState.name" placeholder="请输入菜单名称" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="菜单编码" name="code"><a-input v-model:value="formState.code" placeholder="请输入菜单编码" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="图标" name="icon"><a-input v-model:value="formState.icon" placeholder="如: SettingOutlined" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="排序" name="sort_order"><a-input-number v-model:value="formState.sort_order" :min="0" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="路由路径" name="path"><a-input v-model:value="formState.path" placeholder="如: /system/users" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="上级菜单" name="parent">
              <a-tree-select v-model:value="formState.parent" :tree-data="parentTreeData" placeholder="不选则为顶级菜单" allow-clear tree-default-expand-all />
            </a-form-item>
          </a-col>
          <a-col :span="12"><a-form-item label="启用"><a-switch v-model:checked="formState.is_active" checked-children="是" un-checked-children="否" /></a-form-item></a-col>
        </a-row>
        <a-form-item name="allowed_paths">
          <template #label>
            接口路径白名单
            <a-tooltip placement="right" color="white">
              <template #title>
                <div style="font-size:12px;line-height:2;color:#333;max-width:360px">
                  <div><code style="background:#f5f5f5;padding:0 4px">*</code> 匹配任意字符，<code style="background:#f5f5f5;padding:0 4px">?</code> 匹配单个字符</div>
                  <div style="margin-top:6px;font-weight:500">示例：</div>
                  <div><code style="background:#f5f5f5;padding:0 4px">GET:/api/v1/accounts/users/*</code> 用户列表</div>
                  <div><code style="background:#f5f5f5;padding:0 4px">/api/v1/config/*</code> 配置中心（不限方法）</div>
                  <div><code style="background:#f5f5f5;padding:0 4px">POST:/api/v1/blog/*</code> 博客仅新增</div>
                  <div><code style="background:#f5f5f5;padding:0 4px">/api/v1/monitor/*</code> 整个监控模块</div>
                </div>
              </template>
              <QuestionCircleOutlined style="color:#bbb;margin-left:4px;cursor:help;font-size:14px" />
            </a-tooltip>
          </template>
          <a-textarea v-model:value="formState.allowed_paths" :rows="4" placeholder="每行一条，如：
GET:/api/v1/accounts/users/*
POST:/api/v1/accounts/roles/*
/api/v1/config/*" /></a-form-item>
      </a-form>
    </a-modal>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, ExpandOutlined, CompressOutlined, QuestionCircleOutlined } from '@ant-design/icons-vue'
import { menuApi } from '@/api/menu'
import { menusToTreeData, flatMenusToTreeByDepth, filterTreeBySearch } from '@/utils/menuTree'
import type { Menu } from '@/types'

// ─── 状态 ───
const activeTab = ref('menu')
const loading = ref(false)

// 菜单权限
const allMenus = ref<Menu[]>([])
const menuSearchText = ref('')
const menuExpandedKeys = ref<(string | number)[]>([])
const originalMenuTreeData = ref<any[]>([])
const filteredMenuTreeData = ref<any[]>([])
const parentTreeData = ref<any[]>([])

const loadData = async () => {
  loading.value = true
  try {
    const menus = await menuApi.getMenus()
    allMenus.value = menus
    const fullTree = flatMenusToTreeByDepth(menus)
    originalMenuTreeData.value = menusToTreeData(fullTree, 'id')
    filteredMenuTreeData.value = [...originalMenuTreeData.value]
    parentTreeData.value = menusToTreeData(fullTree, 'id')
    expandAllMenuTree()
  } finally { loading.value = false }
}

const expandAllMenuTree = () => {
  const collect = (nodes: any[]): (string | number)[] => {
    const keys: (string | number)[] = []
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

// ─── 菜单弹窗 ───
const modalVisible = ref(false)
const modalLoading = ref(false)
const modalTitle = ref('新增菜单')
const isEdit = ref(false)
const currentId = ref('')
const formRef = ref()
const formState = reactive({ name: '', code: '', icon: '', path: '', allowed_paths: '', is_active: true, sort_order: 0, parent: undefined as number | undefined })
const formRules = { name: [{ required: true, message: '请输入菜单名称' }], code: [{ required: true, message: '请输入菜单编码' }] }
const resetForm = () => { Object.assign(formState, { name: '', code: '', icon: '', path: '', allowed_paths: '', is_active: true, sort_order: 0, parent: undefined }) }

function jsonPathsToLines(val: string): string {
  try {
    const arr = JSON.parse(val)
    return Array.isArray(arr) ? arr.join('\n') : val
  } catch { return val }
}
function linesToJsonPaths(val: string): string {
  const arr = val.split('\n').map(s => s.trim()).filter(Boolean)
  return JSON.stringify(arr)
}

const findNode = (nodes: any[], key: string): any | null => {
  for (const n of nodes) {
    if (n.key === key) return n
    if (n.children) { const f = findNode(n.children, key); if (f) return f }
  }
  return null
}

const handleAdd = () => { isEdit.value = false; modalTitle.value = '新增菜单'; currentId.value = ''; resetForm(); modalVisible.value = true }
const handleAddChild = (parentKey: number) => { isEdit.value = false; modalTitle.value = '添加子菜单'; currentId.value = ''; resetForm(); formState.parent = parentKey; modalVisible.value = true }
const handleEdit = (key: string) => {
  const node = findNode(filteredMenuTreeData.value, key)
  if (!node) return
  isEdit.value = true; modalTitle.value = '编辑菜单'; currentId.value = key
  Object.assign(formState, { name: node.title, code: node.code, icon: node.icon || '', path: node.path || '', allowed_paths: jsonPathsToLines(node.allowed_paths || '[]'), is_active: node.is_active ?? true, sort_order: node.sort_order ?? 0, parent: undefined })
  modalVisible.value = true
}
const handleDelete = async (key: string) => { try { await menuApi.deleteMenu(key); message.success('删除成功'); loadData() } catch { /* interceptor handles error */ } }
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true
    const data = { ...formState, allowed_paths: linesToJsonPaths(formState.allowed_paths) }
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
  } catch (e: any) {
    if (e?.errorFields) return /* 表单校验失败，已由 AntD 提示 */
    console.error(e)
  }
  finally { modalLoading.value = false }
}
const handleModalCancel = () => { modalVisible.value = false; formRef.value?.resetFields() }

onMounted(() => { loadData() })
</script>

<style scoped>
.page-container { padding: 24px; }
.table-card { margin-bottom: 16px; }
.table-toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.table-toolbar-left { display: flex; gap: 8px; }
.tree-node-content { display: flex; align-items: center; gap: 8px; padding: 2px 0; width: 100%; }
.node-icon { font-size: 14px; color: #1890ff; flex-shrink: 0; }
.node-name { font-weight: 500; font-size: 14px; min-width: 80px; }
.node-path { color: #999; font-family: monospace; font-size: 11px; margin-right: auto; }
.node-code { font-size: 12px; color: #999; min-width: 100px; }
.node-actions { flex-shrink: 0; white-space: nowrap; }
:deep(.ant-tree-node-content-wrapper) { overflow: hidden; flex: 1; width: 0; }
</style>
