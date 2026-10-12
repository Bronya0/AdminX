<template>
  <a-config-provider :locale="zhCN" :theme="antTheme">
    <router-view />
  </a-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useUserStore } from '@/stores/user'
import { theme } from 'ant-design-vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'

const userStore = useUserStore()

const antTheme = computed(() => ({
  algorithm: userStore.theme.isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: {
    colorPrimary: userStore.theme.primaryColor,
  },
}))

// 主题在文档根节点上的副作用只在这里应用：首次渲染、远端拉取、设置页修改、导入都走同一条路径
const applyDocumentTheme = () => {
  const { isDark, primaryColor, fontSize } = userStore.theme
  const root = document.documentElement
  root.classList.toggle('dark', isDark)
  root.style.setProperty('--primary-color', primaryColor)
  // 14px 为默认字号：不写内联样式，保持样式表原有行为
  root.style.fontSize = fontSize === 14 ? '' : `${fontSize}px`
}

watch(
  () => [userStore.theme.isDark, userStore.theme.primaryColor, userStore.theme.fontSize],
  applyDocumentTheme,
  { immediate: true },
)

onMounted(() => {
  // 主题以服务端为准：挂载后拉取一次，本地缓存只用于首屏
  userStore.fetchTheme()
})
</script>

<style>
/* 全局样式已在 main.css 中定义 */
</style>