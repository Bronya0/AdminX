<template>
  <label class="search-field">
    <span class="search-field-label">{{ label }}</span>
    <slot />
  </label>
</template>

<script setup lang="ts">
// 搜索栏（SearchBar）里的一个筛选字段：等分可用宽度，内部的输入框/下拉自动撑满，
// 页面里不再各自写死控件宽度。
defineProps<{
  label: string
}>()
</script>

<style scoped>
.search-field {
  flex: 1 1 0;
  min-width: 90px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-field-label {
  flex: none;
  white-space: nowrap;
}

.search-field :deep(.ant-input),
.search-field :deep(.ant-input-affix-wrapper),
.search-field :deep(.ant-input-search),
.search-field :deep(.ant-input-number),
.search-field :deep(.ant-picker),
.search-field :deep(.ant-select) {
  width: 100%;
}
</style>
