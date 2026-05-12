# AdminX UI

基于 Vue3 + Ant Design Vue 4.x 的 DjangoAdminX 前端实现

## 技术栈

- **Vue 3.5+** - 渐进式 JavaScript 框架
- **TypeScript** - 类型安全的 JavaScript 超集
- **Ant Design Vue 4.x** - 企业级 UI 组件库
- **Vue Router 4** - 官方路由管理器
- **Pinia** - Vue 官方状态管理库
- **Axios** - HTTP 客户端
- **Vite** - 下一代前端构建工具

## 功能模块

### 已实现功能

1. **登录认证**
   - JWT Token 认证
   - 验证码支持（可选）
   - 登录状态持久化
   - 路由守卫

2. **用户管理**
   - 用户 CRUD
   - 角色分配
   - 状态管理
   - 密码重置

3. **角色权限管理**
   - 角色 CRUD
   - 权限分配（功能权限 + 菜单权限）
   - 权限树组件

4. **菜单配置中心**
   - 菜单树管理
   - 支持菜单/按钮/Iframe 类型
   - 拖拽排序
   - 权限绑定

5. **配置中心**
   - 配置项 CRUD
   - 支持多种类型：字符串、整数、布尔、JSON、加密、选项
   - 分组管理

6. **系统资源监控**
   - CPU、内存、磁盘实时监控
   - 网络 IO 统计
   - 网络连接状态
   - 自动刷新

7. **集群节点管理**
   - 节点 CRUD
   - 在线/离线状态监控
   - 主从节点角色
   - 集群概览

8. **主题配置**
   - 主题色切换
   - 暗黑模式
   - 侧边栏折叠
   - 布局配置

## 项目结构

```
adminx-ui/
├── src/
│   ├── api/              # API 接口封装
│   │   ├── auth.ts     # 认证相关 API
│   │   ├── menu.ts     # 菜单管理 API
│   │   ├── config.ts   # 配置中心 API
│   │   ├── cluster.ts  # 集群管理 API
│   │   └── monitor.ts  # 监控相关 API
│   ├── components/       # 公共组件
│   ├── layouts/          # 布局组件
│   │   └── AdminLayout.vue
│   ├── router/           # 路由配置
│   ├── stores/           # Pinia 状态管理
│   │   ├── index.ts
│   │   └── user.ts
│   ├── styles/           # 全局样式
│   ├── types/            # TypeScript 类型定义
│   ├── utils/            # 工具函数
│   │   └── request.ts    # Axios 封装
│   ├── views/            # 页面视图
│   │   ├── login/
│   │   ├── dashboard/
│   │   ├── system/       # 系统管理（用户、角色、菜单）
│   │   ├── config/       # 配置中心
│   │   ├── monitor/      # 系统监控
│   │   └── cluster/      # 集群管理
│   ├── App.vue
│   └── main.ts
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## 快速开始

### 安装依赖

```bash
cd adminx-ui
npm install
```

### 开发环境运行

```bash
npm run dev
```

默认端口：5173

### 构建生产环境

```bash
npm run build
```

### 类型检查

```bash
npm run type-check
```

## 后端 API 配置

在 `vite.config.ts` 中配置代理：

```typescript
server: {
  port: 5173,
  proxy: {
    '/api': {
      target: 'http://localhost:8000',  // Django 后端地址
      changeOrigin: true,
    },
  },
}
```

## 环境变量

创建 `.env` 文件：

```
VITE_API_BASE_URL=/api/v1
```

## 主要特性

### 1. 响应式设计
- 支持桌面端和移动端
- 自适应布局

### 2. 权限控制
- 基于角色的访问控制（RBAC）
- 菜单级权限
- 按钮级权限

### 3. 主题定制
- 支持主题色切换
- 暗黑模式
- 布局配置持久化

### 4. 代码规范
- TypeScript 严格类型检查
- ESLint + Prettier 代码格式化
- 组件化开发

## API 接口对应

前端 API 与 Django 后端接口对应关系：

| 前端 API | 后端接口 |
|---------|---------|
| `/api/auth.ts` | `djangoadminx/accounts/views.py` |
| `/api/menu.ts` | `djangoadminx/menu/views.py` |
| `/api/config.ts` | `djangoadminx/config_center/views.py` |
| `/api/cluster.ts` | `djangoadminx/cluster/views.py` |
| `/api/monitor.ts` | `djangoadminx/monitor/views.py` |

## 开发规范

1. **组件命名**：使用 PascalCase，如 `UserList.vue`
2. **API 封装**：统一放在 `api/` 目录，按模块分类
3. **类型定义**：所有接口和数据结构在 `types/index.ts` 中定义
4. **状态管理**：使用 Pinia，按功能模块拆分
5. **路由配置**：在 `router/index.ts` 中集中管理

## 浏览器支持

- Chrome >= 88
- Firefox >= 78
- Safari >= 14
- Edge >= 88

## 许可证

MIT
