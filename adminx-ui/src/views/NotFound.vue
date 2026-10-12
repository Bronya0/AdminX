<template>
  <div class="not-found">
    <a-result
      status="404"
      title="404"
      :sub-title="subTitle"
    >
      <template #extra>
        <a-button type="primary" @click="$router.push('/')">
          返回首页
        </a-button>
      </template>
    </a-result>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
// 未知内部 path 的友好提示：多半是后端菜单已发布但前端尚无对应页面
const subTitle = computed(() =>
  `抱歉，“${route.path}”不存在，可能是该菜单尚未发布前端页面`
)
</script>

<style scoped>
.not-found {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f2f5;
}
</style>
