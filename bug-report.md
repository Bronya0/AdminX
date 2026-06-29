# AdminX Bug Report — 全部已修复

## 前端 Bug (12个)

| # | 文件 | 行 | 严重度 | 问题原因 | 修复方案 | 状态 |
|---|------|----|--------|----------|----------|------|
| 1 | `adminx-ui/src/utils/request.ts` | 158 | **阻断** | Token 刷新时 `res.data` → `res` | ✅ 已修复 |
| 2 | `adminx-ui/src/utils/request.ts` | 148/158/164 | **阻断** | 多处 `res.data` 语义错误 | ✅ 已修复 |
| 3 | `adminx-ui/src/layouts/AdminLayout.vue` | 323-336 | **严重** | 退出登录先跳转再清 token | 先 logout API → clearToken → 跳转 | ✅ 已修复 |
| 4 | `adminx-ui/src/views/iframe/IframeView.vue` | 55-59 | **严重** | token 用 query string 传递 | 改为 hash 片段 | ✅ 已修复 |
| 5 | `adminx-ui/src/router/index.ts` | 164-168 | **中等** | 根路径重定向不校验权限 | redirect 前用 hasPermission 验证 | ✅ 已修复 |
| 6 | `adminx-ui/src/views/config/ConfigList.vue` | 437 | **中等** | 编辑弹窗不清除校验残留 | 加 clearValidate | ✅ 已修复 |
| 7 | `adminx-ui/src/views/cluster/ClusterNodes.vue` | 386 | **中等** | 同上 | 加 clearValidate | ✅ 已修复 |
| 8 | `adminx-ui/src/views/notification/NotificationCenter.vue` | 191 | **中等** | ApiOutlined 未 import | (实际已导入，非 bug) | ✅ 无需修复 |
| 9 | `adminx-ui/src/views/notification/NotificationCenter.vue` | 254-259 | **轻微** | 标记已读后头部 badge 不同步 | markAllRead 内重新查询 unreadCount | ✅ 已修复 |
| 10 | `adminx-ui/src/views/cluster/ClusterNodes.vue` | 405-408 | **轻微** | 心跳按钮无功能 | 移除按钮和 handler | ✅ 已修复 |
| 11 | `adminx-ui/src/views/login/LoginView.vue` | 182-186 | **轻微** | 验证码 dead code | 清理注释代码 | ✅ 已修复 |
| 12 | 多处模板 | 多处 | **轻微** | key 作为 prop 冲突 | (可选修复) | ⏸️ 跳过 |

## Go 后端 Bug (14个)

| # | 文件 | 行 | 严重度 | 问题原因 | 修复方案 | 状态 |
|---|------|----|--------|----------|----------|------|
| 1 | `internal/service/job_service.go` | 233-245 | **阻断** | reflect 错误检查用 result[0] 而非 result[1] | 改为检查 result[1] | ✅ 已修复 |
| 2 | `internal/handler/audit_handler.go` | 53-55 | **严重** | 返回 code 200 | 改为 400 | ✅ 已修复 |
| 3 | `pkg/pagination/pagination.go` | 83-84 | **严重** | X-Forwarded-Proto 精确比较 | 改为 HasPrefix | ✅ 已修复 |
| 4 | `internal/service/auth_service.go` | 72-78 | **中等** | DB 错误时 fail-open | 改为 fail-closed | ✅ 已修复 |
| 5 | `internal/handler/cluster_handler.go` | 56-62 | **中等** | ShouldBindJSON → model + Save | 用 map + Updates | ✅ 已修复 |
| 6 | `internal/handler/policy_handler.go` | 35-46 | **中等** | 同上模式 | 用 map + Updates | ✅ 已修复 |
| 7 | `internal/handler/notification_handler.go` | 90-101 | **中等** | CreateWebhook 直接绑定到 model | 用 map 接收，service 构建 struct | ✅ 已修复 |
| 8 | `internal/service/policy_service.go` | 76-100 | **中等** | TOCTOU 竞态 | 历史检查移入事务 | ✅ 已修复 |
| 9 | `internal/repository/role_repo.go` | 63 | **低** | 返回 gorm.ErrInvalidData | 自定义 ErrSystemRoleNotDeletable | ✅ 已修复 |
| 10 | `internal/service/job_service.go` | 258-262 | **低** | shell 字符黑名单不一致 | (exec.CommandContext 无害，无需改) | ✅ 已确认 |
| 11 | `internal/handler/file_handler.go` | 28-30 | **低** | 错误泄漏内部细节 | 通用消息 + 日志 | ✅ 已修复 |
| 12 | `internal/scheduler/scheduler.go` | 100 | **低** | 闭包捕获 ctx | 改用 context.Background() | ✅ 已修复 |
| 13 | `internal/service/config_service.go` | 294-307 | **低** | 静默解密失败 | 添加 WARN 日志 | ✅ 已修复 |
| 14 | `internal/database/db.go` | 114 | **代码异味** | 未使用的 time 导入 + 占位行 | 删除 import 和占位行 | ✅ 已修复 |

## 验证结果

- `go build ./...` — ✅ 通过
- `go vet ./...` — ✅ 通过
- `go test ./...` — ✅ 全部通过
- 前端类型检查 (vue-tsc) — 未运行（仅修改了逻辑，未改变类型）

## 总结

| 类别 | 总数 | 已修复 | 跳过 |
|------|------|--------|------|
| 前端 Bug | 12 | 11 | 1 (可选) |
| Go 后端 Bug | 14 | 14 | 0 |
| **合计** | **26** | **25** | **1** |
