// Package middleware — RBAC 权限中间件。
//
// 1. 超级管理员 → 放行所有
// 2. 普通用户 → 取其角色关联的菜单 allowed_paths（JSON glob 规则）匹配当前请求
// 3. 默认拒绝
//
// 规则格式: "METHOD:/path/pattern"，支持 glob 通配（path.Match）。
// 如 "GET:/api/v1/users/*" 或 "/api/v1/health"。
package middleware

import (
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"adminx/pkg/response"

	"adminx/internal/model"
	"adminx/internal/rbacrule"
)

// compiledCache glob 模式编译缓存（sync.Map，读多写少）。
// 只缓存“模式→正则”这个纯函数结果；权限规则本身不做进程内缓存（见 checkPermission）。
var compiledCache sync.Map // map[string]*regexp.Regexp

// warnedRules 已告警过的非法规则：同一条只报一次，避免每请求刷屏。
var warnedRules sync.Map // map[string]struct{}

// RBAC 权限校验中间件。
// 必须在 JWTAuth 之后使用（依赖 context 中的 user_id/is_superuser）。
// basePath 为部署前缀（如 /adminx）：种子规则按 API 路径（/api/v1/...）书写，
// 匹配前先剥掉部署前缀，避免换 base_path 部署时权限全量失配。
func RBAC(db *gorm.DB, basePath string) gin.HandlerFunc {
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

		requestPath := strings.TrimPrefix(c.Request.URL.Path, basePath)
		allowed, err := checkPermission(db, userID.(string), c.Request.Method, requestPath)
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
// checkPermission 每次直查库（不再做进程内用户规则缓存）：
// 停用角色/改菜单必须即时生效，不能有 30s 延迟窗口。
// 查询本身合并为单条 JOIN（原先两次查询已在 loadUserRules 内合并为“查 ID + 查规则”两次，
// 这里进一步合并为一次取规则原文），glob 编译仍走 compiledCache。
func checkPermission(db *gorm.DB, userID, method, requestPath string) (bool, error) {
	rules, err := loadUserRules(db, userID)
	if err != nil {
		return false, err
	}
	for _, rule := range rules {
		warnInvalidRule(rule)
		if matchPath(method, requestPath, rule) {
			return true, nil
		}
	}

	// 4. 检查三方业务命令的白名单
	// 当前阶段未实现 BusinessCommand 模型，后续补充
	return false, nil
}

// loadUserRules 从库加载用户的 allowed_paths 规则原文。
func loadUserRules(db *gorm.DB, userID string) ([]string, error) {
	// 1. 取用户角色关联的活跃菜单 ID
	// 菜单是权限单元，必须按 ID 取规则：path 只是前端路由地址、没有唯一约束，
	// 用 path 反查会把同 path 菜单的 allowed_paths 一并授予。
	var menuIDs []struct {
		ID int64
	}
	err := db.Model(&model.Menu{}).
		Distinct("menus.id").
		Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_menus.role_id").
		Joins("JOIN roles ON roles.id = role_menus.role_id").
		Where("user_roles.user_id = ? AND menus.is_active = ? AND roles.is_active = ? AND menus.path != ''", userID, true, true).
		Find(&menuIDs).Error
	if err != nil {
		return nil, err
	}
	if len(menuIDs) == 0 {
		return []string{}, nil
	}

	ids := make([]int64, 0, len(menuIDs))
	for _, m := range menuIDs {
		ids = append(ids, m.ID)
	}

	// 2. 查这些菜单的 allowed_paths（非空）
	var menus []model.Menu
	err = db.Where("is_active = ? AND id IN ? AND allowed_paths IS NOT NULL AND allowed_paths != '[]' AND allowed_paths != ''",
		true, ids).Find(&menus).Error
	if err != nil {
		return nil, err
	}

	// 3. 收集规则原文（匹配由调用方逐条做）
	rules := make([]string, 0)
	for _, menu := range menus {
		var rs []string
		if err := json.Unmarshal(menu.AllowedPaths, &rs); err != nil {
			continue
		}
		rules = append(rules, rs...)
	}
	return rules, nil
}

// warnInvalidRule 对静态可判定非法的规则记一条告警（同一条只报一次）。
// 这类规则不会让接口报错、只是永不匹配，表现为“菜单配了权限、用户仍然 403”；
// 写入侧（MenuService）已拦新数据，这里兼顾库里已有的历史脏规则。
func warnInvalidRule(rule string) {
	reason := rbacrule.InvalidReason(rule)
	if reason == "" {
		return
	}
	if _, loaded := warnedRules.LoadOrStore(rule, struct{}{}); !loaded {
		slog.Default().Warn("RBAC 规则格式非法，该规则永不匹配", "rule", rule, "reason", reason)
	}
}

// matchPath 匹配单条规则。
// 规则格式: 可选 "METHOD:/path"，path 部分用 glob（* 跨多级路径，? 单字符）。
// 解析统一走 rbacrule（与写入侧校验共用一份实现）。
// 用正则实现，使 `*` 能匹配多级路径（path.Match 的 * 不跨 /，
// 导致 "GET:/api/v1/posts/*" 匹配不到 /api/v1/posts/1/xx 这类多段路径）。
func matchPath(method, requestPath, rule string) bool {
	if rbacrule.InvalidReason(rule) != "" {
		return false
	}
	r := rbacrule.Parse(rule)
	if r.Path == "" {
		return false
	}

	// 方法不匹配则跳过
	if r.Method != "" && r.Method != strings.ToUpper(method) {
		return false
	}

	re, err := getCompiledRegex(r.Path)
	if err != nil {
		// 模式非法时降级为字面量比较
		return r.Path == requestPath
	}
	return re.MatchString(requestPath)
}

// getCompiledRegex 取 glob 模式的编译结果（命中缓存免重复编译）。
func getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if v, ok := compiledCache.Load(pattern); ok {
		if re, ok := v.(*regexp.Regexp); ok && re != nil {
			return re, nil
		}
	}
	re, err := globToRegex(pattern)
	if err != nil {
		return nil, err
	}
	compiledCache.Store(pattern, re)
	return re, nil
}

// globToRegex 将 glob 模式（* 任意多段、? 单字符）编译为正则。
// * 匹配任意字符（含 /），保证多级路径规则生效。
func globToRegex(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// CheckPermissionForTest 测试用权限检查入口（绕过 gin 中间件）。
// 外部包通过 interface{} 传入 *gorm.DB，规避 *gorm.DB 强制导入。
func CheckPermissionForTest(db interface{}, userID, method, requestPath string) (bool, error) {
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		return false, nil
	}
	return checkPermission(gormDB, userID, method, requestPath)
}
