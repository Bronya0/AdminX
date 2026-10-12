<template>
  <div class="theme-settings-inner">
    <a-tabs v-model:activeKey="activeTab" type="card">
        <!-- ===== 布局 ===== -->
        <a-tab-pane key="layout" tab="布局">
          <a-form layout="vertical" style="max-width: 600px;">
            <a-form-item label="布局模式">
              <a-radio-group v-model:value="form.layout" button-style="solid" size="large" @change="updateTheme">
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
              <a-switch v-model:checked="form.collapsed" checked-children="收起" un-checked-children="展开" @change="updateTheme" />
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

            <a-divider>行高</a-divider>

            <a-form-item label="顶栏高度">
              <a-slider v-model:value="form.navHeight" :min="40" :max="88" :step="2" @change="updateTheme" />
              <span style="margin-left: 8px; font-size: 12px; color: #999;">{{ form.navHeight }}px</span>
              <div style="font-size: 12px; color: #999;">
                一级导航那一行（顶部导航 / 混合布局的顶栏），默认 56px
              </div>
            </a-form-item>

            <a-form-item label="页头高度">
              <a-slider v-model:value="form.headerHeight" :min="40" :max="88" :step="2" @change="updateTheme" />
              <span style="margin-left: 8px; font-size: 12px; color: #999;">{{ form.headerHeight }}px</span>
              <div style="font-size: 12px; color: #999;">
                面包屑 + 配置中心/通知/账号那一行，默认 64px；调小可压缩上方留白
              </div>
            </a-form-item>

            <a-divider>导航配色</a-divider>

            <a-form-item label="顶栏底色">
              <ThemeColorField
                v-model="form.navTopBgColor"
                :presets="presetNavColors"
                :default-value="defaultTheme.navTopBgColor"
                @change="updateTheme"
              />
            </a-form-item>

            <a-form-item label="侧栏底色">
              <ThemeColorField
                v-model="form.navSideBgColor"
                :presets="presetNavColors"
                :default-value="defaultTheme.navSideBgColor"
                @change="updateTheme"
              />
            </a-form-item>

            <a-form-item label="导航文字大小">
              <a-slider v-model:value="form.navTextSize" :min="12" :max="18" :step="1" @change="updateTheme" />
              <span style="margin-left: 8px; font-size: 12px; color: #999;">{{ form.navTextSize }}px（默认 14px）</span>
            </a-form-item>

            <a-form-item label="导航文字颜色">
              <a-switch
                v-model:checked="form.navTextAuto"
                checked-children="跟随底色"
                un-checked-children="自定义"
                @change="updateTheme"
              />
              <div v-if="!form.navTextAuto" style="margin-top: 12px;">
                <ThemeColorField
                  v-model="form.navTextColor"
                  :presets="presetNavTextColors"
                  :default-value="defaultTheme.navTextColor"
                  @change="updateTheme"
                />
              </div>
              <div style="margin-top: 8px; font-size: 12px; color: #999;">
                跟随底色：亮底配深字、暗底配白字（分隔线也按底色走）；改成自定义后顶栏与侧栏用同一个字色
              </div>
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
                @change="updateTheme"
              />
            </a-form-item>

            <a-form-item label="主题色">
              <ThemeColorField
                v-model="form.primaryColor"
                :presets="presetColors"
                default-value="#1890ff"
                @change="updateTheme"
              />
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
              <a-slider v-model:value="form.borderRadius" :min="0" :max="16" :step="1" @change="updateTheme" />
              <span style="margin-left: 8px; font-size: 12px; color: #999;">{{ form.borderRadius }}px</span>
            </a-form-item>

            <a-form-item label="字体大小">
              <a-select v-model:value="form.fontSize" style="width: 160px;" @change="updateTheme" :options="fontSizeOptions" />
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <!-- ===== 站点信息 ===== -->
        <a-tab-pane key="site" tab="站点信息">
          <a-form layout="vertical" style="max-width: 600px;">
            <a-form-item label="站点名称">
              <a-input v-model:value="form.siteName" placeholder="如: AdminX" @change="onSiteNameChange" />
            </a-form-item>

            <a-form-item label="站点描述">
              <a-textarea v-model:value="form.siteDesc" placeholder="如: 企业级 Admin 框架" :rows="2" @change="onSiteDescChange" />
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

        <!-- ===== 预览 ===== -->
        <a-tab-pane key="preview" tab="预览">
          <div class="preview-area" :class="{ 'preview-dark': form.isDark }">
            <div class="preview-frame">
              <!-- 模拟顶栏 -->
              <div class="preview-header" :style="{ height: previewHeight(form.headerHeight, PREVIEW_SCALE.header), background: form.isDark ? '#1f1f1f' : '#fff' }">
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
                    background: form.navSideBgColor,
                  }"
                >
                  <template v-if="form.showLogo">
                    <div class="preview-sider-logo" :style="{ borderColor: userStore.navSide.menuTheme === 'light' ? 'rgba(0, 0, 0, 0.06)' : 'rgba(255, 255, 255, 0.1)' }">
                      <div class="preview-logo-dot" :style="{ background: form.primaryColor }" />
                      <span v-if="!form.collapsed" :style="{ color: userStore.navSide.menuTheme === 'light' ? 'rgba(0, 0, 0, 0.88)' : '#fff', fontSize: '10px' }">{{ form.siteName }}</span>
                    </div>
                  </template>
                  <div class="preview-sider-item" v-for="i in 4" :key="i"
                       :style="{ background: i === 1 ? form.primaryColor : 'transparent' }" />
                </div>

                <!-- 顶导（top/mix） -->
                <div v-if="form.layout !== 'side'" class="preview-topnav" :style="{ height: previewHeight(form.navHeight, PREVIEW_SCALE.nav), background: form.navTopBgColor }">
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
      <div class="settings-footer">
        <span class="settings-footer-hint">导入 / 导出只包含主题外观（布局、主题、导航配色等），不含站点信息与登录页设置</span>
        <a-space>
          <a-button @click="exportTheme">
            <DownloadOutlined /> 导出主题
          </a-button>
          <a-upload :show-upload-list="false" accept=".json,application/json" :before-upload="importTheme">
            <a-button>
              <UploadOutlined /> 导入主题
            </a-button>
          </a-upload>
          <a-button type="primary" :loading="saving" @click="saveAll">保存设置</a-button>
        </a-space>
      </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import {
  AppstoreOutlined,
  DownloadOutlined,
  MenuFoldOutlined,
  MenuOutlined,
  UploadOutlined,
} from '@ant-design/icons-vue'
import { useUserStore, defaultTheme } from '@/stores/user'
import type { ThemeConfig } from '@/types'
import ThemeColorField from './ThemeColorField.vue'
import { commonApi } from '@/api/common'
import { themeApi } from '@/api/theme'
import { downloadJsonFile, readJsonFile } from '@/utils/jsonFile'

const userStore = useUserStore()

const activeTab = ref('layout')

const presetColors = ['#1890ff', '#f5222d', '#fa541c', '#faad14', '#52c41a', '#13c2c2', '#1677ff', '#2f54eb', '#722ed1', '#eb2f96']

// 导航区底色预设：平台现状深蓝、主题主色、白与几个中性色
const presetNavColors = ['#001529', '#1677ff', '#ffffff', '#f5f7fa', '#1f2937', '#0f172a']

// 导航文字色预设：白、浅灰、深字、主色
const presetNavTextColors = ['#ffffff', '#d9d9d9', '#000000', '#1890ff']

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

// 预览区是缩略图：按平台默认高度等比缩放（顶栏 56 → 28、页头 64 → 36）
const PREVIEW_SCALE = { nav: 1 / 2, header: 9 / 16 }
const previewHeight = (value: number, ratio: number) => `${Math.round(value * ratio)}px`

// 表单同步自 store
// 注意: 不能在 setup 顶层访问 document/localStorage（SSR/测试环境会报错，
// 且 favicon 读取 DOM 绝对 URL 会把部署域名硬编码写回后端）。
// favicon 从 store 读取（store 由 fetchSiteInfo 维护），loginBgImage 在 onMounted 初始化。
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
  navHeight: userStore.theme.navHeight ?? defaultTheme.navHeight,
  headerHeight: userStore.theme.headerHeight ?? defaultTheme.headerHeight,
  navTopBgColor: userStore.theme.navTopBgColor ?? defaultTheme.navTopBgColor,
  navSideBgColor: userStore.theme.navSideBgColor ?? defaultTheme.navSideBgColor,
  navTextSize: userStore.theme.navTextSize ?? defaultTheme.navTextSize,
  navTextAuto: userStore.theme.navTextAuto ?? defaultTheme.navTextAuto,
  navTextColor: userStore.theme.navTextColor ?? defaultTheme.navTextColor,
  siteName: userStore.siteName,
  siteDesc: userStore.siteDesc,
  siteLogo: userStore.siteLogo,
  favicon: userStore.favicon,
  loginBgImage: '',
  idleTimeout: userStore.idleTimeout,
  appVersion: userStore.appVersion,
})

// 打开设置页时先拉取服务端最新主题再回填表单，避免用本地缓存覆盖服务端的值
onMounted(async () => {
  await userStore.fetchTheme()
  syncFormFromStore()
  // 登录背景图以配置中心为唯一事实源（saveSiteInfo 持久化到后端）
  commonApi.getSiteInfo().then((res) => {
    if (res.login_bg_image !== undefined) form.loginBgImage = res.login_bg_image || ''
  }).catch(() => { /* 静默：保持默认 */ })
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
  form.navHeight = userStore.theme.navHeight ?? defaultTheme.navHeight
  form.headerHeight = userStore.theme.headerHeight ?? defaultTheme.headerHeight
  form.navTopBgColor = userStore.theme.navTopBgColor ?? defaultTheme.navTopBgColor
  form.navSideBgColor = userStore.theme.navSideBgColor ?? defaultTheme.navSideBgColor
  form.navTextSize = userStore.theme.navTextSize ?? defaultTheme.navTextSize
  form.navTextAuto = userStore.theme.navTextAuto ?? defaultTheme.navTextAuto
  form.navTextColor = userStore.theme.navTextColor ?? defaultTheme.navTextColor
  form.siteName = userStore.siteName
  form.siteDesc = userStore.siteDesc
  form.siteLogo = userStore.siteLogo
  form.favicon = userStore.favicon
  form.idleTimeout = userStore.idleTimeout
  form.appVersion = userStore.appVersion
}

// ---- 主题：表单 → store 即时预览；写入服务端只发生在「保存设置」或「导入」时 ----
const buildTheme = (): ThemeConfig => ({
  layout: form.layout,
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
  navHeight: form.navHeight,
  headerHeight: form.headerHeight,
  navTopBgColor: form.navTopBgColor,
  navSideBgColor: form.navSideBgColor,
  navTextSize: form.navTextSize,
  navTextAuto: form.navTextAuto,
  navTextColor: form.navTextColor,
})

const updateTheme = () => userStore.updateTheme(buildTheme())

// ---- 站点信息字段的即时副作用（仅 DOM 反馈，持久化统一由 saveAll 完成）----

const onSiteNameChange = () => { document.title = form.siteName }

const onSiteDescChange = () => {}
const onSiteLogoChange = () => {}

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

const onLoginBgChange = () => {}
const onIdleTimeoutChange = () => {}
const onAppVersionChange = () => {}

// ---- 保存：主题写入服务端（全站一份），站点信息写入配置中心 ----
const saving = ref(false)
const saveAll = async () => {
  saving.value = true
  try {
    await userStore.saveTheme(buildTheme())
    await commonApi.saveSiteInfo({
      site_name: form.siteName,
      site_desc: form.siteDesc,
      site_logo: form.siteLogo,
      favicon: form.favicon,
      idle_timeout: form.idleTimeout,
      login_bg_image: form.loginBgImage,
      app_version: form.appVersion,
    })
    // 同步 store 中的站点信息字段
    userStore.siteName = form.siteName
    userStore.siteDesc = form.siteDesc
    userStore.siteLogo = form.siteLogo
    userStore.favicon = form.favicon
    userStore.idleTimeout = form.idleTimeout
    userStore.appVersion = form.appVersion
    message.success('设置已保存')
  } catch { /* interceptor handles error */ }
  finally { saving.value = false }
}

// ---- 主题导入导出（仅外观主题，不含站点信息）----
const exportTheme = async () => {
  try {
    const file = await themeApi.exportTheme()
    downloadJsonFile('theme', file)
    message.success('主题配置已导出')
  } catch { /* interceptor handles error */ }
}

const confirmImportTheme = (data: unknown) => {
  Modal.confirm({
    title: '确认导入主题配置？',
    content: '文件中出现的主题项会覆盖当前全局主题，未出现的项保持不变；导入后对所有用户生效。',
    okText: '确认导入',
    cancelText: '取消',
    onOk: async () => {
      try {
        userStore.setTheme(await themeApi.importTheme(data))
        syncFormFromStore()
        message.success('主题配置已导入')
      } catch { /* interceptor handles error */ }
    },
  })
}

// 选择文件后先读取并二次确认；返回 false 阻止 a-upload 自动上传
const importTheme = (file: File): false => {
  readJsonFile(file).then(confirmImportTheme).catch((e: Error) => message.error(e.message))
  return false
}

</script>

<style scoped>
.theme-settings-inner {
  min-height: 200px;
}

.settings-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.settings-footer-hint {
  font-size: 12px;
  color: #999;
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
  /* 高度由 form.headerHeight 等比缩放后内联给出 */
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
