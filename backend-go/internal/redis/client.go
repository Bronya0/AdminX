// Package redis 封装 go-redis 客户端初始化。
//
// Redis 用于: 缓存（配置中心）、限流（令牌桶）、JWT 黑名单、
// 调度器分布式锁、WebSocket 跨实例广播（pub/sub）。
package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"djangoadminx/internal/config"
)

// Init 初始化 Redis 客户端并验证连通性。
func Init(cfg *config.Config, logger *slog.Logger) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("解析 REDIS_URL 失败: %w", err)
	}

	client := redis.NewClient(opt)

	// 连通性验证（5s 超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}

	logger.Info("Redis 连接成功", "url", cfg.Redis.URL)
	return client, nil
}

// Close 关闭 Redis 连接。
func Close(client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
