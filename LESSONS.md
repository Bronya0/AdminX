[gin] 2026-09-28 真：gin 默认信任全部代理，未调 SetTrustedProxies 时 XFF 可伪造 ClientIP。@internal/router/router.go
[sql] 2026-09-28 小心：列名 desc 是 PG 保留字，原始 SQL 里必须写 "desc"；PG 报 42601 而 sqlite 能跑@repository/*_repo.go
[service] 2026-09-28 小心：Update 用零值覆盖会清空未传字段（PATCH 只传 is_active 清空 trigger_config）。@job_service.go
[api] 2026-09-28 小心：allowed_paths 线上是字符串数组，发 JSON 字符串会被后端 400。@menu 权限页
[api] 2026-09-28 小心：前端表单字段名须与 DTO 同名，错名=静默空操作。@cluster_handler.go
[gorm] 2026-09-28 真：软删除行仍占唯一索引，查重前须 Unscoped()，否则 500。@user_repo.go
[jwt] 2026-09-28 真：iat 精度为秒，比 last_logout 前须 Truncate(time.Second)。@jwt.go
[tool] 2026-09-28 真：本仓库 gofmt 未全量清洁，勿跑 gofmt -w .（大量无关改动）。
[gin] 2026-09-28 真：c.FormFile 会把整个 body 先落临时盘，大文件上传须用 MultipartReader。@file_handler.go
[rbac] 2026-10-12 高危：权限规则不做进程内缓存，停用角色/改菜单必须即时生效，TTL 缓存=30s 越权窗口。@middleware/rbac.go
[captcha] 2026-10-12 高危：验证码不能用 SVG 输出（文本格式，答案明文在节点里，抓一次响应即可读出），必须出 PNG；答案也要 crypto/rand，库内 RandText 是 math/rand。@internal/captcha
[job] 2026-09-28 真：trigger_config 解析/校验只有 internal/trigger 一处，新增解析必须接它。@internal/trigger
[cors] 2026-10-12 小心：POST 必带 Origin，不在白名单即被 gin-contrib/cors 空体 403，前端显示“没有权限”，掩盖真实错误。@configs/config.example.yaml
[gin] 2026-10-12 真：c.JSON 不防重复写，中间件复用会写响应的解析函数会拼出两段 JSON（软鉴权只能解析、不能应答）。@middleware/jwt.go
[gorm] 2026-10-12 真：读整行改完再 Save 会覆盖并发更新（心跳抹掉升级指令）；单字段更新一律按列 Update。@cluster_service.go
[前端] 2026-10-12 小心：筛选控件必须对应后端真实 query 参数，gin 丢弃未知字段=静默查全量；后端不支持的控件要删掉。@adminx-ui/src/views
[前端] 2026-10-12 高危：编辑弹窗提交整个表单对象会带上新增流程残留的 password → 后端改密并吊销该用户全部 token。@views/system/UserList.vue
[前端] 2026-10-12 中危：派生状态只在某个事件回调里刷新（MIX 选中一级菜单只在路由变化时写），触发条件不成立即静默失效；一律用 computed 推导。@AdminLayout.vue
[前端] 2026-10-12 中危：主区放进不同 v-if 分支会让 <router-view> 被卸载重建，页面 onMounted 重拉服务端状态覆盖未保存的编辑（点混合布局像没反应）。@AdminLayout.vue
[sqlite] 2026-10-13 中危：请求路径上新增写库（清理/采样）在单写者下撞第二进程会 database is locked 拖 5s 并 500，家务写须限频且失败不致命。@internal/captcha
[audit] 2026-10-13 高危：写端点漏审计是静默的（改配置/执行 shell 任务无痕）；新增写端点必须在 router 用 audited() 声明。@internal/router/router.go
[前端] 2026-10-13 中危：跨仓库搬修复前核对后端契约，parent_id/job_id/unread 等与 Django 的 parent/job 不等价，照搬=静默丢参。@frontend-prod
[前端] 2026-10-19 高危：axios 拦截器把非 JSON 的 404 原样透传成成功，前端调不存在的接口照样 resolve，界面提示成功。@utils/request.ts
[前端] 2026-10-19 中危：菜单树只按 path 前缀找父，外链/iframe 的绝对 URL 拼出 /https: 前缀，整棵子树挂到树根；parent_id 优先。@utils/menuTree.ts
[init-data] 2026-10-19 中危：种子无条件写 is_active/is_visible=true，运维应急停用的菜单跑一次 init-data 就复活。@cmd/init-data/main.go
[sqlite] 2026-10-19 高危：手工写 SQLite 时间列必须带 +08:00 时区，否则 Go 按 UTC 解析，last_logout 变未来时间，全部 token 立即失效，登录后被踢回登录页。@users.last_logout
[sql] 2026-10-19 中危：LIKE 过了 EscapeLike 却不加 ESCAPE '\'，SQLite 把 \_ 当字面量，带下划线的搜索在本地全 0 条（PG 正常）。@repository/*_repo.go
[前端] 2026-10-19 中危：rc-table 树形只判 children 字段是否存在（空数组为真），叶子输出 children:[] 会多出展开图标。@utils/menuTree.ts
