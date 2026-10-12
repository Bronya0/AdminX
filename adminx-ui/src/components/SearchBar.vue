<template>
  <div class="search-bar">
    <div class="search-bar-fields">
      <slot />
    </div>

    <a-space v-if="$slots.actions" class="search-bar-actions" :size="8">
      <slot name="actions" />
    </a-space>
  </div>
</template>

<script setup lang="ts">
// 列表页搜索栏的统一实现：左侧筛选字段等分可用宽度并可伸缩，右侧操作按钮固定宽度。
// 宽度不够时先压缩字段，而不是把按钮挤到第二行（1080p 下也能保持单行）。
//
// 用法：
//   <SearchBar>
//     <SearchField label="键"><a-input v-model:value="..." allow-clear /></SearchField>
//     <template #actions>
//       <a-button type="primary">查询</a-button>
//     </template>
//   </SearchBar>
</script>

<style scoped>
.search-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px 16px;
}

/* min-content 而不是写死像素：字段组的下限自动等于各字段最小宽度之和（含间距），
   字段多的页面自然获得更大的下限，字段少的页面不会白白占位。 */
.search-bar-fields {
  flex: 1 1 520px;
  min-width: min-content;
  display: flex;
  align-items: center;
  gap: 16px;
}

.search-bar-actions {
  flex: none;
  margin-left: auto;
}
</style>
