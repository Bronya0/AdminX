// Package main 是 DjangoAdminX Go 后端的入口。
//
// 完整启动流程: 配置 → 日志 → DB → Redis → AES → JWT → repository → service → handler → router → HTTP。
// 支持 SIGINT/SIGTERM 优雅关闭。
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"djangoadminx/internal/captcha"
	"djangoadminx/internal/config"
	"djangoadminx/internal/database"
	"djangoadminx/internal/handler"
	"djangoadminx/internal/jwt"
	"djangoadminx/internal/repository"
	"djangoadminx/internal/router"
	redisclient "djangoadminx/internal/redis"
	"djangoadminx/internal/scheduler"
	"djangoadminx/internal/service"
	"djangoadminx/internal/websocket"
	"djangoadminx/pkg/crypto"
	"djangoadminx/pkg/logger"
)

func main() {
	configPath := flag.String("config", "", "配置文件路径")
	enableScheduler := flag.Bool("scheduler", false, "启用内置调度器（默认不启用，建议独立进程运行）")
	flag.Parse()

	// 1. 配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 日志
	log := logger.Init(cfg.Logging.Level, cfg.Logging.Format)
	log.Info("DjangoAdminX Go 后端启动中", "mode", cfg.Server.Mode)

	// 3. 数据库
	db, err := database.Init(cfg, log)
	if err != nil {
		log.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}
	defer func() { _ = database.Close(db) }()

	// 4. Redis
	rdb, redisErr := redisclient.Init(cfg, log)
	if redisErr != nil {
		log.Warn("Redis 连接失败，降级为无缓存模式", "error", redisErr)
	} else {
		defer func() { _ = redisclient.Close(rdb) }()
	}

	// 5. AES-GCM（配置中心加密用，密钥未配置时为 nil）
	var aesGCM *crypto.AESGCM
	if cfg.Security.AESKey != "" {
		aesGCM, err = crypto.NewAESGCM(cfg.Security.AESKey)
		if err != nil {
			log.Warn("AES 密钥配置失败，配置中心加密功能不可用", "error", err)
		}
	}

	// 6. JWT Manager
	jwtMgr := jwt.New(
		cfg.JWT.Secret, cfg.JWT.Issuer,
		cfg.AccessExpireDuration(), cfg.RefreshExpireDuration(), rdb,
	)

	// 7. Repository 层
	userRepo := repository.NewUserRepo(db)
	roleRepo := repository.NewRoleRepo(db)
	menuRepo := repository.NewMenuRepo(db)
	lockRepo := repository.NewLoginLockRepo(db)
	logRepo := repository.NewLoginLogRepo(db)
	configRepo := repository.NewConfigRepo(db)
	auditRepo := repository.NewAuditRepo(db)
	jobRepo := repository.NewJobRepo(db)
	notifRepo := repository.NewNotificationRepo(db)
	clusterRepo := repository.NewClusterRepo(db)

	// 8. Service 层
	authSvc := service.NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, log,
		cfg.Server.Mode != "release", 5, 15*time.Minute)
	userSvc := service.NewUserService(db, userRepo)
	roleSvc := service.NewRoleService(db, roleRepo)
	menuSvc := service.NewMenuService(db, menuRepo)
	configSvc := service.NewConfigService(db, configRepo, rdb, aesGCM)
	auditSvc := service.NewAuditService(auditRepo)
	jobSvc := service.NewJobService(db, jobRepo, log)
	notifSvc := service.NewNotificationService(db, notifRepo, log)
	clusterSvc := service.NewClusterService(db, clusterRepo)
	fileSvc := service.NewFileService(db, cfg)
	monitorSvc := service.NewMonitorService()
	policySvc := service.NewPolicyService(db)

	// 9. Handler 层
	authH := handler.NewAuthHandler(authSvc, userSvc, menuSvc, log)
	userH := handler.NewUserHandler(userSvc, log)
	roleH := handler.NewRoleHandler(roleSvc, log)
	menuH := handler.NewMenuHandler(menuSvc, log)
	cfgH := handler.NewConfigHandler(configSvc, log)
	auditH := handler.NewAuditHandler(auditSvc, log)
	jobH := handler.NewJobHandler(jobSvc, jobRepo, log)
	notifH := handler.NewNotificationHandler(notifSvc, log)
	clsH := handler.NewClusterHandler(clusterSvc, log)
	fileH := handler.NewFileHandler(fileSvc, log)
	monH := handler.NewCommonHandler(db, monitorSvc, log)
	policyH := handler.NewPolicyHandler(policySvc, log)
	capMgr := captcha.NewManager(rdb)
	capH := handler.NewCaptchaHandler(capMgr, log)

	// 10. WebSocket Hub
	hub := websocket.NewHub(rdb, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	wsH := handler.NewWSHandler(hub, jwtMgr, log)

	// 11. 调度器（可选，--scheduler 启用）
	if *enableScheduler || cfg.Scheduler.Enabled {
		schedMgr, err := scheduler.New(jobSvc, jobRepo, rdb, log)
		if err != nil {
			log.Error("调度器初始化失败", "error", err)
		} else {
			if err := schedMgr.Start(ctx); err != nil {
				log.Error("调度器启动失败", "error", err)
			}
			defer func() { _ = schedMgr.Stop() }()
			log.Info("调度器已启用（内置模式）")
		}
	}

	// 12. 路由
	r := router.New(&router.Deps{
		Config: cfg, DB: db, Redis: rdb,
		AuthH: authH, UserH: userH, RoleH: roleH, MenuH: menuH,
		CfgH: cfgH, AuditH: auditH, JobH: jobH, NotifH: notifH,
		ClsH: clsH, FileH: fileH, MonH: monH, PolicyH: policyH,
		CapH: capH, WSH: wsH,
		JWTCfg: router.JWTConfig{
			Secret: cfg.JWT.Secret, Issuer: cfg.JWT.Issuer,
			AccessExpire: cfg.AccessExpireDuration(), RefreshExpire: cfg.RefreshExpireDuration(),
		},
	})

	// 13. HTTP 服务
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr: addr, Handler: r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       time.Duration(cfg.Server.TimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(cfg.Server.TimeoutSec) * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("HTTP 服务监听中", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP 服务启动失败", "error", err)
			os.Exit(1)
		}
	}()

	// 14. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("收到退出信号，开始优雅关闭", "signal", sig.String())

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("优雅关闭失败", "error", err)
	}
	log.Info("服务已停止")
}
