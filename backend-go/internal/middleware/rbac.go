// Package middleware — RBAC 权限中间件。
//
// 对齐 Django RBACPermission:
// 1. 超级管理员 → 放行所有
// 2. 普通用户 → 取其角色关联的菜单 allowed_paths（JSON glob 规则）匹配当前请求
// 3. 默认拒绝
//
// 规则格式: "METHOD:/path/pattern"，支持 glob 通配（path.Match）。
// 如 "GET:/api/v1/users/*" 或 "/api/v1/health"。
package middleware

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"djangoadminx/pkg/response"

	"djangoadminx/internal/model"
)

// RBAC 权限校验中间件。
// 必须在 JWTAuth 之后使用（依赖 context 中的 user_id/is_superuser）。
func RBAC(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 超级管理员放行
		if isSuper, exists := c.Get("is_superuser"); exists {
			if isSuperBool, ok := isSuper.(bool); ok && isSuperBool {
				c.Next()
				return
			}
		}

		userID, _ := c.Get("user_id")
		if userID == nil {
			response.Fail(c, 401, "未认证")
			c.Abort()
			return
		}

		allowed, err := checkPermission(db, userID.(string), c.Request.Method, c.Request.URL.Path)
		if err != nil {
			response.Fail(c, 500, "权限校验失败")
			c.Abort()
			return
		}
		if !allowed {
			response.Fail(c, 403, "无权限访问此资源")
			c.Abort()
			return
		}
		c.Next()
	}
}

// checkPermission 查询用户角色关联菜单的 allowed_paths，glob 匹配请求。
// 对齐 Django permissions.py: superuser 放行；否则查 role→menu→allowed_paths。
func checkPermission(db *gorm.DB, userID, method, requestPath string) (bool, error) {
	// 1. 取用户角色关联的活跃菜单的 path 集合
	var menuPaths []struct {
		Path string
	}
	err := db.Model(&model.Menu{}).
		Distinct("menus.path").
		Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_menus.role_id").
		Where("user_roles.user_id = ? AND menus.is_active = ? AND menus.path != ''", userID, true).
		Find(&menuPaths).Error
	if err != nil {
		return false, err
	}
	if len(menuPaths) == 0 {
		return false, nil
	}

	pathSet := make(map[string]bool, len(menuPaths))
	for _, mp := range menuPaths {
		pathSet[mp.Path] = true
	}

	// 2. 查这些菜单的 allowed_paths（非空）
	var menus []model.Menu
	err = db.Where("is_active = ? AND path IN ? AND allowed_paths IS NOT NULL AND allowed_paths != '[]' AND allowed_paths != ''",
		true, keys(pathSet)).Find(&menus).Error
	if err != nil {
		return false, err
	}

	// 3. 逐条匹配 allowed_paths 规则
	for _, menu := range menus {
		var rules []string
		if err := json.Unmarshal(menu.AllowedPaths, &rules); err != nil {
			continue
		}
		for _, rule := range rules {
			if matchPath(method, requestPath, rule) {
				return true, nil
			}
		}
	}

	// 4. 检查三方业务命令的白名单（对齐 Django BusinessCommand 逻辑）
	// 当前阶段未实现 BusinessCommand 模型，后续补充
	return false, nil
}

// matchPath 匹配单条规则。对齐 Django _match_path。
// 规则格式: 可选 "METHOD:/path"，path 部分用 glob（path.Match）。
func matchPath(method, requestPath, rule string) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return false
	}

	ruleMethod := ""
	rulePath := rule
	if idx := strings.Index(rule, ":"); idx >= 0 {
		ruleMethod = strings.ToUpper(rule[:idx])
		rulePath = strings.TrimSpace(rule[idx+1:])
	}

	// 方法不匹配则跳过
	if ruleMethod != "" && ruleMethod != strings.ToUpper(method) {
		return false
	}

	// glob 匹配（对齐 Python fnmatch）
	matched, err := path.Match(rulePath, requestPath)
	if err != nil {
		return false
	}
	return matched
}

// keys 提取 map 的 key 切片。
func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
