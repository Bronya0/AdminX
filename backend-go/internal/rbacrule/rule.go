// Package rbacrule — RBAC 接口路径白名单规则的解析与校验（唯一实现）。
//
// 规则格式：可选 "METHOD:" 前缀 + 以 "/" 开头的路径，路径部分支持 glob
// （* 跨多级路径、? 单字符）。例："GET:/api/v1/users/*"、"/api/v1/health"。
//
// 解析与校验必须共用这一份：middleware 用它匹配请求、service 用它校验写入。
// 两边各写一份的下场是「接口校验通过、运行时永不匹配」——菜单配了权限、
// 用户仍然 403，且没有任何报错提示，只能靠人肉比对规则字符串。
package rbacrule

import (
	"strings"
	"unicode"
)

// methods 允许出现在 "METHOD:" 前缀里的 HTTP 方法（大写）。
var methods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"HEAD":    true,
	"OPTIONS": true,
}

// Rule 拆分后的单条规则。
type Rule struct {
	Raw    string // 原始规则（已 TrimSpace）
	Method string // 方法前缀（大写），空串表示不限方法
	Path   string // 路径部分（glob 模式）
}

// Parse 按第一个冒号拆分方法前缀与路径；无冒号时整条规则都是路径。
//
// 与匹配行为保持一致：只要出现冒号就按「冒号前是方法」理解。
// 因此把路径参数写成 ":id" 的规则（如 /api/v1/x/:id/）会被判成方法非法 ——
// 匹配侧同样取不到正确路径，这种规则本来就永不生效。
func Parse(raw string) Rule {
	rule := strings.TrimSpace(raw)
	r := Rule{Raw: rule, Path: rule}
	if idx := strings.Index(rule, ":"); idx >= 0 {
		r.Method = strings.ToUpper(strings.TrimSpace(rule[:idx]))
		r.Path = strings.TrimSpace(rule[idx+1:])
	}
	return r
}

// InvalidReason 返回规则不合法的原因；合法时返回空串。
//
// 只拦「能静态判定必然失效」的写法（方法名拼错、冒号写成空格、路径不以 / 开头），
// 不拦合法的少见写法（如正则元字符会被匹配侧转义成字面量）。
func InvalidReason(raw string) string {
	rule := strings.TrimSpace(raw)
	if rule == "" {
		return "" // 空行由调用方过滤
	}
	if strings.IndexFunc(rule, unicode.IsSpace) >= 0 {
		return "不能含空格（方法与路径之间用冒号连接，如 GET:/api/v1/x/*）"
	}
	r := Parse(rule)
	if strings.Contains(rule, ":") && !methods[r.Method] {
		return "方法前缀无效（只支持 GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS；路径参数请用 * 通配，不要写 :id）"
	}
	if !strings.HasPrefix(r.Path, "/") {
		return "路径必须以 / 开头"
	}
	return ""
}
