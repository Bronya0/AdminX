package rbacrule

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		raw        string
		wantMethod string
		wantPath   string
	}{
		{"GET:/api/v1/users/*", "GET", "/api/v1/users/*"},
		{"get:/api/v1/users/*", "GET", "/api/v1/users/*"},
		{"GET: /api/v1/users/*", "GET", "/api/v1/users/*"},
		{"/api/v1/health", "", "/api/v1/health"},
		{"  /api/v1/health  ", "", "/api/v1/health"},
		{"", "", ""},
	}
	for _, c := range cases {
		got := Parse(c.raw)
		if got.Method != c.wantMethod || got.Path != c.wantPath {
			t.Errorf("Parse(%q) = (%q, %q), want (%q, %q)",
				c.raw, got.Method, got.Path, c.wantMethod, c.wantPath)
		}
	}
}

func TestInvalidReason(t *testing.T) {
	// 合法写法（含种子里的实际值）不应被判非法
	valid := []string{
		"GET:/api/v1/common/dashboard/*",
		"/api/v1/config/*",
		"POST:/api/v1/accounts/roles/accessible_menus/*",
		"GET:/api/v1/posts",
		"DELETE:/api/v1/files/records/*",
		"", // 空行由调用方过滤
	}
	for _, rule := range valid {
		if reason := InvalidReason(rule); reason != "" {
			t.Errorf("InvalidReason(%q) = %q, want 合法", rule, reason)
		}
	}

	// 静态可判定必然失效的写法
	invalid := []string{
		"GTE:/api/v1/users/*", // 方法名拼错：永不匹配
		"GET /api/v1/users/*", // 冒号写成空格
		"api/v1/users/*",      // 路径不以 / 开头
		"/api/v1/users/:id/",  // 路径参数用 :id（匹配侧同样取不到正确路径）
		"GET:api/v1/users/*",  // 冒号后不以 / 开头
	}
	for _, rule := range invalid {
		if reason := InvalidReason(rule); reason == "" {
			t.Errorf("InvalidReason(%q) 应判非法，实际通过", rule)
		}
	}
}
