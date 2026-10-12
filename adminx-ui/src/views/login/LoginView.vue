<template>
  <div class="login-container" :style="bgStyle">
    <div class="login-box">
      <div class="login-title">
        <h1>{{ siteName }}</h1>
        <p>{{ siteDesc }}</p>
      </div>

      <a-form
        :model="formState"
        :rules="rules"
        ref="formRef"
        @finish="handleSubmit"
        layout="vertical"
      >
        <a-form-item name="username">
          <a-input
            v-model:value="formState.username"
            size="large"
            placeholder="用户名"
            :prefix="h(UserOutlined)"
          />
        </a-form-item>

        <a-form-item name="password">
          <a-input-password
            v-model:value="formState.password"
            size="large"
            placeholder="密码"
            :prefix="h(LockOutlined)"
          />
        </a-form-item>

        <!-- 验证码 -->
        <a-form-item name="captchaText" v-if="captchaEnabled">
          <a-input
            v-model:value="formState.captchaText"
            size="large"
            placeholder="验证码"
            :prefix="h(SafetyOutlined)"
          >
            <template #suffix>
              <img
                v-if="captchaImage"
                class="captcha-img"
                :src="captchaImage"
                alt="验证码"
                title="看不清？点击刷新"
                @click="fetchCaptcha"
              />
            </template>
          </a-input>
        </a-form-item>

        <a-form-item>
          <a-button
            type="primary"
            size="large"
            block
            :loading="loading"
            html-type="submit"
          >
            登录
          </a-button>
        </a-form-item>
      </a-form>

      <div class="login-footer">
        <a-checkbox v-model:checked="rememberMe">记住我</a-checkbox>
        <a @click="forgotPassword">忘记密码？</a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined, SafetyOutlined } from '@ant-design/icons-vue'
import { useUserStore } from '@/stores/user'
import { commonApi } from '@/api/common'
import { captchaApi } from '@/api/auth'

const router = useRouter()
const userStore = useUserStore()

// 状态
const loading = ref(false)
const rememberMe = ref(false)
const REMEMBER_KEY = 'adminx_remember_username' 
const captchaEnabled = ref(false)
const captchaImage = ref('')
const captchaId = ref('')
const formRef = ref()
const siteName = ref('AdminX')
const siteDesc = ref('企业级 Admin 框架')
const loginBgImage = ref('')

// 表单状态
const formState = reactive({
  username: '',
  password: '',
  captchaText: '',
})

// 表单验证规则（用 computed 使 captchaText 的 required 随 captchaEnabled 动态变化，
// 否则 rules 在 setup 求值一次后不再更新）
const rules = computed(() => ({
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  captchaText: [{ required: captchaEnabled.value, message: '请输入验证码', trigger: 'blur' }],
}))

// 获取验证码（返回是否成功：加载失败时登录页要给出明确提示，
// 否则用户只看到一个空的验证码输入框）
const fetchCaptcha = async (): Promise<boolean> => {
  try {
    const res = await captchaApi.getCaptcha()
    // 后端未返回图片（版本不匹配等）按失败处理：不校验的话页面只会空白，
    // 看不出是接口出错还是渲染问题
    if (!res.image) {
      console.error('验证码接口未返回图片（后端版本可能不匹配）', res)
      return false
    }
    captchaId.value = res.captchaId
    captchaImage.value = res.image
    return true
  } catch (e) {
    console.error('获取验证码失败', e)
    return false
  }
}

// 提交登录
const handleSubmit = async () => {
  loading.value = true
  try {
    const result = await userStore.login(
      formState.username,
      formState.password,
      captchaEnabled.value ? captchaId.value : undefined,
      captchaEnabled.value ? formState.captchaText : undefined
    )

    await Promise.all([userStore.fetchUserInfo(), userStore.fetchSiteInfo()])
    // 记住我：仅记住用户名（密码绝不落盘）
    try {
      if (rememberMe.value) localStorage.setItem(REMEMBER_KEY, formState.username)
      else localStorage.removeItem(REMEMBER_KEY)
    } catch { /* 隐私模式忽略 */ }
    message.success('登录成功')
    router.push(userStore.user?.home_page || '/')
  } catch {
    // 错误提示已由全局拦截器统一处理，这里不再重复弹；失败时刷新验证码
    if (captchaEnabled.value) fetchCaptcha()
  } finally {
    loading.value = false
  }
}

// 忘记密码
const forgotPassword = () => {
  message.info('请联系管理员重置密码')
}

// 计算背景样式
const bgStyle = computed(() => {
  if (loginBgImage.value) {
    return {
      background: `url(${loginBgImage.value}) center / cover no-repeat`,
    }
  }
  // 为空时使用 CSS 中定义的默认渐变
  return {}
})

// 挂载时读站点信息里的验证码开关（公共端点返回，后端默认开启）。
// 读不到时按“需要”处理：登录页宁可多显示一次验证码，也不能漏掉这层防爆破校验。
const applyCaptchaRequired = async (required: boolean) => {
  captchaEnabled.value = required
  if (!required) return
  if (!(await fetchCaptcha())) {
    message.error('验证码加载失败，请检查后端服务与 Redis 配置')
  }
}

onMounted(async () => {
  // 清除残留的旧 token
  userStore.clearToken()
  // 回填记住的用户名
  try {
    const remembered = localStorage.getItem(REMEMBER_KEY)
    if (remembered) {
      rememberMe.value = true
      formState.username = remembered
    }
  } catch { /* ignore */ }
  // 站点名称/背景图/验证码开关以配置中心为唯一事实源
  // （此前本地 localStorage 值会覆盖服务端配置）
  let captchaRequired = true
  try {
    const res = await commonApi.getSiteInfo()
    siteName.value = res.site_name
    siteDesc.value = res.site_desc
    loginBgImage.value = res.login_bg_image || ''
    if (typeof res.login_captcha_required === 'boolean') captchaRequired = res.login_captcha_required
  } catch (e) {
    // 使用默认值（含验证码默认开启）
  }
  await applyCaptchaRequired(captchaRequired)
})
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-box {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

.login-title {
  text-align: center;
  margin-bottom: 32px;
}

.login-title h1 {
  font-size: 28px;
  color: #333;
  margin-bottom: 8px;
}

.login-title p {
  color: #666;
  font-size: 14px;
}

.captcha-img {
  width: 120px;
  height: 40px;
  object-fit: cover;
  border-radius: 4px;
  cursor: pointer;
  display: block;
}

.login-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 16px;
}

@media (max-width: 576px) {
  .login-box {
    width: 90%;
    padding: 24px;
  }
}
</style>
