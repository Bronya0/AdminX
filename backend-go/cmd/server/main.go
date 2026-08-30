// Package main 是 Go 后端的入口。
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
	"sync/atomic"
	"syscall"
	"time"

	"adminx/internal/captcha"
	"adminx/internal/config"
	"adminx/internal/database"
	"adminx/internal/handler"
	"adminx/internal/jwt"
	"adminx/internal/repository"
	"adminx/internal/router"
	redisclient "adminx/internal/redis"
	"adminx/internal/scheduler"
	"adminx/internal/service"
	"adminx/internal/websocket"
	"adminx/pkg/crypto"
	"adminx/pkg/logger"
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
	log.Info("Go 后端启动中", "mode", cfg.Server.Mode)

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
	// 登录锁定常开（不随 mode 变化）：防爆破是认证基线，debug 模式更需要。
	authSvc := service.NewAuthService(db, userRepo, lockRepo, logRepo, jwtMgr, log,
		true, 5, 15*time.Minute)
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

	// 任务 CRUD → 调度器重载：标记 reload_pending（跨进程）+ 本进程内立即重载。
	// 调度器可能在本进程之后才启动，用原子指针解耦时序。
	var schedMgrPtr atomic.Pointer[scheduler.Manager]
	jobSvc.SetOnJobsChanged(func() {
		if err := jobRepo.SetReloadPending(); err != nil {
			log.Warn("标记 reload_pending 失败", "error", err)
		}
		if m := schedMgrPtr.Load(); m != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := m.Reload(ctx); err != nil {
				log.Error("调度器重载失败", "error", err)
			}
		}
	})

	// 9. Handler 层
	capMgr := captcha.NewManager(rdb)
	authH := handler.NewAuthHandler(authSvc, userSvc, menuSvc, capMgr, cfg.Security.LoginCaptchaRequired, log)
	userH := handler.NewUserHandler(userSvc, auditSvc, log)
	roleH := handler.NewRoleHandler(roleSvc, auditSvc, log)
	menuH := handler.NewMenuHandler(menuSvc, auditSvc, log)
	cfgH := handler.NewConfigHandler(configSvc, log)
	auditH := handler.NewAuditHandler(auditSvc, log)
	jobH := handler.NewJobHandler(jobSvc, jobRepo, log)
	notifH := handler.NewNotificationHandler(notifSvc, log)
	clsH := handler.NewClusterHandler(clusterSvc, log)
	fileH := handler.NewFileHandler(fileSvc, log)
	monH := handler.NewCommonHandler(db, monitorSvc, log)
	policyH := handler.NewPolicyHandler(policySvc, log)
	capH := handler.NewCaptchaHandler(capMgr, log)

	// 10. WebSocket Hub
	hub := websocket.NewHub(rdb, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	wsH := handler.NewWSHandler(hub, jwtMgr, cfg.Security.WSAllowedOrigins, log)

	// 11. 调度器（可选，--scheduler 启用）
	if *enableScheduler || cfg.Scheduler.Enabled {
		schedMgr, err := scheduler.New(jobSvc, jobRepo, rdb, log)
		if err != nil {
			log.Error("调度器初始化失败", "error", err)
		} else {
			if err := schedMgr.Start(ctx); err != nil {
				log.Error("调度器启动失败", "error", err)
			}
			schedMgrPtr.Store(schedMgr)
			defer func() { _ = schedMgr.Stop() }()
			log.Info("调度器已启用（内置模式）")
		}
	}

	// 组件注册端点无共享密钥时的告警（生产必须配置 security.component_secret）
	if cfg.Security.ComponentSecret == "" {
		log.Warn("security.component_secret 未配置：/cluster/components/* 公开端点处于无鉴权模式，生产环境请配置共享密钥")
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
			// 手动 cancel 让调度器/WebSocket 子协程退出（os.Exit 不执行 defer）
			cancel()
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
