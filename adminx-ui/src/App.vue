<template>
  <router-view />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

onMounted(() => {
  // 初始化主题颜色
  if (userStore.theme.primaryColor) {
    document.documentElement.style.setProperty('--ant-primary-color', userStore.theme.primaryColor)
  }
})
</script>

<style>
/* 全局样式已在 main.css 中定义 */
</style>
