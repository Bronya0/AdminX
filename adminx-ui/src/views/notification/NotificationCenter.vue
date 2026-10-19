<template>
  <div class="page-container">
    <a-card>
      <template #title>
        <span><BellOutlined /> 通知中心</span>
      </template>
      <template #extra>
        <a-button size="small" style="margin-right: 8px;" @click="openWhModal">
          <ApiOutlined /> Webhook 配置
        </a-button>
        <a-button size="small" @click="handleMarkAllRead" v-if="unreadCount > 0">
          全部标记已读
        </a-button>
      </template>

      <div style="margin-bottom: 16px">
        <SearchBar>
          <!-- 后端只支持 unread 一个过滤条件：标题/类型筛不了，就不放控件（放了也是静默无效） -->
          <SearchField label="只看未读">
            <a-switch v-model:checked="unreadOnly" @change="page = 1; fetchData()" />
          </SearchField>

          <template #actions>
            <a-button @click="fetchData">
              <ReloadOutlined /> 刷新
            </a-button>
          </template>
        </SearchBar>
      </div>

      <div v-if="loading" style="text-align: center; padding: 40px;">
        <a-spin />
      </div>

      <a-list v-else :data-source="listData" :locale="{ emptyText: '暂无通知' }">
        <template #renderItem="{ item }">
          <a-list-item
            :class="{ 'unread': !item.is_read }"
            style="cursor: pointer; padding: 12px 16px;"
            @click="handleMarkRead(item)"
          >
            <a-list-item-meta>
              <template #avatar>
                <a-tag :color="tagColor(item.notification_type)" style="margin: 0;">
                  {{ typeLabel(item.notification_type) }}
                </a-tag>
              </template>
              <template #title>
                <span style="display: inline-flex; align-items: center; gap: 8px;">
                  <span
                    v-if="!item.is_read"
                    style="display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: #1890ff; flex-shrink: 0;"
                  />
                  <span
                    v-else
                    style="display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: #d9d9d9; flex-shrink: 0;"
                  />
                  <span :style="{ fontWeight: item.is_read ? 'normal' : 'bold' }">
                    {{ item.title }}
                  </span>
                </span>
              </template>
              <template #description>
                <div style="color: #999; font-size: 12px;">{{ formatRelativeTime(item.created_at) }}</div>
                <div v-if="item.content" style="margin-top: 4px; color: #666;">{{ item.content }}</div>
              </template>
            </a-list-item-meta>
            <template #actions>
              <a-button
                v-if="!item.is_read"
                type="link"
                size="small"
                @click.stop="handleMarkRead(item)"
              >
                标记已读
              </a-button>
              <span v-else style="color: #bfbfbf; font-size: 12px;">已读</span>
            </template>
          </a-list-item>
        </template>
      </a-list>

      <a-pagination
        v-if="total > 0"
        v-model:current="page"
        v-model:pageSize="pageSize"
        :total="total"
        showSizeChanger
        :showTotal="(t: number) => `共 ${t} 条`"
        style="margin-top: 16px; text-align: right;"
        @change="onPageChange"
      />
    </a-card>

    <!-- Webhook 配置弹窗 -->
    <a-modal
      v-model:open="whModalVisible"
      title="Webhook 配置"
      :footer="null"
      width="800px"
      @cancel="closeWhModal"
    >
      <div class="wh-toolbar">
        <a-button type="primary" size="small" @click="whAdd"><PlusOutlined /> 新增</a-button>
        <a-button size="small" @click="whRefresh"><ReloadOutlined /> 刷新</a-button>
      </div>

      <a-table
        :data-source="whList"
        :columns="whColumns"
        :loading="whLoading"
        :pagination="false"
        row-key="id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'events'">
            <a-tag v-for="e in (record.events || '').split(',').filter(Boolean)" :key="e">{{ e }}</a-tag>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'success' : 'error'">{{ record.is_active ? '启用' : '禁用' }}</a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="whTest(record)"><PlayCircleOutlined /></a-button>
              <a-button type="link" size="small" @click="whEdit(record)"><EditOutlined /></a-button>
              <a-popconfirm title="确定删除？" @confirm="whDelete(record)">
                <a-button type="link" danger size="small"><DeleteOutlined /></a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>

      <!-- 新增/编辑子表单 -->
      <a-divider v-if="whFormVisible" />
      <a-form v-if="whFormVisible" ref="whFormRef" :model="whForm" :rules="whRules" layout="vertical">
        <a-form-item label="名称" name="name">
          <a-input v-model:value="whForm.name" placeholder="Webhook 名称" />
        </a-form-item>
        <a-form-item label="URL" name="url">
          <a-input v-model:value="whForm.url" placeholder="https://hooks.example.com/alert" />
        </a-form-item>
        <a-form-item label="签名密钥">
          <a-input-password v-model:value="whForm.secret" placeholder="HMAC-SHA256 密钥（留空表示不修改）" />
        </a-form-item>
        <a-form-item label="触发事件">
          <a-select v-model:value="whForm.events" mode="multiple" placeholder="选择触发事件">
            <a-select-option value="info">信息</a-select-option>
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="warning">警告</a-select-option>
            <a-select-option value="error">错误</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="whForm.is_active" checked-children="是" un-checked-children="否" />
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" :loading="whSaving" @click="whSave">{{ whEditingId ? '更新' : '保存' }}</a-button>
            <a-button @click="whFormVisible = false">取消</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { BellOutlined, ReloadOutlined, ApiOutlined, PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons-vue'
import { notificationApi } from '@/api/notification'
import { webhookApi } from '@/api/webhook'
import { formatRelativeTime } from '@/utils/format'
import type { Notification, WebhookConfig } from '@/types'
import { message } from 'ant-design-vue'
import SearchBar from '@/components/SearchBar.vue'
import SearchField from '@/components/SearchField.vue'

// ── 通知列表 ──
const listData = ref<Notification[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const unreadCount = ref(0)
const unreadOnly = ref(false)

const tagColor = (type: string) =>
  ({ info: 'blue', success: 'green', warning: 'orange', error: 'red' })[type] || 'default'

const typeLabel = (type: string) =>
  ({ info: '信息', success: '成功', warning: '警告', error: '错误' })[type] || type

const fetchData = async () => {
  loading.value = true
  try {
    // 列表与未读数无依赖，并行取（串行会让每次翻页多等一倍 RTT）
    const [res, cnt] = await Promise.all([
      notificationApi.list({
        page: page.value,
        size: pageSize.value,
        unread: unreadOnly.value ? true : undefined,
      }),
      notificationApi.unreadCount(),
    ])
    listData.value = res.results
    total.value = res.count
    unreadCount.value = cnt.count
  } finally {
    loading.value = false
  }
}

// 分页 change 回调：AntD 签名 (page, pageSize)。
// v-model 已同步 page/pageSize ref；切换 pageSize 时强制回到第 1 页，
// 否则停留在旧页码可能超出新的总页数。
const onPageChange = (p: number, ps: number) => {
  if (ps !== pageSize.value) {
    pageSize.value = ps
    page.value = 1
  } else {
    page.value = p
  }
  fetchData()
}

const handleMarkRead = async (item: Notification) => {
  if (item.is_read) return
  // 乐观更新，失败回滚并恢复未读数（否则 UI 显示已读、后端仍是未读，刷新后“复活”）
  item.is_read = true
  unreadCount.value = Math.max(0, unreadCount.value - 1)
  try {
    await notificationApi.markRead(item.id)
  } catch {
    item.is_read = false
    unreadCount.value += 1
  }
}

const handleMarkAllRead = async () => {
  // 快照用于失败回滚
  const prevRead = listData.value.map((item) => item.is_read ?? false)
  const prevUnread = unreadCount.value
  unreadCount.value = 0
  listData.value.forEach(item => { item.is_read = true })
  try {
    await notificationApi.markAllRead()
    // 同步更新头部 badge（重新查询后端确保一致性）
    try {
      const cnt = await notificationApi.unreadCount()
      unreadCount.value = cnt.count
    } catch { /* ignore */ }
    message.success('已全部标记已读')
  } catch {
    listData.value.forEach((item, i) => { item.is_read = prevRead[i] ?? false })
    unreadCount.value = prevUnread
  }
}

// ── Webhook 配置 ──
const whModalVisible = ref(false)
const whList = ref<WebhookConfig[]>([])
const whLoading = ref(false)
const whFormVisible = ref(false)
const whSaving = ref(false)
const whEditingId = ref<string | null>(null)
const whFormRef = ref()

const whColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 140 },
  { title: 'URL', dataIndex: 'url', key: 'url', ellipsis: true },
  { title: '事件', key: 'events', width: 180 },
  { title: '状态', key: 'is_active', width: 80 },
  { title: '操作', key: 'action', width: 140 },
]

const whForm = ref({
  name: '',
  url: '',
  secret: '',
  events: [] as string[],
  is_active: true,
})

const whRules = {
  name: [{ required: true, message: '请输入名称' }],
  url: [{ required: true, message: '请输入 URL' }],
}

const loadWh = async () => {
  whLoading.value = true
  try {
    const res = await webhookApi.getList()
    whList.value = res.results
  } finally {
    whLoading.value = false
  }
}

// 打开弹窗即拉一次列表：只在 onMounted 加载会让首次打开永远是空表，
// 用户必须先点一次“刷新”才能看到已有的 webhook。
const openWhModal = () => {
  whModalVisible.value = true
  whFormVisible.value = false
  loadWh()
}

const closeWhModal = () => {
  whFormVisible.value = false
}

const whRefresh = () => { loadWh() }

const whAdd = () => {
  whEditingId.value = null
  whForm.value = { name: '', url: '', secret: '', events: [], is_active: true }
  whFormVisible.value = true
}

const whEdit = (record: WebhookConfig) => {
  whEditingId.value = record.id
  whForm.value = {
    name: record.name,
    url: record.url,
    secret: '',
    events: (record.events || '').split(',').filter(Boolean),
    is_active: record.is_active,
  }
  whFormVisible.value = true
}

const whDelete = async (record: WebhookConfig) => {
  try {
    await webhookApi.delete(record.id)
    message.success('删除成功')
    loadWh()
  } catch { /* interceptor handles error */ }
}

const whTest = async (record: WebhookConfig) => {
  try {
    const res = await webhookApi.test(record.id)
    // 后端返回的是真实投递结果：对方 4xx/5xx、URL 不通都算失败，不能一律提示成功
    if (res.ok) {
      message.success(`测试消息已发送（HTTP ${res.status_code}）`)
    } else {
      message.error(`测试发送失败：${res.message || '未知原因'}`)
    }
  } catch { /* interceptor handles error */ }
}

const whSave = async () => {
  try {
    await whFormRef.value.validate()
    whSaving.value = true
    const data = {
      ...whForm.value,
      events: whForm.value.events.join(','),
    }
    if (whEditingId.value) {
      await webhookApi.update(whEditingId.value, data)
      message.success('更新成功')
    } else {
      await webhookApi.create(data)
      message.success('创建成功')
    }
    whFormVisible.value = false
    loadWh()
  } catch {
    // validation
  } finally {
    whSaving.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-container { padding: 24px; }
.list-item.unread { background: #f0f5ff; }
:deep(.ant-list-item:hover) { background: #fafafa; }
.wh-toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
