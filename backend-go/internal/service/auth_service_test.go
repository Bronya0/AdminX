package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"adminx/internal/jwt"
	"adminx/internal/model"
	"adminx/internal/repository"
)

func TestAuthService_Login_Success(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	// seed 用户
	user := seedUser(t, db, "testuser", "password123")

	result, err := authSvc.Login(context.Background(), "testuser", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}
	if result.Access == "" {
		t.Error("Access token 为空")
	}
	if result.Refresh == "" {
		t.Error("Refresh token 为空")
	}
	if result.User == nil || result.User.ID != user.ID {
		t.Error("返回的用户信息不符")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	seedUser(t, db, "testuser", "correct")

	_, err := authSvc.Login(context.Background(), "testuser", "wrongpass", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("错误密码应返回 error")
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	_, err := authSvc.Login(context.Background(), "noone", "pass", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("不存在的用户应返回 error")
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	// 创建已禁用的用户
	seedUser(t, db, "disabled", "pass123")
	db.Model(&model.User{}).Where("username = ?", "disabled").Update("is_active", false)

	_, err := authSvc.Login(context.Background(), "disabled", "pass123", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("禁用用户应返回 error")
	}
}

func TestAuthService_Login_Lock(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, nil, true, 3, 15*time.Minute)

	seedUser(t, db, "victim", "correct")

	// 连续失败 3 次触发锁定
	for i := 0; i < 3; i++ {
		authSvc.Login(context.Background(), "victim", "wrong", "127.0.0.1", "test-agent")
	}

	_, err := authSvc.Login(context.Background(), "victim", "correct", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("被锁定后正确密码也应返回 error")
	}
}

func TestAuthService_Logout(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	user := seedUser(t, db, "logoutuser", "pass123")

	err := authSvc.Logout(context.Background(), user.ID, "")
	if err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}
}

func TestAuthService_Refresh(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	user := seedUser(t, db, "refreshuser", "pass123")

	refreshToken, _ := jwtMgr.GenerateRefreshToken(user.ID, user.Username)
	result, err := authSvc.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("Refresh 失败: %v", err)
	}
	if result.Access == "" || result.Refresh == "" {
		t.Fatal("Refresh 应返回新的 access + refresh token")
	}
}

func TestAuthService_Refresh_WithAccessToken(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	seedUser(t, db, "typeuser", "pass123")

	accessToken, _ := jwtMgr.GenerateAccessToken("nonexist", "typeuser")
	_, err := authSvc.Refresh(context.Background(), accessToken)
	if err == nil {
		t.Fatal("用 access token refresh 应返回 error")
	}
}

func TestAuthService_Introspect(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	user := seedUser(t, db, "introspectuser", "pass123")
	role := seedRole(t, db, "admin", false)
	db.Model(user).Association("Roles").Append(role)

	accessToken, _ := jwtMgr.GenerateAccessToken(user.ID, user.Username)
	result, err := authSvc.Introspect(context.Background(), accessToken)
	if err != nil {
		t.Fatalf("Introspect 失败: %v", err)
	}
	if !result.Valid {
		t.Fatal("Valid token 应返回 valid=true")
	}
	if result.UserID != user.ID {
		t.Errorf("UserID = %s, want %s", result.UserID, user.ID)
	}
	if result.Username != "introspectuser" {
		t.Errorf("Username = %s", result.Username)
	}
}

func TestAuthService_Introspect_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	result, err := authSvc.Introspect(context.Background(), "invalid-token")
	if err != nil {
		t.Fatalf("Introspect 不应返回 error: %v", err)
	}
	if result.Valid {
		t.Error("无效 token 应返回 valid=false")
	}
}

func TestAuthService_ListLoginLogs(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	jwtMgr := jwt.New("test-secret", "test-issuer", 30*time.Minute, 7*24*time.Hour, nil)
	authSvc := NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, slog.Default(), true, 5, 15*time.Minute)

	seedUser(t, db, "loguser", "pass123")
	authSvc.Login(context.Background(), "loguser", "pass123", "192.168.1.1", "test-agent")

	logs, count, err := authSvc.ListLoginLogs(0, 10, "", "", nil)
	if err != nil {
		t.Fatalf("ListLoginLogs 失败: %v", err)
	}
	if count == 0 {
		t.Error("应有登录日志记录")
	}
	if len(logs) == 0 {
		t.Error("应返回日志列表")
	}
}
