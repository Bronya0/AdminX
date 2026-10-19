<template>
  <div class="page-container">
    <a-card class="table-card">
      <div class="panel-head">
        <span class="panel-title">菜单权限</span>
        <span class="panel-desc">维护菜单层级、路由路径与权限标识</span>
      </div>

      <TableToolbar>
        <a-button type="primary" @click="handleAdd"><PlusOutlined /> 新增菜单</a-button>
        <a-button @click="expandAllMenuTree"><ExpandOutlined /> 展开全部</a-button>
        <a-button @click="collapseAllMenuTree"><CompressOutlined /> 收起全部</a-button>
        <template #right>
          <a-input-search v-model:value="menuSearchText" placeholder="搜索菜单名称" allow-clear style="width: 240px" @change="filterMenuTree" />
        </template>
      </TableToolbar>

      <!-- 树形表格：展开三角、缩进、选中态、hover、对齐全由组件负责，
           不自绘行结构（自绘会漏掉组件自带的交互）。 -->
      <a-table
        v-if="filteredMenuTreeData.length"
        v-model:expandedRowKeys="expandedRowKeys"
        :columns="columns"
        :data-source="filteredMenuTreeData"
        row-key="key"
        :pagination="false"
        size="middle"
        :custom-row="customRow"
        :row-class-name="rowClassName"
        :scroll="{ x: 'max-content' }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <span class="menu-name" :class="{ 'is-root': record.depth <= 1 }">{{ record.title }}</span>
          </template>
          <template v-else-if="column.key === 'path'">
            <span class="cell-mono">{{ record.path || '—' }}</span>
          </template>
          <template v-else-if="column.key === 'code'">
            <span class="cell-mono">{{ record.code || '—' }}</span>
          </template>
          <template v-else-if="column.key === 'allowed_paths'">
            <a-tooltip v-if="countPaths(record.allowed_paths) > 0" placement="left">
              <template #title>
                <div class="paths-tip">
                  <div v-for="p in pathsList(record.allowed_paths)" :key="p">{{ p }}</div>
                </div>
              </template>
              <a-tag :bordered="false" class="paths-tag">{{ countPaths(record.allowed_paths) }}</a-tag>
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'" :bordered="false">{{ record.is_active ? '启用' : '禁用' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click.stop="handleAddChild(record.key)">添加子菜单</a-button>
              <a-button type="link" size="small" @click.stop="handleEdit(record.key)">编辑</a-button>
              <a-popconfirm title="确定要删除？" @confirm="handleDelete(record.key)">
                <a-button type="link" danger size="small" @click.stop>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
      <a-empty v-else description="没有匹配的菜单" />
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
            <a-form-item label="上级菜单" name="parent_id">
              <a-tree-select v-model:value="formState.parent_id" :tree-data="parentTreeData" placeholder="新增时空着 = 顶级菜单；编辑时空着 = 保持原上级" tree-default-expand-all />
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
                  <div>方法前缀可选，只支持 GET/POST/PUT/PATCH/DELETE 等；路径必须以 <code style="background:#f5f5f5;padding:0 4px">/</code> 开头，路径参数用 <code style="background:#f5f5f5;padding:0 4px">*</code> 而非 <code style="background:#f5f5f5;padding:0 4px">:id</code></div>
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
import { PlusOutlined, ExpandOutlined, CompressOutlined, QuestionCircleOutlined } from '@ant-design/icons-vue'
import { menuApi } from '@/api/menu'
import { menusToTreeData, flatMenusToTreeByDepth, filterTreeBySearch } from '@/utils/menuTree'
import type { Menu } from '@/types'
import TableToolbar from '@/components/TableToolbar.vue'

// ─── 状态 ───
const loading = ref(false)

// 菜单权限
const allMenus = ref<Menu[]>([])
const menuSearchText = ref('')
const expandedRowKeys = ref<(string | number)[]>([])
const originalMenuTreeData = ref<any[]>([])
const filteredMenuTreeData = ref<any[]>([])
const parentTreeData = ref<any[]>([])

// 当前选中的行（再次点击同一行取消选中）
const selectedKey = ref<string | number | null>(null)

const columns = [
  { title: '菜单名称', dataIndex: 'title', key: 'name' },
  { title: '路由路径', dataIndex: 'path', key: 'path', width: 220 },
  { title: '菜单编码', dataIndex: 'code', key: 'code', width: 200 },
  { title: '接口路径数', dataIndex: 'allowed_paths', key: 'allowed_paths', width: 110, align: 'center' as const },
  { title: '状态', dataIndex: 'is_active', key: 'is_active', width: 90 },
  { title: '操作', key: 'action', width: 230, align: 'right' as const },
]

// 接口路径白名单。线上是字符串数组，历史数据可能是 JSON 字符串，两种都兼容。
const pathsList = (val: unknown): string[] => {
  if (Array.isArray(val)) return val.map(String)
  if (typeof val === 'string' && val) {
    try {
      const arr = JSON.parse(val)
      if (Array.isArray(arr)) return arr.map(String)
    } catch {
      /* 不是 JSON 就按每行一条处理 */
    }
    return val.split('\n').map(s => s.trim()).filter(Boolean)
  }
  return []
}

const countPaths = (val: unknown): number => pathsList(val).length

// 点击行 = 选中该行；操作列的按钮各自 stop 冒泡，不会误触
const customRow = (record: any) => ({
  onClick: () => {
    selectedKey.value = selectedKey.value === record.key ? null : record.key
  },
})

// 选中行的底色由 CSS 处理（组件只负责给行加类名）
const rowClassName = (record: any) => (record.key === selectedKey.value ? 'row-selected' : '')

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

// 展开全部：树形表格的展开键 = 所有含子节点的行
const expandAllMenuTree = () => {
  const collect = (nodes: any[]): (string | number)[] => {
    const keys: (string | number)[] = []
    for (const n of nodes) {
      if (n.children && n.children.length) keys.push(n.key)
      if (n.children) keys.push(...collect(n.children))
    }
    return keys
  }
  expandedRowKeys.value = collect(filteredMenuTreeData.value)
}
const collapseAllMenuTree = () => { expandedRowKeys.value = [] }
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
// 方法前缀白名单，与后端 internal/rbacrule 的 methods 表保持一致
const PATH_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']

// checkAllowedPathLine 返回单行规则的错误原因，合法时返回空串。
// 只拦能静态判定必然失效的写法：方法名拼错、冒号写成空格、路径不以 / 开头。
function checkAllowedPathLine(line: string): string {
  if (/\s/.test(line)) return '不能含空格（方法与路径之间用冒号连接，如 GET:/api/v1/x/*）'
  const idx = line.indexOf(':')
  if (idx >= 0) {
    const method = line.slice(0, idx).toUpperCase()
    if (!PATH_METHODS.includes(method)) {
      return '方法前缀无效（只支持 GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS；路径参数请用 * 通配，不要写 :id）'
    }
    return line.slice(idx + 1).startsWith('/') ? '' : '路径部分必须以 / 开头'
  }
  return line.startsWith('/') ? '' : '路径必须以 / 开头'
}

// 表单校验：逐行报错并带上行号，定位比一句笼统提示有用。
// 必须声明在 formRules 之前：const 不提升，写在后面 setup 初始化时就会报错。
const validateAllowedPathLines = async (_rule: unknown, value: string) => {
  const lines = (value || '').split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = (lines[i] || '').trim()
    if (!line) continue
    const reason = checkAllowedPathLine(line)
    if (reason) throw new Error(`第 ${i + 1} 行「${line}」：${reason}`)
  }
}

const formState = reactive({ name: '', code: '', icon: '', path: '', allowed_paths: '', is_active: true, sort_order: 0, parent_id: undefined as number | undefined })
const formRules = {
  name: [{ required: true, message: '请输入菜单名称' }],
  code: [{ required: true, message: '请输入菜单编码' }],
  // 规则判定与后端 internal/rbacrule 一致：写错的规则后端不会报错、只是永不匹配
  // （菜单配了权限、用户仍然 403），所以提交前就在这里拦住。
  allowed_paths: [{ validator: validateAllowedPathLines, trigger: 'blur' }],
}
const resetForm = () => { Object.assign(formState, { name: '', code: '', icon: '', path: '', allowed_paths: '', is_active: true, sort_order: 0, parent_id: undefined }) }

// allowed_paths 的线上格式是字符串数组（对应后端 jsonb 列 + RBAC 的 []string），
// 不是 JSON 字符串：发 JSON 字符串会被后端拒绑
// （cannot unmarshal string into Go struct field ... of type []string）。
function pathsToLines(val: unknown): string {
  if (Array.isArray(val)) return val.join('\n')
  // 容忍历史数据/旧接口里的 JSON 字符串形态
  if (typeof val === 'string' && val) {
    try {
      const arr = JSON.parse(val)
      return Array.isArray(arr) ? arr.join('\n') : val
    } catch { return val }
  }
  return ''
}
function linesToPaths(val: string): string[] {
  return val.split('\n').map(s => s.trim()).filter(Boolean)
}

const findNode = (nodes: any[], key: string): any | null => {
  for (const n of nodes) {
    if (n.key === key) return n
    if (n.children) { const f = findNode(n.children, key); if (f) return f }
  }
  return null
}

const handleAdd = () => { isEdit.value = false; modalTitle.value = '新增菜单'; currentId.value = ''; resetForm(); modalVisible.value = true }
const handleAddChild = (parentKey: number) => { isEdit.value = false; modalTitle.value = '添加子菜单'; currentId.value = ''; resetForm(); formState.parent_id = parentKey; selectedKey.value = parentKey; modalVisible.value = true }
const handleEdit = (key: string) => {
  const node = findNode(filteredMenuTreeData.value, key)
  if (!node) return
  isEdit.value = true; modalTitle.value = '编辑菜单'; currentId.value = key
  selectedKey.value = key
  Object.assign(formState, { name: node.title, code: node.code, icon: node.icon || '', path: node.path || '', allowed_paths: pathsToLines(node.allowed_paths), is_active: node.is_active ?? true, sort_order: node.sort_order ?? 0, parent_id: undefined })
  modalVisible.value = true
}
const handleDelete = async (key: string) => { try { await menuApi.deleteMenu(key); message.success('删除成功'); loadData() } catch { /* interceptor handles error */ } }
const handleModalOk = async () => {
  try {
    await formRef.value.validate()
    modalLoading.value = true
    const data = { ...formState, allowed_paths: linesToPaths(formState.allowed_paths) }
    // 未选上级就不发 parent_id：后端据此判定「不改动父子关系」（nil = 保持）
    if (!data.parent_id) delete data.parent_id
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

.panel-head { display: flex; align-items: baseline; gap: 10px; margin-bottom: 18px; }
.panel-title { font-size: 16px; font-weight: 600; }
.panel-desc { font-size: 12px; color: var(--ant-color-text-secondary); }

/* 一级菜单加粗，与子菜单区分 */
.menu-name.is-root { font-weight: 600; }

/* 路径与编码用等宽字体 + 语义次要色 */
.cell-mono { font-family: monospace; font-size: 12px; color: var(--ant-color-text-secondary); }

/* 数字标签用默认指针，提示悬浮内容才是重点 */
.paths-tag { cursor: default; margin-inline-end: 0; }

/* 悬浮提示里的白名单逐条列出，用等宽字体便于看路径 */
.paths-tip { font-family: monospace; font-size: 12px; line-height: 1.9; }

/* 选中行：AntD 自带的 hover 背景优先级更高，这里必须盖住 */
:deep(.ant-table-tbody > tr.row-selected > td) {
  background: var(--ant-color-primary-bg) !important;
}

:deep(.ant-table-tbody > tr.row-selected > td .menu-name) {
  color: var(--ant-color-primary);
  font-weight: 600;
}

/* 树形表格的展开图标默认偏小，调到与其他内容协调 */
:deep(.ant-table-row-expand-icon) { font-size: 11px; }
</style>
