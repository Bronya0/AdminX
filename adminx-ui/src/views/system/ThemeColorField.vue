<template>
  <div class="theme-color-field">
    <div class="theme-color-field-main">
      <a-input
        :value="shown"
        type="color"
        style="width: 48px; height: 36px; padding: 2px; cursor: pointer;"
        @change="onInputChange"
      />
      <span style="font-family: monospace;">{{ shown }}</span>
      <a-button size="small" @click="reset">恢复默认</a-button>
    </div>
    <div class="theme-color-field-presets">
      <div
        v-for="c in presets"
        :key="c"
        :style="{
          background: c,
          width: '28px',
          height: '28px',
          borderRadius: '6px',
          cursor: 'pointer',
          border: shown === c ? '3px solid #333' : '2px solid #e8e8e8',
        }"
        @click="pick(c)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

// 主题设置里「取色器 + 色块预设 + 恢复默认」这一组的唯一实现：
// 主题色、顶栏底色、侧栏底色、导航文字色都用它，避免四处抄同一段。
const props = withDefaults(
  defineProps<{
    modelValue: string
    presets: string[]
    /** 「恢复默认」写回的值 */
    defaultValue: string
    /** modelValue 为空时的展示值 */
    fallback?: string
  }>(),
  { fallback: '' }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()

const shown = computed(() => props.modelValue || props.fallback || props.defaultValue)

const pick = (color: string) => {
  emit('update:modelValue', color)
  emit('change', color)
}

const onInputChange = (e: Event) => {
  pick((e.target as HTMLInputElement).value)
}

const reset = () => pick(props.defaultValue)
</script>

<style scoped>
.theme-color-field-main {
  display: flex;
  align-items: center;
  gap: 16px;
}

.theme-color-field-presets {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
