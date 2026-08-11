import {
  FileOutlined, SettingOutlined, TeamOutlined, UserOutlined, ControlOutlined,
  ClusterOutlined, CloudServerOutlined, MonitorOutlined, BlockOutlined,
  ClockCircleOutlined, BellOutlined, AuditOutlined, FileSearchOutlined,
  LoginOutlined, DashboardOutlined, HomeOutlined, AppstoreOutlined,
  DatabaseOutlined, KeyOutlined, SafetyOutlined, ToolOutlined,
  CloudOutlined, FolderOutlined, LinkOutlined, GlobalOutlined,
  ApiOutlined, CodeOutlined, BugOutlined, ThunderboltOutlined, SkinOutlined,
} from '@ant-design/icons-vue'
import type { Component } from 'vue'

/**
 * 根据图标名称字符串解析为 @ant-design/icons-vue 组件。
 * 使用静态命名导入（白名单），保证 tree-shaking 生效，避免全量图标进包。
 * 支持简写（如 "Cloud" → "CloudOutlined"）。
 * 白名单之外的名称统一兜底为 FileOutlined。
 */
const iconMap: Record<string, Component> = {
  FileOutlined, SettingOutlined, TeamOutlined, UserOutlined, ControlOutlined,
  ClusterOutlined, CloudServerOutlined, MonitorOutlined, BlockOutlined,
  ClockCircleOutlined, BellOutlined, AuditOutlined, FileSearchOutlined,
  LoginOutlined, DashboardOutlined, HomeOutlined, AppstoreOutlined,
  DatabaseOutlined, KeyOutlined, SafetyOutlined, ToolOutlined,
  CloudOutlined, FolderOutlined, LinkOutlined, GlobalOutlined,
  ApiOutlined, CodeOutlined, BugOutlined, ThunderboltOutlined, SkinOutlined,
}

export function resolveIcon(iconName?: string): Component {
  if (!iconName) return FileOutlined

  // 精确匹配
  const exact = iconMap[iconName]
  if (exact) return exact

  // 尝试添加 Outlined 后缀（简写兼容）
  const withSuffix = iconMap[`${iconName}Outlined`]
  if (withSuffix) return withSuffix

  return FileOutlined
}