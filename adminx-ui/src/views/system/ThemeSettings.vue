<template>
  <div class="theme-settings-inner">
    <a-tabs v-model:activeKey="activeTab" type="card">
        <!-- ===== 布局 ===== -->
        <a-tab-pane key="layout" tab="布局">
          <a-form layout="vertical" style="max-width: 600px;">
            <a-form-item label="布局模式">
              <a-radio-group v-model:value="form.layout" button-style="solid" size="large" @change="onLayoutChange">
                <a-radio-button value="side">
                  <MenuFoldOutlined /> 侧边栏
                </a-radio-button>
                <a-radio-button value="top">
                  <MenuOutlined /> 顶部导航
                </a-radio-button>
                <a-radio-button value="mix">
                  <AppstoreOutlined /> 混合布局
                </a-radio-button>
              </a-radio-group>
              <div style="margin-top: 4px; font-size: 12px; color: #999;">
                {{ layoutDesc(form.layout) }}
              </div>
            </a-form-item>

            <a-form-item label="侧边栏默认状态">
              <a-switch v-model:checked="form.collapsed" checked-children="收起" un-checked-children="展开" @change="onCollapsedChange" />
              <span style="margin-left: 8px; color: #999; font-size: 12px;">页面初始加载时侧边栏是否收起</span>
            </a-form-item>

            <a-form-item label="显示 Logo">
              <a-switch v-model:checked="form.showLogo" @change="updateTheme" />
              <span style="margin-left: 8px; color: #999; font-size: 12px;">侧边栏顶部是否显示站点 Logo 和名称</span>
            </a-form-item>

            <a-form-item label="显示版本号">
              <a-switch v-model:checked="form.showVersion" @change="updateTheme" />
              <span style="margin-left: 8px; color: #999; font-size: 12px;">侧边栏底部是否显示平台版本号</span>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <!-- ===== 主题 ===== -->
        <a-tab-pane key="theme" tab="主题">
          <a-form layout="vertical" style="max-width: 600px;">
            <a-form-item label="主题模式">
              <a-segmented
                v-model:value="form.isDark"
                :options="[
                  { label: '☀ 浅色', value: false },
                  { label: '🌙 深色', value: true },
                ]"
                block
                @change="onDarkChange"
              />
            </a-form-item>

            <a-form-item label="主题色">
              <div style="display: flex; align-items: center; gap: 16px;">
                <a-input
                  v-model:value="form.primaryColor"
                  type="color"
                  style="width: 48px; height: 36px; padding: 2px; cursor: pointer;"
                  @change="onPrimaryColorChange"
                />
                <span style="font-family: monospace;">{{ form.primaryColor }}</span>
                <a-button size="small" @click="resetPrimaryColor">恢复默认</a-button>
              </div>
              <div style="margin-top: 12px; display: flex; gap: 8px; flex-wrap: wrap;">
                <div
                  v-for="c in presetColors"
                  :key="c"
                  :style="{ background: c, width: '28px', height: '28px', borderRadius: '6px', cursor: 'pointer', border: form.primaryColor === c ? '3px solid #333' : '2px solid #e8e8e8' }"
                  @click="selectPresetColor(c)"
                />
              </div>
            </a-form-item>

            <a-divider>内容区域</a-divider>

            <a-form-item label="显示面包屑">
              <a-switch v-model:checked="form.showBreadcrumb" @change="updateTheme" />
            </a-form-item>

            <a-form-item label="显示多标签">
              <a-switch v-model:checked="form.showTabs" @change="updateTheme" />
              <span style="margin-left: 8px; color: #999; font-size: 12px;">（功能开发中）</span>
            </a-form-item>

            <a-form-item label="显示页脚">
              <a-switch v-model:checked="form.showFooter" @change="updateTheme" />
            </a-form-item>

            <a-divider>视觉效果</a-divider>

            <a-form-item label="圆角大小">
              <a-slider v-model:value="form.borderRadius" :min="0" :max="16" :step="1" @change="onBorderRadiusChange" />
              <span style="margin-left: 8px; font-size: 12px; color: #999;">{{ form.borderRadius }}px</span>
            </a-form-item>

            <a-form-item label="字体大小">
              <a-select v-model:value="form.fontSize" style="width: 160px;" @change="onFontSizeChange" :options="fontSizeOptions" />
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <!-- ===== 站点信息 ===== -->
        <a-tab-pane key="site" tab="站点信息">
          <a-form layout="vertical" style="max-width: 600px;">
            <a-form-item label="站点名称">
              <a-input v-model:value="form.siteName" placeholder="如: DjangoAdminX" @change="onSiteNameChange" />
            </a-form-item>

            <a-form-item label="站点描述">
              <a-textarea v-model:value="form.siteDesc" placeholder="如: 企业级 Django Admin 框架" :rows="2" @change="onSiteDescChange" />
            </a-form-item>

            <a-form-item label="Logo URL">
              <a-input v-model:value="form.siteLogo" placeholder="https://example.com/logo.png（可选）" @change="onSiteLogoChange" />
            </a-form-item>

            <a-form-item label="浏览器标签图标 (Favicon)">
              <a-input v-model:value="form.favicon" placeholder="https://example.com/favicon.ico（可选）" @change="onFaviconChange" />
            </a-form-item>

            <a-form-item label="平台版本号">
              <a-input v-model:value="form.appVersion" placeholder="如: 1.0.0" @change="onAppVersionChange" style="width: 200px;" />
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <!-- ===== 登录页 ===== -->
        <a-tab-pane key="login" tab="登录页">
          <a-form layout="vertical" style="max-width: 600px;">
            <a-form-item label="登录页背景图 URL">
              <a-input v-model:value="form.loginBgImage" placeholder="如: https://example.com/bg.jpg（为空则使用默认渐变背景）" @change="onLoginBgChange" />
              <div v-if="form.loginBgImage" style="margin-top: 8px; padding: 8px; background: #f5f5f5; border-radius: 6px;">
                <img :src="form.loginBgImage" alt="预览" style="max-width: 100%; max-height: 120px; border-radius: 4px;"
                     @error="(e: any) => e.target.style.display = 'none'" />
              </div>
            </a-form-item>

            <a-form-item label="会话空闲超时（分钟）">
              <a-input-number v-model:value="form.idleTimeout" :min="0" :max="480" style="width: 160px;" @change="onIdleTimeoutChange" />
              <span style="margin-left: 8px; color: #999; font-size: 12px;">0 = 不超时</span>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <!-- ===== NTP 时间同步 ===== -->
        <a-tab-pane key="ntp" tab="NTP 时间同步">
          <a-form layout="vertical" style="max-width: 600px;">
            <a-form-item label="启用 NTP">
              <a-switch v-model:checked="ntpForm.enabled" @change="onNtpEnabledChange" />
              <span style="margin-left: 8px; color: #999; font-size: 12px;">
                开启后系统将定时向 NTP 服务器同步时间
              </span>
            </a-form-item>

            <a-form-item label="NTP 服务器地址">
              <a-input
                v-model:value="ntpForm.server"
                placeholder="如: ntp.aliyun.com"
                :disabled="!ntpForm.enabled"
                @change="onNtpServerChange"
              />
            </a-form-item>

            <a-form-item label="同步间隔（分钟）">
              <a-input-number
                v-model:value="ntpForm.interval"
                :min="5"
                :max="1440"
                :disabled="!ntpForm.enabled"
                style="width: 160px;"
                @change="onNtpIntervalChange"
              />
              <span style="margin-left: 8px; color: #999; font-size: 12px;">建议 30~120 分钟</span>
            </a-form-item>

            <a-divider />

            <a-form-item label="手动同步">
              <a-space>
                <a-button type="primary" @click="syncNtpNow" :loading="ntpSyncing" :disabled="!ntpForm.server">
                  立即同步
                </a-button>
                <a-tag v-if="ntpResult" :color="ntpResult.success ? 'success' : 'error'">
                  {{ ntpResult.success ? `同步成功，偏差 ${ntpResult.offset} 秒` : ntpResult.error || '同步失败' }}
                </a-tag>
              </a-space>
            </a-form-item>

            <a-alert
              v-if="ntpForm.enabled && !ntpForm.server"
              type="warning"
              message="请先填写 NTP 服务器地址"
              style="margin-bottom: 16px;"
              banner
            />
            <a-alert
              v-if="!ntpForm.enabled"
              type="info"
              message="NTP 时间同步未启用，系统使用本机时间"
              banner
            />
          </a-form>
        </a-tab-pane>

        <!-- ===== 预览 ===== -->
        <a-tab-pane key="preview" tab="预览">
          <div class="preview-area" :class="{ 'preview-dark': form.isDark }">
            <div class="preview-frame">
              <!-- 模拟顶栏 -->
              <div class="preview-header" :style="{ background: form.isDark ? '#1f1f1f' : '#fff' }">
                <div class="preview-header-left">
                  <div class="preview-logo-dot" :style="{ background: form.primaryColor }" />
                  <span v-if="form.showBreadcrumb" style="font-size: 12px; opacity: 0.5;">首页 / 页面</span>
                </div>
                <div class="preview-header-right">
                  <div class="preview-dot"></div>
                  <div class="preview-dot"></div>
                  <div class="preview-avatar" :style="{ background: form.primaryColor }"></div>
                </div>
              </div>

              <!-- 模拟侧边栏 + 内容 -->
              <div class="preview-body">
                <div
                  v-if="form.layout !== 'top'"
                  class="preview-sider"
                  :style="{
                    width: form.collapsed ? '40px' : '120px',
                    background: form.isDark ? '#000' : '#001529',
                  }"
                >
                  <template v-if="form.showLogo">
                    <div class="preview-sider-logo" :style="{ borderColor: 'rgba(255,255,255,0.1)' }">
                      <div class="preview-logo-dot" :style="{ background: form.primaryColor }" />
                      <span v-if="!form.collapsed" style="color: #fff; font-size: 10px;">{{ form.siteName }}</span>
                    </div>
                  </template>
                  <div class="preview-sider-item" v-for="i in 4" :key="i"
                       :style="{ background: i === 1 ? form.primaryColor : 'transparent' }" />
                </div>

                <!-- 顶导（top/mix） -->
                <div v-if="form.layout !== 'side'" class="preview-topnav" :style="{ background: form.isDark ? '#000' : '#001529' }">
                  <div class="preview-topnav-item" v-for="i in 4" :key="i"
                       :style="{ background: i === 1 ? form.primaryColor : 'transparent' }" />
                </div>

                <div class="preview-content" :style="{ background: form.isDark ? '#141414' : '#f0f2f5', borderRadius: form.borderRadius + 'px' }">
                  <div v-if="form.showTabs" class="preview-tabs" :style="{ background: form.isDark ? '#1f1f1f' : '#fff' }">
                    <div class="preview-tab" :style="{ borderColor: form.primaryColor }">标签1</div>
                    <div class="preview-tab">标签2</div>
                  </div>
                  <div class="preview-card" :style="{ background: form.isDark ? '#1f1f1f' : '#fff', borderRadius: form.borderRadius + 'px' }">
                    <div class="preview-card-line" style="width: 60%" />
                    <div class="preview-card-line" style="width: 80%" />
                    <div class="preview-card-line" style="width: 40%" />
                    <a-button :type="'primary'" size="small" :style="{ background: form.primaryColor, borderColor: form.primaryColor }">
                      按钮
                    </a-button>
                  </div>
                </div>
              </div>

              <!-- 模拟页脚 -->
              <div v-if="form.showFooter" class="preview-footer" :style="{ background: form.isDark ? '#1f1f1f' : '#f0f2f5' }">
                <span style="font-size: 10px; opacity: 0.5;">{{ form.siteName }} © {{ new Date().getFullYear() }}</span>
              </div>
            </div>
          </div>
        </a-tab-pane>
      </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  MenuFoldOutlined,
  MenuOutlined,
  AppstoreOutlined,
} from '@ant-design/icons-vue'
import { useUserStore } from '@/stores/user'
import { commonApi } from '@/api/common'

const userStore = useUserStore()

const activeTab = ref('layout')

const presetColors = ['#1890ff', '#f5222d', '#fa541c', '#faad14', '#52c41a', '#13c2c2', '#1677ff', '#2f54eb', '#722ed1', '#eb2f96']

const fontSizeOptions = [
  { label: '小 (13px)', value: 13 },
  { label: '默认 (14px)', value: 14 },
  { label: '中 (15px)', value: 15 },
  { label: '大 (16px)', value: 16 },
]

const layoutDesc = (layout: string) => {
  const map: Record<string, string> = {
    side: '菜单全部展示在左侧侧边栏',
    top: '菜单全部展示在顶部导航栏',
    mix: '一级菜单在顶部，子菜单在左侧侧边栏',
  }
  return map[layout] || ''
}

// 表单同步自 store
const form = reactive({
  layout: userStore.theme.layout,
  collapsed: userStore.theme.collapsed,
  isDark: userStore.theme.isDark,
  primaryColor: userStore.theme.primaryColor,
  showLogo: userStore.theme.showLogo,
  showBreadcrumb: userStore.theme.showBreadcrumb,
  showTabs: userStore.theme.showTabs,
  showFooter: userStore.theme.showFooter,
  showVersion: userStore.theme.showVersion !== false,
  borderRadius: userStore.theme.borderRadius ?? 6,
  fontSize: userStore.theme.fontSize ?? 14,
  siteName: userStore.siteName,
  siteDesc: userStore.siteDesc,
  siteLogo: userStore.siteLogo,
  favicon: (document.querySelector('link[rel="icon"]') as HTMLLinkElement)?.href || '',
  loginBgImage: localStorage.getItem('login_bg_image') || '',
  idleTimeout: userStore.idleTimeout,
  appVersion: userStore.appVersion,
})

// 同步 store → form（当 store 被其他地方修改时）
onMounted(() => {
  syncFormFromStore()
  // 恢复 NTP 配置
  ntpForm.enabled = localStorage.getItem('ntp_enabled') === '1'
  ntpForm.server = localStorage.getItem('ntp_server') || ''
  const savedInterval = localStorage.getItem('ntp_interval')
  if (savedInterval) ntpForm.interval = Number(savedInterval)
})

const syncFormFromStore = () => {
  form.layout = userStore.theme.layout
  form.collapsed = userStore.theme.collapsed
  form.isDark = userStore.theme.isDark
  form.primaryColor = userStore.theme.primaryColor
  form.showLogo = userStore.theme.showLogo
  form.showBreadcrumb = userStore.theme.showBreadcrumb
  form.showTabs = userStore.theme.showTabs
  form.showFooter = userStore.theme.showFooter
  form.showVersion = userStore.theme.showVersion !== false
  form.borderRadius = userStore.theme.borderRadius ?? 6
  form.fontSize = userStore.theme.fontSize ?? 14
  form.siteName = userStore.siteName
  form.siteDesc = userStore.siteDesc
  form.siteLogo = userStore.siteLogo
  form.idleTimeout = userStore.idleTimeout
  form.appVersion = userStore.appVersion
}

// ---- 通用主题更新 ----
const updateTheme = () => {
  userStore.updateTheme({
    layout: form.layout as any,
    isDark: form.isDark,
    primaryColor: form.primaryColor,
    collapsed: form.collapsed,
    showLogo: form.showLogo,
    showBreadcrumb: form.showBreadcrumb,
    showTabs: form.showTabs,
    showFooter: form.showFooter,
    showVersion: form.showVersion,
    borderRadius: form.borderRadius,
    fontSize: form.fontSize,
  })
}

// ---- 各字段处理 ----
const onLayoutChange = () => {
  updateTheme()
  message.success(`已切换为 ${layoutDesc(form.layout)}`)
}

const onDarkChange = () => {
  if (form.isDark) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  updateTheme()
}

const onPrimaryColorChange = () => {
  document.documentElement.style.setProperty('--primary-color', form.primaryColor)
  updateTheme()
}

const selectPresetColor = (color: string) => {
  form.primaryColor = color
  onPrimaryColorChange()
}

const resetPrimaryColor = () => {
  form.primaryColor = '#1890ff'
  onPrimaryColorChange()
}

const onCollapsedChange = () => {
  updateTheme()
}

const onBorderRadiusChange = () => {
  updateTheme()
}

const onFontSizeChange = () => {
  document.documentElement.style.fontSize = form.fontSize + 'px'
  updateTheme()
}

const onSiteNameChange = () => {
  userStore.siteName = form.siteName
  document.title = form.siteName
}

const onSiteDescChange = () => {
  userStore.siteDesc = form.siteDesc
}

const onSiteLogoChange = () => {
  userStore.siteLogo = form.siteLogo
}

const onFaviconChange = () => {
  const link = document.querySelector('link[rel="icon"]') as HTMLLinkElement
  if (link) {
    link.href = form.favicon
  } else if (form.favicon) {
    const newLink = document.createElement('link')
    newLink.rel = 'icon'
    newLink.href = form.favicon
    document.head.appendChild(newLink)
  }
}

const onLoginBgChange = () => {
  if (form.loginBgImage) {
    localStorage.setItem('login_bg_image', form.loginBgImage)
  } else {
    localStorage.removeItem('login_bg_image')
  }
}

const onIdleTimeoutChange = () => {
  userStore.idleTimeout = form.idleTimeout
}

const onAppVersionChange = () => {
  userStore.appVersion = form.appVersion
}

// ─── NTP 时间同步 ───
const ntpForm = reactive({
  enabled: false,
  server: '',
  interval: 60,
})

const ntpSyncing = ref(false)
const ntpResult = ref<{ success: boolean; offset?: number; error?: string } | null>(null)

const onNtpEnabledChange = () => {
  if (ntpForm.enabled) {
    localStorage.setItem('ntp_enabled', '1')
  } else {
    localStorage.removeItem('ntp_enabled')
    ntpResult.value = null
  }
}

const onNtpServerChange = () => {
  localStorage.setItem('ntp_server', ntpForm.server)
}

const onNtpIntervalChange = () => {
  localStorage.setItem('ntp_interval', String(ntpForm.interval))
}

const syncNtpNow = async () => {
  ntpSyncing.value = true
  ntpResult.value = null
  try {
    const res = await commonApi.ntpSync()
    ntpResult.value = res
  } catch {
    ntpResult.value = { success: false, error: '请求失败' }
  } finally {
    ntpSyncing.value = false
  }
}
</script>

<style scoped>
.theme-settings-inner {
  min-height: 200px;
}

/* ===== 预览区域 ===== */
.preview-area {
  border: 2px solid #e8e8e8;
  border-radius: 8px;
  overflow: hidden;
  background: #fafafa;
  transition: all 0.3s;
}
.preview-area.preview-dark {
  border-color: #333;
  background: #0a0a0a;
}

.preview-frame {
  display: flex;
  flex-direction: column;
  min-height: 360px;
}

.preview-header {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
  border-bottom: 1px solid #e8e8e8;
}
.preview-dark .preview-header {
  border-color: #333;
}
.preview-header-left, .preview-header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-logo-dot {
  width: 14px;
  height: 14px;
  border-radius: 4px;
}
.preview-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ccc;
}
.preview-dark .preview-dot {
  background: #555;
}
.preview-avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
}

.preview-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.preview-sider {
  display: flex;
  flex-direction: column;
  padding: 6px 4px;
  gap: 4px;
  transition: width 0.3s;
  overflow: hidden;
  flex-shrink: 0;
}
.preview-sider-logo {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 6px;
  border-bottom: 1px solid;
  margin-bottom: 4px;
  white-space: nowrap;
}
.preview-sider-item {
  height: 8px;
  border-radius: 3px;
  opacity: 0.6;
}

.preview-topnav {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  height: 28px;
}
.preview-topnav-item {
  height: 6px;
  border-radius: 3px;
  padding: 0 8px;
  opacity: 0.6;
}

.preview-content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.preview-content {
  flex: 1;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preview-tabs {
  display: flex;
  gap: 4px;
  padding: 6px 8px 0;
  border-radius: 4px 4px 0 0;
}
.preview-tab {
  padding: 2px 10px;
  font-size: 10px;
  border-bottom: 2px solid transparent;
}
.preview-tab:first-child {
  border-bottom-style: solid;
  font-weight: bold;
}

.preview-card {
  flex: 1;
  padding: 12px;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.preview-dark .preview-card {
  box-shadow: 0 1px 4px rgba(0,0,0,0.3);
}
.preview-card-line {
  height: 6px;
  border-radius: 3px;
  background: #e8e8e8;
}
.preview-dark .preview-card-line {
  background: #333;
}

.preview-footer {
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-top: 1px solid #e8e8e8;
}
.preview-dark .preview-footer {
  border-color: #333;
}
</style>
