<template>
  <div class="table-toolbar">
    <div class="table-toolbar-left">
      <slot />
    </div>

    <div v-if="$slots.right" class="table-toolbar-right">
      <slot name="right" />
    </div>
  </div>
</template>

<script setup lang="ts">
// 表格上方的操作条：左侧放操作按钮与筛选控件，右侧放状态标签或次要操作。
// 各列表页共用的唯一实现 —— 间距、换行、左右分组只在这里定义一次。
//
// 用法：
//   <TableToolbar>
//     <a-button type="primary">新建</a-button>
//     <template #right><a-tag>状态</a-tag></template>
//   </TableToolbar>
</script>

<style scoped>
.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px 16px;
  margin-bottom: 16px;
}

/* 左右两组各自成行内弹性组：窄窗口下组内换行，而不是把整组挤到下一行 */
.table-toolbar-left,
.table-toolbar-right {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
