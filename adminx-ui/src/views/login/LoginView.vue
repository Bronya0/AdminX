<template>
  <div class="login-container">
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
              <div
                class="captcha-img"
                v-html="captchaSvg"
                @click="fetchCaptcha"
                style="cursor: pointer"
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
import { ref, reactive, h, onMounted } from 'vue'
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
const captchaEnabled = ref(false)
const captchaSvg = ref('')
const captchaId = ref('')
const formRef = ref()
const siteName = ref('DjangoAdminX')
const siteDesc = ref('企业级 Django Admin 框架')

// 表单状态
const formState = reactive({
  username: '',
  password: '',
  captchaText: '',
})

// 表单验证规则
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  captchaText: [{ required: captchaEnabled.value, message: '请输入验证码', trigger: 'blur' }],
}

// 获取验证码
const fetchCaptcha = async () => {
  try {
    const res = await captchaApi.getCaptcha()
    captchaId.value = res.captchaId
    captchaSvg.value = res.svg
  } catch (e) {
    console.error('获取验证码失败', e)
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

    console.log('Login result:', result)
    console.log('User info:', userStore.user)
    console.log('Username:', userStore.username)

    message.success('登录成功')
    router.push('/')
  } catch (e: any) {
    console.error('Login error:', e)
    // 登录失败，刷新验证码
    if (captchaEnabled.value) {
      fetchCaptcha()
    }
  } finally {
    loading.value = false
  }
}

// 忘记密码
const forgotPassword = () => {
  message.info('请联系管理员重置密码')
}

onMounted(async () => {
  // 获取站点名称
  try {
    const res = await commonApi.getSiteInfo()
    siteName.value = res.site_name
    siteDesc.value = res.site_desc
  } catch (e) {
    // 使用默认值
  }
  // 检查是否需要验证码（从配置读取）
  // captchaEnabled.value = true
  // if (captchaEnabled.value) {
  //   fetchCaptcha()
  // }
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
  width: 100px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.captcha-img svg {
  width: 100%;
  height: 100%;
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
