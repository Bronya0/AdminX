<template>
  <div class="page-container">
    <a-card class="table-card">
      <a-tabs v-model:activeKey="activeTab" type="card">
        <a-tab-pane key="config" tab="字典配置">
          <ConfigList />
        </a-tab-pane>
        <a-tab-pane key="theme" tab="主题设置">
          <ThemeSettings />
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import ConfigList from './ConfigList.vue'
import ThemeSettings from '@/views/system/ThemeSettings.vue'

const route = useRoute()
const activeTab = ref(route.query.tab === 'theme' ? 'theme' : 'config')
</script>

<style scoped>
.page-container {
  padding: 24px;
}
.table-card {
  margin-bottom: 16px;
}
</style>
