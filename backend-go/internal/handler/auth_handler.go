// Package handler HTTP 处理层（薄封装，只做绑定/调用 service/格式化响应）。
package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/captcha"
	"adminx/internal/model"
	"adminx/internal/service"
)

// AuthHandler 认证相关 HTTP 处理器。
type AuthHandler struct {
	authSvc              *service.AuthService
	userSvc              *service.UserService
	menuSvc              *service.MenuService
	captcha              *captcha.Manager
	loginCaptchaRequired bool
	logger               *slog.Logger
}

func NewAuthHandler(authSvc *service.AuthService, userSvc *service.UserService, menuSvc *service.MenuService,
	captchaMgr *captcha.Manager, loginCaptchaRequired bool, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authSvc: authSvc, userSvc: userSvc, menuSvc: menuSvc,
		captcha: captchaMgr, loginCaptchaRequired: loginCaptchaRequired,
		logger: logger,
	}
}

// loginRequest 登录请求体。
type loginRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaText string `json:"captcha_text"`
}

// Login POST /accounts/login/
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}

	// 验证码校验：传入则必须有效；login_captcha_required 开启时必须传入。
	// 一次性消费（Verify 内部校验后删除），防止重放。
	if req.CaptchaID != "" || req.CaptchaText != "" || h.loginCaptchaRequired {
		if req.CaptchaID == "" || req.CaptchaText == "" {
			response.Fail(c, 400, "请输入验证码")
			return
		}
		if h.captcha == nil {
			// 未配置 Redis 时无法校验验证码：必填模式下拒绝登录（fail-closed），
			// 非必填模式下降级为不校验（与无 Redis 时整体降级策略一致）。
			if h.loginCaptchaRequired {
				response.Fail(c, 503, "验证码服务不可用，请配置 Redis")
				return
			}
		} else {
			ok, err := h.captcha.Verify(c.Request.Context(), req.CaptchaID, req.CaptchaText)
			if err != nil {
				h.logger.WarnContext(c.Request.Context(), "验证码校验失败", "error", err)
				response.Fail(c, 503, "验证码服务不可用")
				return
			}
			if !ok {
				response.Fail(c, 400, "验证码错误或已过期")
				return
			}
		}
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	result, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password, ip, ua)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	// 用 DTO 序列化 user（补 role_names / is_online）
	response.OK(c, gin.H{
		"access":  result.Access,
		"refresh": result.Refresh,
		"user":    model.ToUserDTO(result.User),
	})
}

// logoutRequest 登出请求体。
type logoutRequest struct {
	Refresh string `json:"refresh"`
}

// Logout POST /accounts/logout/
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req) // refresh 可选

	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	if err := h.authSvc.Logout(c.Request.Context(), uid, req.Refresh); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}

// refreshRequest 刷新 token 请求体。
type refreshRequest struct {
	Refresh string `json:"refresh" binding:"required"`
}

// Refresh POST /accounts/refresh/
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}

	result, err := h.authSvc.Refresh(c.Request.Context(), req.Refresh)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, result)
}

// introspectRequest token 校验请求体。
type introspectRequest struct {
	Token  string `json:"token" binding:"required"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

// Introspect POST /accounts/introspect/（供业务容器调用）
func (h *AuthHandler) Introspect(c *gin.Context) {
	var req introspectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}

	result, err := h.authSvc.Introspect(c.Request.Context(), req.Token)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, result)
}

// Me GET /accounts/users/me/ — 返回当前用户信息 + 权限 + 菜单。
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	user, err := h.userSvc.GetByID(uid)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}

	// 查用户菜单（含 allowed_paths）
	menus, err := h.menuSvc.MenusByUser(uid)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "查询用户菜单失败", "error", err)
		menus = []model.Menu{}
	}

	// 组装权限码（permission_code 非空）
	permissions := make([]string, 0, len(menus))
	for _, m := range menus {
		if m.PermissionCode != "" {
			permissions = append(permissions, m.PermissionCode)
		}
	}

	response.OK(c, gin.H{
		"user":        model.ToUserDTO(user),
		"permissions": permissions,
		"menus":       menus,
	})
}

// UpdateMe PATCH /accounts/users/me/ — 更新自己的资料。
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	var in service.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	// 不允许自己改 is_superuser / roles / password（改密必须走 change-password 接口校验旧密码）
	in.IsSuperuser = nil
	in.Roles = nil
	in.Password = ""

	user, err := h.userSvc.Update(uid, in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, model.ToUserDTO(user))
}

// LoginLogs GET /accounts/login-logs/ — 登录日志列表。
func (h *AuthHandler) LoginLogs(c *gin.Context) {
	// 委托给 userSvc 或直接查；这里复用 authSvc 的 logRepo
	// 为简洁，直接用 pagination 解析后调用 userSvc.ListLoginLogs
	p := pagination.Parse(c)
	// 简化: 通过 userSvc 暴露登录日志查询（实际委托 logRepo）
	logs, count, err := h.authSvc.ListLoginLogs(p.Offset(), p.Size, c.Query("username"), c.Query("ip"), nil)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, logs)
}
