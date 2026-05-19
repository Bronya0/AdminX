<template>
  <div>
    <a-tabs v-model:activeKey="activeTab" type="card">
      <!-- ── 系统组件 ── -->
      <a-tab-pane key="system">
        <template #tab>
          <a-space size="small">
            <span>系统组件</span>
            <a-badge :count="systemErrorCount" :number-style="{ backgroundColor: '#ff4d4f' }"
              :overflow-count="9" show-zero />
          </a-space>
        </template>

        <div style="margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center">
          <a-space v-if="sysData?.platform">
            <a-tag>Python {{ sysData.platform.python_version }}</a-tag>
            <a-tag>Django {{ sysData.platform.django_version }}</a-tag>
            <a-tag>{{ sysData.platform.os }}</a-tag>
          </a-space>
          <a-button size="small" @click="loadSys" :loading="sysLoading">
            <ReloadOutlined /> 刷新
          </a-button>
        </div>

        <a-table :data-source="sysComponents" :columns="sysColumns" :loading="sysLoading"
          :pagination="false" size="small" row-key="key">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="sysStatusColor(record.status)">
                {{ sysStatusText(record.status) }}
              </a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <!-- ── 业务组件 ── -->
      <a-tab-pane key="business">
        <template #tab>
          <a-space size="small">
            <span>业务组件</span>
            <a-badge :count="bizComponents.length" :number-style="{ backgroundColor: '#1890ff' }"
              :overflow-count="99" show-zero />
          </a-space>
        </template>

        <div style="margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center">
          <a-space>
            <a-tag color="success">在线 {{ bizOnline }}</a-tag>
            <a-tag color="error">离线 {{ bizComponents.length - bizOnline }}</a-tag>
            <a-tag v-if="pendingCommandCount > 0" color="warning">待指令 {{ pendingCommandCount }}</a-tag>
          </a-space>
          <a-space>
            <a-input-search v-model:value="bizSearch" placeholder="搜索组件名/标识"
              allow-clear style="width: 200px" @search="loadBiz" />
            <a-button size="small" @click="loadBiz" :loading="bizLoading"><ReloadOutlined /></a-button>
          </a-space>
        </div>

        <a-table :data-source="bizComponents" :columns="bizColumns" :loading="bizLoading"
          :pagination="bizPagination" row-key="id" @change="handleTableChange">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === 'online' ? 'success' : 'error'">
                {{ record.status === 'online' ? '在线' : '离线' }}
              </a-tag>
            </template>
            <template v-if="column.key === 'last_heartbeat'">
              {{ record.last_heartbeat ? formatDateTime(record.last_heartbeat) : '—' }}
            </template>
            <template v-if="column.key === 'pending_command'">
              <a-tag v-if="record.pending_command === 'upgrade'" color="orange">
                升级 → {{ record.upgrade_version }}
              </a-tag>
              <a-tag v-else-if="record.pending_command === 'uninstall'" color="red">
                待卸载
              </a-tag>
              <span v-else style="color: #bbb">—</span>
            </template>
            <template v-if="column.key === 'action'">
              <a-space>
                <a-button size="small" type="link" @click="openUpgradeModal(record)"
                  :disabled="record.pending_command === 'uninstall'">升级</a-button>
                <a-button v-if="record.pending_command === 'upgrade'" size="small" type="link"
                  @click="cancelUpgrade(record)">取消升级</a-button>
                <a-popconfirm v-if="record.pending_command !== 'uninstall'"
                  title="确认下发卸载指令？业务服务收到后将自行执行卸载。" @confirm="setUninstall(record)">
                  <a-button size="small" type="link" danger>卸载</a-button>
                </a-popconfirm>
                <a-button v-if="record.pending_command === 'uninstall'" size="small" type="link"
                  @click="cancelUninstall(record)">取消卸载</a-button>
                <a-popconfirm title="确认从平台删除该组件记录？" @confirm="deleteComponent(record)">
                  <a-button size="small" type="link" danger>删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <!-- 升级弹窗 -->
    <a-modal v-model:open="upgradeModalOpen" title="设置升级任务"
      @ok="submitUpgrade" :confirm-loading="upgradeLoading">
      <a-form :model="upgradeForm" layout="vertical" style="margin-top: 8px">
        <a-form-item label="目标版本" required>
          <a-input v-model:value="upgradeForm.upgrade_version" placeholder="如 2.1.0" />
        </a-form-item>
        <a-form-item label="升级包下载地址" required>
          <a-input v-model:value="upgradeForm.upgrade_url"
            placeholder="http://... 或文件中心地址" />
        </a-form-item>
        <a-form-item label="SHA-256 校验值（可选）">
          <a-input v-model:value="upgradeForm.upgrade_checksum"
            placeholder="留空则跳过完整性校验" />
        </a-form-item>
        <a-alert type="info" show-icon
          message="业务服务在下次心跳（≤30s）时收到升级指令，由业务服务自行完成升级。" />
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { componentApi } from '@/api/cluster'
import { formatDateTime } from '@/utils/format'
import type { ServiceComponent, SystemComponentsData } from '@/types'

const activeTab = ref('system')

// ── 系统组件 ──
const sysLoading = ref(true)
const sysData = ref<SystemComponentsData | null>(null)
const sysComponents = computed(() => sysData.value?.components ?? [])
const systemErrorCount = computed(() =>
  sysComponents.value.filter(c => c.status !== 'ok').length
)

const sysStatusColor = (status: string) => {
  if (status === 'ok') return 'success'
  if (status === 'offline') return 'default'
  return 'error'
}
const sysStatusText = (status: string) => {
  if (status === 'ok') return '正常'
  if (status === 'offline') return '未运行'
  return '异常'
}

const sysColumns = [
  { title: '组件', dataIndex: 'name', key: 'name', width: 120 },
  { title: '状态', key: 'status', width: 100 },
  { title: '说明', dataIndex: 'message', key: 'message' },
]

// ── 业务组件 ──
const bizLoading = ref(false)
const bizSearch = ref('')
const bizComponents = ref<ServiceComponent[]>([])
const bizPagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true })

const bizOnline = computed(() => bizComponents.value.filter(c => c.status === 'online').length)
const pendingCommandCount = computed(() => bizComponents.value.filter(c => c.pending_command).length)

const bizColumns = [
  { title: '组件名称', dataIndex: 'name', key: 'name' },
  { title: '标识', dataIndex: 'app_label', key: 'app_label' },
  { title: '版本', dataIndex: 'version', key: 'version', width: 100 },
  { title: '服务地址', dataIndex: 'host', key: 'host' },
  { title: '状态', key: 'status', width: 80 },
  { title: '最后心跳', key: 'last_heartbeat', width: 160 },
  { title: '待指令', key: 'pending_command', width: 140 },
  { title: '操作', key: 'action', width: 200 },
]

const loadSys = async () => {
  sysLoading.value = true
  try {
    sysData.value = await componentApi.systemComponents()
  } catch {
    // 静默失败
  } finally {
    sysLoading.value = false
  }
}

const loadBiz = async () => {
  bizLoading.value = true
  try {
    const res = await componentApi.list({ search: bizSearch.value || undefined })
    bizComponents.value = res.results ?? (res as any)
    bizPagination.total = res.count ?? bizComponents.value.length
  } catch {
    // 静默失败
  } finally {
    bizLoading.value = false
  }
}

const handleTableChange = (pag: any) => {
  bizPagination.current = pag.current
  bizPagination.pageSize = pag.pageSize
  loadBiz()
}

// ── 升级弹窗 ──
const upgradeModalOpen = ref(false)
const upgradeLoading = ref(false)
const currentComponent = ref<ServiceComponent | null>(null)
const upgradeForm = reactive({ upgrade_version: '', upgrade_url: '', upgrade_checksum: '' })

const openUpgradeModal = (record: ServiceComponent) => {
  currentComponent.value = record
  upgradeForm.upgrade_version = record.upgrade_version || ''
  upgradeForm.upgrade_url = record.upgrade_url || ''
  upgradeForm.upgrade_checksum = record.upgrade_checksum || ''
  upgradeModalOpen.value = true
}

const submitUpgrade = async () => {
  if (!upgradeForm.upgrade_version || !upgradeForm.upgrade_url) {
    message.warning('目标版本和下载地址不能为空')
    return
  }
  upgradeLoading.value = true
  try {
    await componentApi.setUpgrade(currentComponent.value!.id, upgradeForm)
    message.success('升级任务已设置，等待业务服务心跳拉取')
    upgradeModalOpen.value = false
    loadBiz()
  } finally {
    upgradeLoading.value = false
  }
}

const cancelUpgrade = async (record: ServiceComponent) => {
  await componentApi.cancelUpgrade(record.id)
  message.success('升级任务已取消')
  loadBiz()
}

const setUninstall = async (record: ServiceComponent) => {
  await componentApi.setUninstall(record.id)
  message.success('卸载指令已下发，等待业务服务心跳拉取')
  loadBiz()
}

const cancelUninstall = async (record: ServiceComponent) => {
  await componentApi.cancelUninstall(record.id)
  message.success('卸载任务已取消')
  loadBiz()
}

const deleteComponent = async (record: ServiceComponent) => {
  await componentApi.delete(record.id)
  message.success('已删除')
  loadBiz()
}

onMounted(() => {
  loadSys()
  loadBiz()
})
</script>

<style scoped>
:deep(.ant-table-tbody > tr > td) {
  vertical-align: middle;
}
</style>
