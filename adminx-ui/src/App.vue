<template>
  <a-config-provider :theme="antTheme">
    <router-view />
  </a-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { theme } from 'ant-design-vue'

const userStore = useUserStore()

const antTheme = computed(() => ({
  algorithm: userStore.theme.isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: {
    colorPrimary: userStore.theme.primaryColor,
  },
}))

onMounted(() => {
  // 初始化暗色模式
  if (userStore.theme.isDark) {
    document.documentElement.classList.add('dark')
  }
})
</script>

<style>
/* 全局样式已在 main.css 中定义 */
</style>