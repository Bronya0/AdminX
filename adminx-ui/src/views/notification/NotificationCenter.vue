<template>
  <div class="page-container">
    <a-card>
      <template #title>
        <span><BellOutlined /> 通知中心</span>
      </template>
      <template #extra>
        <a-badge :count="unreadCount">
          <a-button size="small" @click="fetchData">
            <ReloadOutlined /> 刷新
          </a-button>
        </a-badge>
        <a-button size="small" style="margin-left: 8px;" @click="handleMarkAllRead" v-if="unreadCount > 0">
          全部标记已读
        </a-button>
      </template>

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
                <span :style="{ fontWeight: item.is_read ? 'normal' : 'bold' }">
                  {{ item.title }}
                </span>
              </template>
              <template #description>
                <div style="color: #999; font-size: 12px;">{{ item.created_at }}</div>
                <div v-if="item.content" style="margin-top: 4px; color: #666;">{{ item.content }}</div>
              </template>
            </a-list-item-meta>
          </a-list-item>
        </template>
      </a-list>

      <a-pagination
        v-if="total > 0"
        v-model:current="page"
        :total="total"
        :pageSize="pageSize"
        showSizeChanger
        show-total
        style="margin-top: 16px; text-align: right;"
        @change="fetchData"
      />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { BellOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { notificationApi } from '@/api/notification'
import type { Notification } from '@/types'
import { message } from 'ant-design-vue'

const listData = ref<Notification[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const unreadCount = ref(0)

const tagColor = (type: string) =>
  ({ info: 'blue', success: 'green', warning: 'orange', error: 'red' })[type] || 'default'

const typeLabel = (type: string) =>
  ({ info: '信息', success: '成功', warning: '警告', error: '错误' })[type] || type

const fetchData = async () => {
  loading.value = true
  try {
    const res = await notificationApi.list({ page: page.value, size: pageSize.value })
    listData.value = res.results
    total.value = res.count
    // 同时刷新未读数量
    const cnt = await notificationApi.unreadCount()
    unreadCount.value = cnt.count
  } finally {
    loading.value = false
  }
}

const handleMarkRead = async (item: Notification) => {
  if (item.is_read) return
  await notificationApi.markRead(item.id)
  item.is_read = true
  unreadCount.value = Math.max(0, unreadCount.value - 1)
}

const handleMarkAllRead = async () => {
  await notificationApi.markAllRead()
  unreadCount.value = 0
  listData.value.forEach(item => { item.is_read = true })
  message.success('已全部标记已读')
}

onMounted(fetchData)
</script>

<style scoped>
.list-item.unread {
  background: #f0f5ff;
}
:deep(.ant-list-item:hover) {
  background: #fafafa;
}
</style>
