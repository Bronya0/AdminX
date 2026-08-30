package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"adminx/internal/jwt"
	"adminx/internal/model"
)

// ── matchPath 纯逻辑测试 ──

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		requestPath string
		rule        string
		want        bool
	}{
		// 精确匹配
		{"精确匹配", "GET", "/api/health", "GET:/api/health", true},
		{"方法不匹配", "POST", "/api/health", "GET:/api/health", false},
		// 通配符
		{"通配符/任意方法", "GET", "/api/users/1", "/api/users/*", true},
		{"通配符/POST", "POST", "/api/users/1", "/api/users/*", true},
		{"通配符/多级", "GET", "/api/users/1/profile", "GET:/api/users/1/*", true},
		{"通配符/不匹配", "GET", "/api/roles/1", "GET:/api/users/*", false},
		// 无方法前缀
		{"无方法前缀匹配", "GET", "/api/health", "/api/health", true},
		// 单字符通配 ?
		{"单字符通配", "GET", "/api/user/1", "GET:/api/user/?", true},
		{"单字符通配不匹配", "GET", "/api/user/12", "GET:/api/user/?", false},
		// 边界情况
		{"空规则", "GET", "/api/test", "", false},
		{"空规则空格", "GET", "/api/test", "   ", false},
		// 非法 pattern 降级为精确匹配
		{"非法pattern匹配", "GET", "/api/[test", "GET:/api/[test", true},
		{"非法pattern不匹配", "GET", "/api/other", "GET:/api/[test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPath(tt.method, tt.requestPath, tt.rule)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q, %q) = %v, want %v",
					tt.method, tt.requestPath, tt.rule, got, tt.want)
			}
		})
	}
}

// ── ParseRate 纯逻辑测试 ──

func TestParseRate(t *testing.T) {
	tests := []struct {
		rate     string
		wantReq  int
		wantWin  time.Duration
	}{
		{"30/minute", 30, time.Minute},
		{"100/hour", 100, time.Hour},
		{"5/second", 5, time.Second},
		{"10/s", 10, time.Second},
		{"200/day", 200, 24 * time.Hour},
		{"invalid", 100, time.Minute},
		{"-1/minute", 100, time.Minute},
		{"abc/minute", 100, time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.rate, func(t *testing.T) {
			limit := ParseRate(tt.rate)
			if limit.Requests != tt.wantReq {
				t.Errorf("Requests = %d, want %d", limit.Requests, tt.wantReq)
			}
			if limit.Window != tt.wantWin {
				t.Errorf("Window = %v, want %v", limit.Window, tt.wantWin)
			}
		})
	}
}

// ── Middleware 集成测试（httptest + SQLite）──

func setupMiddlewareTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接 SQLite 失败: %v", err)
	}
	// SQLite :memory: 每条连接独立，必须限制连接池为 1
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	// 创建测试表
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY, username TEXT, password TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1, is_superuser INTEGER DEFAULT 0,
			last_login DATETIME, last_activity DATETIME, last_logout DATETIME,
			date_joined DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY, name TEXT, is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS menus (
			id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT, name TEXT,
			path TEXT, is_active INTEGER DEFAULT 1, allowed_paths TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS user_roles (user_id TEXT, role_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS role_menus (role_id TEXT, menu_id INTEGER)`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("建表失败: %v", err)
		}
	}
	return db
}

func setupGinContext(method, path, token string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	if token != "" {
		c.Request.Header.Set("Authorization", "Bearer "+token)
	}
	return c, w
}

func TestJWTAuth_NoToken(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	c, w := setupGinContext("GET", "/api/test", "")
	JWTAuth(jwtMgr, db)(c)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !c.IsAborted() {
		t.Error("无 token 应 abort")
	}
}

func TestJWTAuth_BadFormat(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	c, w := setupGinContext("GET", "/api/test", "")
	c.Request.Header.Set("Authorization", "Basic abc123")
	JWTAuth(jwtMgr, db)(c)

	if !c.IsAborted() {
		t.Error("错误格式应 abort")
	}
	_ = w
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	c, w := setupGinContext("GET", "/api/test", "invalid-token-here")
	JWTAuth(jwtMgr, db)(c)

	if !c.IsAborted() {
		t.Error("无效 token 应 abort")
	}
	_ = w
}

func TestJWTAuth_UserNotFound(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	// 签发 token 对应用户但用户不存在于 DB
	token, _ := jwtMgr.GenerateAccessToken("nonexistent", "ghost")

	c, w := setupGinContext("GET", "/api/test", token)
	JWTAuth(jwtMgr, db)(c)

	if !c.IsAborted() {
		t.Error("用户不存在应 abort")
	}
	_ = w
}

func TestJWTAuth_Success(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	// 创建用户
	db.Exec(`INSERT INTO users (id, username, is_active, is_superuser) VALUES ('u1', 'testuser', 1, 1)`)

	token, _ := jwtMgr.GenerateAccessToken("u1", "testuser")

	c, w := setupGinContext("GET", "/api/test", token)
	JWTAuth(jwtMgr, db)(c)

	if c.IsAborted() {
		t.Error("有效 token 不应 abort")
	}
	uid, _ := c.Get("user_id")
	if uid != "u1" {
		t.Errorf("user_id = %v, want u1", uid)
	}
	_ = w
}

func TestJWTAuth_InactiveUser(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	db.Exec(`INSERT INTO users (id, username, is_active) VALUES ('u2', 'disabled', 0)`)

	token, _ := jwtMgr.GenerateAccessToken("u2", "disabled")

	c, _ := setupGinContext("GET", "/api/test", token)
	JWTAuth(jwtMgr, db)(c)

	if !c.IsAborted() {
		t.Error("禁用用户应 abort")
	}
}

func TestJWTAuth_RefreshTokenRejected(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	db.Exec(`INSERT INTO users (id, username, is_active) VALUES ('u3', 'refresh', 1)`)

	refreshToken, _ := jwtMgr.GenerateRefreshToken("u3", "refresh")

	c, _ := setupGinContext("GET", "/api/test", refreshToken)
	JWTAuth(jwtMgr, db)(c)

	if !c.IsAborted() {
		t.Error("refresh token 用于 API 应 abort")
	}
}

func TestRequireSuperuser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("超级管理员放行", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("is_superuser", true)
		c.Request = httptest.NewRequest("GET", "/", nil)

		RequireSuperuser()(c)
		if c.IsAborted() {
			t.Error("超级管理员不应 abort")
		}
	})

	t.Run("普通用户拒绝", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("is_superuser", false)
		c.Request = httptest.NewRequest("GET", "/", nil)

		RequireSuperuser()(c)
		if !c.IsAborted() {
			t.Error("普通用户应 abort")
		}
	})

	t.Run("未认证拒绝", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)

		RequireSuperuser()(c)
		if !c.IsAborted() {
			t.Error("未认证应 abort")
		}
	})
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	RequestID()(c)

	rid, exists := c.Get("request_id")
	if !exists {
		t.Fatal("request_id 未设置")
	}
	if rid.(string) == "" {
		t.Error("request_id 为空")
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header 未设置")
	}
}

func TestRequestID_PreserveExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Request-ID", "my-custom-id")

	RequestID()(c)

	rid, _ := c.Get("request_id")
	if rid.(string) != "my-custom-id" {
		t.Errorf("request_id = %s, want my-custom-id", rid)
	}
}

func TestRBAC_SuperuserBypass(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("is_superuser", true)
	c.Request = httptest.NewRequest("GET", "/api/admin", nil)

	RBAC(db, "")(c)
	if c.IsAborted() {
		t.Error("超级管理员应绕过 RBAC")
	}
}

func TestRBAC_Unauthenticated(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	RBAC(db, "")(c)
	if !c.IsAborted() {
		t.Error("未认证应 abort")
	}
}

func TestRBAC_NoMenuPermission(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)

	// 创建用户但无菜单关联
	db.Exec(`INSERT INTO users (id, username) VALUES ('u1', 'nomenu')`)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "u1")
	c.Request = httptest.NewRequest("GET", "/api/secret", nil)

	RBAC(db, "")(c)
	if !c.IsAborted() {
		t.Error("无菜单关联的用户应被拒绝")
	}
}

func TestRBAC_WithMenuPermission(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)

	// 创建用户/角色/菜单 + 关联
	db.Exec(`INSERT INTO users (id, username) VALUES ('u2', 'hasmenu')`)
	db.Exec(`INSERT INTO roles (id, name) VALUES ('r1', 'viewer')`)
	db.Exec(`INSERT INTO menus (id, code, name, path, is_active, allowed_paths) VALUES (1, 'users', '用户管理', '/api/users', 1, '["GET:/api/users/*"]')`)
	db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ('u2', 'r1')`)
	db.Exec(`INSERT INTO role_menus (role_id, menu_id) VALUES ('r1', 1)`)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "u2")
	c.Request = httptest.NewRequest("GET", "/api/users/list", nil)

	RBAC(db, "")(c)
	if c.IsAborted() {
		t.Error("有菜单权限的用户不应被 abort")
	}
}

// 确保 model import 被使用
var _ = &model.Menu{}
