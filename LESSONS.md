[gin] 2026-09-28 真：gin 默认信任全部代理，未调 SetTrustedProxies 时 XFF 可伪造 ClientIP。@internal/router/router.go
[sql] 2026-09-28 小心：列名 desc 是 PG 保留字，原始 SQL 里必须写 "desc"；PG 报 42601 而 sqlite 能跑@repository/*_repo.go
[service] 2026-09-28 小心：Update 用零值覆盖会清空未传字段（PATCH 只传 is_active 清空 trigger_config）。@job_service.go
[api] 2026-09-28 小心：allowed_paths 线上是字符串数组，发 JSON 字符串会被后端 400。@menu 权限页
[api] 2026-09-28 小心：前端表单字段名须与 DTO 同名，错名=静默空操作。@cluster_handler.go
[gorm] 2026-09-28 真：软删除行仍占唯一索引，查重前须 Unscoped()，否则 500。@user_repo.go
[jwt] 2026-09-28 真：iat 精度为秒，比 last_logout 前须 Truncate(time.Second)。@jwt.go
[tool] 2026-09-28 真：本仓库 gofmt 未全量清洁，勿跑 gofmt -w .（大量无关改动）。
[gin] 2026-09-28 真：c.FormFile 会把整个 body 先落临时盘，大文件上传须用 MultipartReader。@file_handler.go
[rbac] 2026-09-28 小心：菜单 path 为空时其 allowed_paths 不参与授权。@middleware/rbac.go
[job] 2026-09-28 真：trigger_config 解析/校验只有 internal/trigger 一处，新增解析必须接它。@internal/trigger
