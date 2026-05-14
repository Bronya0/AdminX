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
  message.warning('会话已超时，请重新登录')
  await userStore.logout()
  router.push('/login')
}

// 监听用户活动事件
const activityEvents = ['mousedown', 'mousemove', 'keydown', 'scroll', 'touchstart', 'click']

const startListening = () => {
  activityEvents.forEach(event => {
    window.addEventListener(event, resetIdleTimer, { passive: true })
  })
  resetIdleTimer()
}

const stopListening = () => {
  activityEvents.forEach(event => {
    window.removeEventListener(event, resetIdleTimer)
  })
  if (idleTimer) clearTimeout(idleTimer)
  if (countdownTimer) clearInterval(countdownTimer)
  if (configRefreshTimer) clearInterval(configRefreshTimer)
}

// 当超时配置变化时重设
watch(() => userStore.idleTimeout, (val) => {
  idleTimeoutMs.value = val > 0 ? val * 60 * 1000 : 0
  if (val > 0 && userStore.isLoggedIn) {
    resetIdleTimer()
  }
})

onMounted(() => {
  const timeout = userStore.idleTimeout ?? 30
  idleTimeoutMs.value = timeout > 0 ? timeout * 60 * 1000 : 0
  if (idleTimeoutMs.value > 0 && userStore.isLoggedIn) {
    startListening()
    // 每分钟轮询配置中心的超时设置，使配置修改即时生效
    configRefreshTimer = setInterval(refreshTimeoutConfig, 60 * 1000)
  }
})

onUnmounted(() => {
  stopListening()
})
</script>
