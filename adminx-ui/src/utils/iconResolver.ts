import * as Icons from '@ant-design/icons-vue'
import type { Component } from 'vue'

/**
 * 根据图标名称字符串动态解析为 @ant-design/icons-vue 组件。
 * Vite + ES modules 会 tree-shake 未使用的图标。
 * 支持简写（如 "Cloud" → "CloudOutlined"）
 */
export function resolveIcon(iconName?: string): Component {
  if (!iconName) return Icons.FileOutlined

  // 精确匹配
  const exact = (Icons as Record<string, Component>)[iconName]
  if (exact) return exact

  // 尝试添加 Outlined 后缀
  const withSuffix = (Icons as Record<string, Component>)[`${iconName}Outlined`]
  if (withSuffix) return withSuffix

  // 尝试添加 Filled 后缀
  const withFilled = (Icons as Record<string, Component>)[`${iconName}Filled`]
  if (withFilled) return withFilled

  return Icons.FileOutlined
}