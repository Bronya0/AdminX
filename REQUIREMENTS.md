# DjangoAdminX — 后端功能需求文档

> 所有后端功能已开发完成 ✅

---

## 1. 操作审计日志

### 需求
记录每次核心模型（User、Role、Menu、Config、ClusterNode 等）的详细变更，包括：
- 操作人、操作时间、IP 地址
- 操作类型（create / update / delete）
- 变更对象（Model + ID）
- 变更前和变更后的值（old → new）

### 实现方案
- `AuditLog` Model，存 operation/operator/object_repr/old_values/new_values
- 用 Django `post_save` / `pre_delete` 信号自动记录
- API 只读查看 + 搜索/筛选
- 管理端可查看，不可删除

---

## 2. 密码策略

### 需求
复盖安全基线：
- 最小长度、包含大小写/数字/特殊字符（已有 Django 基础校验）
- 密码过期时间（可配置，如 90 天）
- 历史密码禁止复用（记住最近 N 次）
- 首次登录强制修改密码

### 实现方案
- `PasswordPolicy` Model — 存规则配置（min_length, require_upper, require_lower, require_digit, require_special, expire_days, history_count）
- `PasswordHistory` Model — 存用户历史密码（hash）
- 密码修改时校验是否与历史匹配
- 登录时检查密码是否过期
- 不新增第三方库，用 `django.contrib.auth.hashers` 自带

---

## 3. 验证码

### 需求
- 登录验证码（可选开关，可配）
- 图形验证码或纯数字/字母

### 实现方案
- 自实现简单验证码（不引入第三方复杂库，追求轻量）
- 基于 `PIL` / `pillow` 生成图片，Redis 存储校验码
- 可选是否启用，通过配置中心控制
- 接口: `GET /api/common/captcha/` 获取验证码 → `POST /api/accounts/login/` 携带 captcha

---

## 4. 文件存储抽象

### 需求
- 统一文件上传接口
- 支持本地存储 + MinIO（对象存储）
- 上传后返回 URL，前端直接使用

### 实现方案
- `FileStorage` Model — 记录文件元信息（原名、大小、MIME、存储位置、URL）
- 存储后端抽象：`LocalStorageBackend` / `MinioStorageBackend`
- 通过配置中心 `FILE_STORAGE_BACKEND` 切换
- API: `POST /api/common/files/upload/` → 返回文件 ID + URL

---

## 5. 数据导入导出

### 需求
- 导出任意 Model 数据为 Excel（支持指定字段）
- 从 Excel 导入数据到指定 Model（按模板）

### 实现方案
- 基于 `openpyxl`（轻量，不需 pandas）
- 通用导出：`GET /api/common/export/?model=xxx&fields=a,b,c&format=excel`
- 通用导入：`POST /api/common/import/` + Excel 文件
- 导入前预览、字段映射
- 导入结果返回成功/失败条数

---

## 6. API 版本管理

### 需求
- `/api/v1/` 前缀路由
- 为后续 API 升级保留版本空间

### 实现方案
- `config/urls.py` 中增加 `v1/` 路由前缀
- 保持现有 API 不变，只是 URL 前缀变化
- 向后兼容：`/api/xxx` 重定向到 `/api/v1/xxx` 或共存

---

## 7. 缓存管理面板

### 需求
- API 查看当前缓存统计（keys、内存、命中率）
- 按前缀清理缓存
- 全量清理

### 实现方案
- 只读 API：缓存 keys 数量、内存占用
- 清理 API：`POST /api/common/cache/clear/` 带 `prefix` 参数
- 若 Redis 不可用，提示不可用

---

## 8. WSDL 发布（WebService 对外发布）

### 需求
- 对外发布 SOAP 接口，生成 WSDL
- 将 Django Model 数据通过 WebService 暴露出去

### 实现方案
- 用 `spyne` 发布 SOAP 服务，走 Django 路由
- 在 `/api/webservice/publish/` 挂载
- 支持 User、Menu、Config、Cluster 四个数据服务
- JWT 认证保护，WSDL 客户端可访问 `?wsdl` 获取

### 状态 ✅ 已完成

---

## 9. 审计日志完善 + Supervisor 配置

### 需求
- Supervisor / systemd 管理 Gunicorn + Scheduler 进程

### 实现方案
- 提供 supervisor 配置模板
- 提供 systemd service 文件模板
- 完善启动脚本

---

## 实现顺序（建议）

```
第一批（高优先级，进入可用状态） ✅
  1. 操作审计日志     — 安全底线 ✅
  2. 密码策略         — 安全基线 ✅
  3. 文件存储抽象     — 所有文件上传的基础 ✅
  4. API 版本管理     — 改路由前缀，改动小 ✅

第二批（功能完善） ✅
  5. 验证码           — 防暴力登录 ✅
  6. 数据导入导出     — 运维常用 ✅
  7. 缓存管理面板     — 在线运维 ✅

第三批（进阶） ✅
  8. WSDL 发布        — 特定场景需求 ✅
  9. Supervisor 配置  — 部署支持 ✅
```