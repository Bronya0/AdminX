package repository

import (
	"io"
	"log/slog"
	"testing"

	"gorm.io/gorm"

	"adminx/internal/config"
	"adminx/internal/database"
)

// setupTestDB 建 SQLite 内存库（走生产同一套 Init + SQLite DDL）。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Init(
		&config.Config{Database: config.DBConfig{Driver: config.DriverSQLite, DSN: ":memory:"}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("初始化测试库失败: %v", err)
	}
	return db
}

// TestSchedulerHeartbeat 心跳写入/存活判定（时间比较不依赖数据库方言函数）。
func TestSchedulerHeartbeat(t *testing.T) {
	repo := NewJobRepo(setupTestDB(t))

	alive, err := repo.IsSchedulerAlive()
	if err != nil {
		t.Fatalf("IsSchedulerAlive 失败: %v", err)
	}
	if alive {
		t.Error("没有心跳记录时不应判定调度器存活")
	}

	if err := repo.WriteHeartbeat(); err != nil {
		t.Fatalf("WriteHeartbeat 失败: %v", err)
	}
	if err := repo.WriteHeartbeat(); err != nil {
		t.Fatalf("重复 WriteHeartbeat 应幂等（upsert）: %v", err)
	}
	if alive, err = repo.IsSchedulerAlive(); err != nil || !alive {
		t.Fatalf("写完心跳后应判定存活: alive=%v err=%v", alive, err)
	}

	// 置位 reload_pending 不能覆盖 last_heartbeat
	if err := repo.SetReloadPending(); err != nil {
		t.Fatalf("SetReloadPending 失败: %v", err)
	}
	if alive, err = repo.IsSchedulerAlive(); err != nil || !alive {
		t.Fatalf("置位 reload_pending 后心跳应保留: alive=%v err=%v", alive, err)
	}
}

// TestSetReloadPendingDoesNotFakeHeartbeat 置位重载标记不得伪造调度器心跳：
// 否则 web 进程每调一次 /jobs/reload/，组件状态页就会把调度器误报为在线 30s。
func TestSetReloadPendingDoesNotFakeHeartbeat(t *testing.T) {
	repo := NewJobRepo(setupTestDB(t))

	if err := repo.SetReloadPending(); err != nil {
		t.Fatalf("SetReloadPending 失败: %v", err)
	}
	alive, err := repo.IsSchedulerAlive()
	if err != nil {
		t.Fatalf("IsSchedulerAlive 失败: %v", err)
	}
	if alive {
		t.Error("只置位 reload_pending 不应被判定为调度器存活")
	}

	// 标记仍可被调度器进程消费
	got, err := repo.ConsumeReloadPending()
	if err != nil || !got {
		t.Fatalf("reload_pending 应可消费: got=%v err=%v", got, err)
	}
}

// TestConsumeReloadPending 重载标记只能被消费一次。
func TestConsumeReloadPending(t *testing.T) {
	repo := NewJobRepo(setupTestDB(t))

	got, err := repo.ConsumeReloadPending()
	if err != nil {
		t.Fatalf("ConsumeReloadPending 失败: %v", err)
	}
	if got {
		t.Error("未置位时不应返回 true")
	}

	if err := repo.SetReloadPending(); err != nil {
		t.Fatalf("SetReloadPending 失败: %v", err)
	}
	if got, err = repo.ConsumeReloadPending(); err != nil || !got {
		t.Fatalf("首次消费应返回 true: got=%v err=%v", got, err)
	}
	if got, err = repo.ConsumeReloadPending(); err != nil || got {
		t.Fatalf("二次消费应返回 false: got=%v err=%v", got, err)
	}

	// 再次置位后仍可消费
	if err := repo.SetReloadPending(); err != nil {
		t.Fatalf("再次 SetReloadPending 失败: %v", err)
	}
	if got, err = repo.ConsumeReloadPending(); err != nil || !got {
		t.Fatalf("再次置位后消费应返回 true: got=%v err=%v", got, err)
	}
}
