package database

import (
	"io"
	"log/slog"
	"testing"

	"gorm.io/gorm"

	"adminx/internal/config"
)

// openSQLiteMemory 用生产同一条 Init 路径（sqlite 方言 + DDL 迁移）建内存库。
func openSQLiteMemory(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := Init(
		&config.Config{Database: config.DBConfig{Driver: config.DriverSQLite, DSN: ":memory:"}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("初始化 SQLite 内存库失败: %v", err)
	}
	return db
}

// TestSQLiteStatementsCoverAllModels 守卫测试：SQLite DDL 必须覆盖 Models() 里所有模型的
// 表、列以及 M2M 中间表。新增/重命名字段却忘记同步 DDL 时，SQLite 模式会在运行时报
// "no such column"，这里提前拦住。
func TestSQLiteStatementsCoverAllModels(t *testing.T) {
	db := openSQLiteMemory(t)
	tables := sqliteTables(t, db)

	for _, m := range Models() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			t.Fatalf("解析模型失败: %v", err)
		}
		schema := stmt.Schema

		if !tables[schema.Table] {
			t.Errorf("SQLite DDL 缺少表 %s（模型 %s）", schema.Table, schema.Name)
			continue
		}
		cols := sqliteColumns(t, db, schema.Table)
		for _, f := range schema.Fields {
			if f.DBName == "" || f.IgnoreMigration {
				continue
			}
			if !cols[f.DBName] {
				t.Errorf("表 %s 缺少列 %s", schema.Table, f.DBName)
			}
		}

		for _, rel := range schema.Relationships.Relations {
			join := rel.JoinTable
			if join == nil {
				continue
			}
			if !tables[join.Name] {
				t.Errorf("SQLite DDL 缺少中间表 %s", join.Name)
				continue
			}
			joinCols := sqliteColumns(t, db, join.Name)
			for _, f := range join.Fields {
				if f.DBName == "" || f.IgnoreMigration {
					continue
				}
				if !joinCols[f.DBName] {
					t.Errorf("中间表 %s 缺少列 %s", join.Name, f.DBName)
				}
			}
		}
	}
}

func sqliteTables(t *testing.T, db *gorm.DB) map[string]bool {
	t.Helper()
	var names []string
	if err := db.Raw(`SELECT name FROM sqlite_master WHERE type = 'table'`).Scan(&names).Error; err != nil {
		t.Fatalf("查询 SQLite 表名失败: %v", err)
	}
	return toSet(names)
}

func sqliteColumns(t *testing.T, db *gorm.DB, table string) map[string]bool {
	t.Helper()
	var names []string
	if err := db.Raw(`SELECT name FROM pragma_table_info(?)`, table).Scan(&names).Error; err != nil {
		t.Fatalf("查询表 %s 的列失败: %v", table, err)
	}
	return toSet(names)
}

func toSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}
