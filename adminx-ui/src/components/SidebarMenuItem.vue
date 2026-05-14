<template>
  <a-sub-menu v-if="item.children && item.children.length" :key="item.key">
    <template #icon>
      <component :is="getIcon(item.icon)" />
    </template>
    <template #title>{{ item.title }}</template>
    <SidebarMenuItem v-for="child in item.children" :key="child.key" :item="child" :getIcon="getIcon" />
  </a-sub-menu>
  <a-menu-item v-else :key="item.key">
    <template #icon>
      <component :is="getIcon(item.icon)" />
    </template>
    {{ item.title }}
  </a-menu-item>
</template>

<script setup lang="ts">
import type { SidebarItem } from '@/types'

defineProps<{
  item: SidebarItem
  getIcon: (icon?: string) => any
}>()
</script>
