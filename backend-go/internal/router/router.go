// Package router 注册所有 HTTP 路由，对齐 Django 的 urls.py 路由树。
//
// 路由前缀: /djangoadminx/api/v1/（对齐 Django FORCE_SCRIPT_NAME）。
// 完整覆盖阶段 1-6 所有模块。
package router

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"djangoadminx/internal/config"
	"djangoadminx/internal/handler"
	"djangoadminx/internal/jwt"
	appmw "djangoadminx/internal/middleware"
	"djangoadminx/pkg/response"
)

// Deps 路由依赖（依赖注入）。
type Deps struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client

	// 所有 handler
	AuthH   *handler.AuthHandler
	UserH   *handler.UserHandler
	RoleH   *handler.RoleHandler
	MenuH   *handler.MenuHandler
	CfgH    *handler.ConfigHandler
	AuditH  *handler.AuditHandler
	JobH    *handler.JobHandler
	NotifH  *handler.NotificationHandler
	ClsH    *handler.ClusterHandler
	FileH   *handler.FileHandler
	MonH    *handler.CommonHandler // monitor + dashboard + health
	PolicyH *handler.PolicyHandler
	CapH    *handler.CaptchaHandler
	WSH     *handler.WSHandler

	JWTCfg JWTConfig
}

// JWTConfig JWT 鉴权中间件依赖。
type JWTConfig struct {
	Secret        string
	Issuer        string
	AccessExpire  time.Duration
	RefreshExpire time.Duration
}

// New 构造 gin Engine 并注册所有路由。
func New(deps *Deps) *gin.Engine {
	if deps.Config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if deps.Config.Security.TrustProxyHeaders {
		_ = r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	}

	r.Use(gin.Recovery())
	r.Use(appmw.RequestID())
	r.Use(appmw.RequestLog())
	r.Use(corsMiddleware(deps.Config))

	base := deps.Config.Server.BasePath
	api := r.Group(base + "/api/v1")

	// ── 公共端点（无鉴权）──
	public := api.Group("")
	{
		public.GET("/common/health/", healthHandler(deps.DB))
		public.GET("/common/ping/", func(c *gin.Context) {
			response.OK(c, gin.H{"message": "pong"})
		})
		// captcha（无鉴权）
		public.GET("/captcha/captcha/", deps.CapH.Generate)
		public.POST("/captcha/captcha/verify/", deps.CapH.Verify)

		// cluster 业务组件注册/心跳（AllowAny，对齐 Django）
		if deps.ClsH != nil {
			public.POST("/cluster/components/register/", deps.ClsH.Register)
			public.POST("/cluster/components/heartbeat/", deps.ClsH.Heartbeat)
		}
	}

	// ── accounts 公共路由（登录/刷新/introspect，限流）──
	accountsPublic := api.Group("/accounts")
	accountsPublic.Use(appmw.RateLimit(deps.Redis, "anon", appmw.Limit{Requests: 30, Window: time.Minute}))
	{
		accountsPublic.POST("/login/", deps.AuthH.Login)
		accountsPublic.POST("/refresh/", deps.AuthH.Refresh)
		accountsPublic.POST("/introspect/",
			appmw.RateLimit(deps.Redis, "introspect", appmw.Limit{Requests: 300, Window: time.Minute}),
			deps.AuthH.Introspect)
	}

	// ── JWT 鉴权中间件工厂 ──
	jwtMW := func() gin.HandlerFunc {
		mgr := jwt.New(deps.JWTCfg.Secret, deps.JWTCfg.Issuer,
			deps.JWTCfg.AccessExpire, deps.JWTCfg.RefreshExpire, deps.Redis)
		return appmw.JWTAuth(mgr, deps.DB)
	}

	// ── me 路由（仅 JWT，无 RBAC）──
	me := api.Group("")
	me.Use(jwtMW())
	{
		me.GET("/accounts/users/me/", deps.AuthH.Me)
		me.PATCH("/accounts/users/me/", deps.AuthH.UpdateMe)
		me.POST("/accounts/logout/", deps.AuthH.Logout)
		me.GET("/accounts/login-logs/", deps.AuthH.LoginLogs)
		me.GET("/menu/user_tree/", deps.MenuH.UserTree)
		me.GET("/accounts/roles/all/", deps.RoleH.All)
		me.GET("/notification/messages/", deps.NotifH.List)
		me.POST("/notification/messages/:id/mark_read/", deps.NotifH.MarkRead)
		me.POST("/notification/messages/mark_all_read/", deps.NotifH.MarkAllRead)
		me.GET("/notification/messages/unread_count/", deps.NotifH.UnreadCount)
		me.GET("/policy/change-password-view/", func(c *gin.Context) {
			response.OK(c, gin.H{"note": "POST /policy/change-password/"})
		})
		me.POST("/policy/change-password/", deps.PolicyH.ChangePassword)
		me.GET("/config/by_group/", deps.CfgH.ByGroup)
		me.GET("/config/get_value/", deps.CfgH.GetValue)
	}

	// ── RBAC 路由（JWT + RBAC）──
	rbac := api.Group("")
	rbac.Use(jwtMW(), appmw.RBAC(deps.DB))
	{
		// 用户
		rbac.GET("/accounts/users/", deps.UserH.List)
		rbac.GET("/accounts/users/:id/", deps.UserH.Get)
		rbac.POST("/accounts/users/", deps.UserH.Create)
		rbac.PUT("/accounts/users/:id/", deps.UserH.Update)
		rbac.PATCH("/accounts/users/:id/", deps.UserH.Update)
		rbac.DELETE("/accounts/users/:id/", deps.UserH.Delete)

		// 角色
		rbac.GET("/accounts/roles/", deps.RoleH.List)
		rbac.GET("/accounts/roles/:id/", deps.RoleH.Get)
		rbac.POST("/accounts/roles/", deps.RoleH.Create)
		rbac.PUT("/accounts/roles/:id/", deps.RoleH.Update)
		rbac.PATCH("/accounts/roles/:id/", deps.RoleH.Update)
		rbac.DELETE("/accounts/roles/:id/", deps.RoleH.Delete)

		// 菜单
		rbac.GET("/menu/", deps.MenuH.List)
		rbac.GET("/menu/:id/", deps.MenuH.Get)
		rbac.POST("/menu/", deps.MenuH.Create)
		rbac.PUT("/menu/:id/", deps.MenuH.Update)
		rbac.PATCH("/menu/:id/", deps.MenuH.Update)
		rbac.DELETE("/menu/:id/", deps.MenuH.Delete)

		// 配置中心
		rbac.GET("/config/", deps.CfgH.List)
		rbac.GET("/config/:id/", deps.CfgH.Get)
		rbac.POST("/config/", deps.CfgH.Create)
		rbac.PUT("/config/:id/", deps.CfgH.Update)
		rbac.PATCH("/config/:id/", deps.CfgH.Update)
		rbac.DELETE("/config/:id/", deps.CfgH.Delete)
		rbac.GET("/config/groups/", deps.CfgH.Groups)

		// 审计
		rbac.GET("/audit/", deps.AuditH.List)
		rbac.GET("/audit/:id/", deps.AuditH.Get)

		// 定时任务
		rbac.GET("/jobs/", deps.JobH.List)
		rbac.GET("/jobs/:id/", deps.JobH.Get)
		rbac.POST("/jobs/", deps.JobH.Create)
		rbac.PUT("/jobs/:id/", deps.JobH.Update)
		rbac.PATCH("/jobs/:id/", deps.JobH.Update)
		rbac.DELETE("/jobs/:id/", deps.JobH.Delete)
		rbac.POST("/jobs/:id/run_once/", deps.JobH.RunOnce)
		rbac.GET("/jobs/status/", deps.JobH.Status)
		rbac.GET("/jobs/logs/", deps.JobH.Logs)

		// 通知 webhook 管理
		rbac.GET("/notification/webhooks/", deps.NotifH.ListWebhooks)
		rbac.POST("/notification/webhooks/", deps.NotifH.CreateWebhook)
		rbac.PUT("/notification/webhooks/:id/", deps.NotifH.UpdateWebhook)
		rbac.PATCH("/notification/webhooks/:id/", deps.NotifH.UpdateWebhook)
		rbac.DELETE("/notification/webhooks/:id/", deps.NotifH.DeleteWebhook)
		rbac.GET("/notification/webhook-logs/", deps.NotifH.WebhookLogs)

		// 集群节点 + 业务组件管理
		rbac.GET("/cluster/nodes/", deps.ClsH.ListNodes)
		rbac.POST("/cluster/nodes/", deps.ClsH.CreateNode)
		rbac.PUT("/cluster/nodes/:id/", deps.ClsH.UpdateNode)
		rbac.PATCH("/cluster/nodes/:id/", deps.ClsH.UpdateNode)
		rbac.DELETE("/cluster/nodes/:id/", deps.ClsH.DeleteNode)
		rbac.GET("/cluster/components/", deps.ClsH.ListComponents)
		rbac.POST("/cluster/components/:id/set_upgrade/", deps.ClsH.SetUpgrade)
		rbac.POST("/cluster/components/:id/cancel_upgrade/", deps.ClsH.CancelUpgrade)
		rbac.POST("/cluster/components/:id/set_uninstall/", deps.ClsH.SetUninstall)
		rbac.POST("/cluster/components/:id/cancel_uninstall/", deps.ClsH.CancelUninstall)

		// 文件中心
		rbac.POST("/files/upload/", deps.FileH.Upload)
		rbac.GET("/files/records/", deps.FileH.Records)
		rbac.DELETE("/files/records/:id/", deps.FileH.Delete)

		// 系统监控
		rbac.GET("/monitor/resources/", deps.MonH.Monitor)
		rbac.GET("/monitor/netstat/", deps.MonH.NetStat)

		// 仪表盘
		rbac.GET("/common/dashboard/stats/", deps.MonH.Dashboard)

		// 密码策略管理（仅管理员）
		rbac.GET("/policy/policy/", deps.PolicyH.GetPolicy)
		rbac.PUT("/policy/policy/", deps.PolicyH.UpdatePolicy)
		rbac.PATCH("/policy/policy/", deps.PolicyH.UpdatePolicy)
	}

	// ── WebSocket（无 REST 鉴权，连接时可选校验 token）──
	if deps.WSH != nil {
		r.GET(base+"/ws/log/", deps.WSH.Log)
	}

	return r
}

func healthHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbOK := "up"
		if db != nil {
			sqlDB, err := db.DB()
			if err != nil || sqlDB.Ping() != nil {
				dbOK = "down"
			}
		}
		response.OK(c, gin.H{
			"status":    "ok",
			"db":        dbOK,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	if len(cfg.Security.CORSAllowedOrigins) == 0 {
		cfg.Security.CORSAllowedOrigins = []string{"*"}
	}

	// CORS 规范禁止 AllowOrigins: ["*"] 与 AllowCredentials: true 同时使用，
	// 否则浏览器会拒绝响应；gin-contrib/cors 在此场景下也会 panic。
	// 当配置为通配 "*" 时，关闭 credentials 并改用 echo 模式回显 Origin。
	wildcard := len(cfg.Security.CORSAllowedOrigins) == 1 && cfg.Security.CORSAllowedOrigins[0] == "*"
	corsCfg := cors.Config{
		AllowMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders: []string{"Content-Length", "Content-Disposition"},
		MaxAge:        12 * time.Hour,
	}
	if wildcard {
		corsCfg.AllowAllOrigins = true
		corsCfg.AllowCredentials = false
	} else {
		corsCfg.AllowOrigins = cfg.Security.CORSAllowedOrigins
		corsCfg.AllowCredentials = true
	}
	return cors.New(corsCfg)
}

var _ = http.StatusOK
