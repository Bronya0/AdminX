<template>
  <a-modal
    v-model:open="showWarning"
    :closable="false"
    :mask-closable="false"
    :footer="null"
    title="会话即将超时"
    width="400px"
    class="idle-warning-modal"
  >
    <div style="text-align: center; padding: 16px 0;">
      <p style="font-size: 14px; color: #666;">
        由于长时间无操作，会话将在 <strong style="color: #ff4d4f; font-size: 24px;">{{ countdown }}</strong> 秒后自动超时退出
      </p>
      <a-button type="primary" size="large" @click="extendSession" style="margin-top: 16px;">
        继续会话
      </a-button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { commonApi } from '@/api/common'
import { message } from 'ant-design-vue'

const userStore = useUserStore()
const router = useRouter()

const idleTimeoutMs = ref(0)       // 空闲超时毫秒数
const gracePeriod = 60             // 倒计时秒数
const countdown = ref(gracePeriod)
const showWarning = ref(false)

let idleTimer: ReturnType<typeof setTimeout> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null
let configRefreshTimer: ReturnType<typeof setInterval> | null = null

// 刷新超时配置（使配置中心修改即时生效）
const refreshTimeoutConfig = async () => {
  try {
    const info = await commonApi.getSiteInfo()
    const newTimeout = info.idle_timeout ?? 30
    if (newTimeout !== userStore.idleTimeout) {
      userStore.idleTimeout = newTimeout
    }
  } catch {
    // 静默失败，沿用旧值
  }
}

// 重置空闲计时器
const resetIdleTimer = () => {
  if (idleTimer) clearTimeout(idleTimer)
  if (showWarning.value) return // 已显示警告窗口时不重置

  if (idleTimeoutMs.value > 0) {
    idleTimer = setTimeout(showTimeoutWarning, idleTimeoutMs.value)
  }
}

// 显示超时警告
const showTimeoutWarning = () => {
  countdown.value = gracePeriod
  showWarning.value = true
  // 启动倒计时
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      doLogout()
    }
  }, 1000)
}

// 继续会话
const extendSession = () => {
  showWarning.value = false
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  resetIdleTimer()
}

// 执行登出
const doLogout = async () => {
  if (countdownTimer) clearInterval(countdownTimer)
  showWarning.value = false

  // 必须先调服务端登出（此时 refresh token 还在内存里），再跳登录页。
  // 顺序反了会静默失效：LoginView.onMounted 会 clearToken()，
  // refresh 被清空后 logout() 直接跳过服务端吊销，旧 refresh 7 天内仍可用。
  await userStore.logout()
  message.warning('会话已超时，请重新登录')
  await router.replace('/login').catch(() => {})
}

// 监听用户活动事件
const activityEvents = ['mousedown', 'mousemove', 'keydown', 'scroll', 'touchstart', 'click']

let listening = false

const startListening = () => {
  if (listening) return
  listening = true
  activityEvents.forEach(event => {
    window.addEventListener(event, resetIdleTimer, { passive: true })
  })
  // 每分钟轮询配置中心的超时设置，使配置修改即时生效
  if (!configRefreshTimer) {
    configRefreshTimer = setInterval(refreshTimeoutConfig, 60 * 1000)
  }
  resetIdleTimer()
}

const stopListening = () => {
  if (!listening) return
  listening = false
  activityEvents.forEach(event => {
    window.removeEventListener(event, resetIdleTimer)
  })
  if (idleTimer) {
    clearTimeout(idleTimer)
    idleTimer = null
  }
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  if (configRefreshTimer) {
    clearInterval(configRefreshTimer)
    configRefreshTimer = null
  }
}

// 当超时配置变化时重设
watch(() => userStore.idleTimeout, (val) => {
  idleTimeoutMs.value = val > 0 ? val * 60 * 1000 : 0
  // 关闭超时（val=0）：必须停掉已挂载的 timer 与活动监听，
  // 否则旧 timer 仍会弹超时警告
  if (idleTimeoutMs.value === 0) {
    stopListening()
    return
  }
  if (!userStore.isLoggedIn) return
  // 从"挂载时超时=0、从未挂监听"切到启用：必须补挂活动监听，
  // 否则用户一直在操作也收不到活动事件，到点照样被登出
  if (!listening) startListening()
  else resetIdleTimer()
})

onMounted(() => {
  const timeout = userStore.idleTimeout ?? 30
  idleTimeoutMs.value = timeout > 0 ? timeout * 60 * 1000 : 0
  if (idleTimeoutMs.value > 0 && userStore.isLoggedIn) {
    startListening()
  }
})

onUnmounted(() => {
  stopListening()
})
</script>
