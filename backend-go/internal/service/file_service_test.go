package service

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"adminx/internal/model"
)

// writeLimited 是上传大小限制的唯一实现：超限必须中断且报 errTooLarge。
func TestWriteLimited(t *testing.T) {
	var buf bytes.Buffer
	// 恰好等于上限
	n, err := writeLimited(&buf, []byte("abc"), strings.NewReader("de"), 5)
	if err != nil || n != 5 {
		t.Fatalf("恰好等于上限应通过: n=%d err=%v", n, err)
	}

	// 超过上限 1 字节
	buf.Reset()
	if _, err := writeLimited(&buf, []byte("abc"), strings.NewReader("def"), 5); !errors.Is(err, errTooLarge) {
		t.Errorf("超限 1 字节应返回 errTooLarge, 实际 %v", err)
	}

	// 头部本身就超限
	buf.Reset()
	if _, err := writeLimited(&buf, make([]byte, 6), strings.NewReader(""), 5); !errors.Is(err, errTooLarge) {
		t.Errorf("头部超限应返回 errTooLarge, 实际 %v", err)
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"report.pdf":                  "report.pdf",
		"../../etc/passwd":            "passwd",
		`..\..\windows\evil.pdf`:      "evil.pdf",
		"a\x00b\x01c.pdf":             "abc.pdf",
		"":                            "unnamed",
		".":                           "unnamed",
		strings.Repeat("x", 400) + ".pdf": strings.Repeat("x", 255),
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAllowedUploadExt(t *testing.T) {
	for _, ext := range []string{".jpg", ".pdf", ".zip", ".md"} {
		if !allowedUploadExt(ext) {
			t.Errorf("%s 应在白名单内", ext)
		}
	}
	// 可执行/可嵌脚本类型必须拒绝
	for _, ext := range []string{".sh", ".php", ".jsp", ".html", ".svg", ".exe", ""} {
		if allowedUploadExt(ext) {
			t.Errorf("%s 不应被允许", ext)
		}
	}
}

// 归属隔离：非超管只能看到/删除自己的文件（此前任何有 files 权限的人都能删他人文件）。
func TestFileService_OwnershipIsolation(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFileService(db, nil, slog.Default())
	owner := seedUser(t, db, "file-owner", "pass123")
	other := seedUser(t, db, "file-other", "pass456")

	record := &model.FileRecord{
		OriginalName:   "doc.pdf",
		Size:           10,
		StoragePath:    "2026/01/02/none.pdf",
		UploadedBy:     owner.ID,
	}

	if err := db.Create(record).Error; err != nil {
		t.Fatalf("建文件记录失败: %v", err)
	}

	// 他人可见性：非超管（all=false）看不到，超管（all=true）看得到
	if _, err := svc.FindForUser(record.ID, other.ID, false); err == nil {
		t.Error("非超管应看不到他人文件")
	}
	if _, err := svc.FindForUser(record.ID, owner.ID, false); err != nil {
		t.Errorf("本人应能看到自己的文件: %v", err)
	}
	if _, err := svc.FindForUser(record.ID, other.ID, true); err != nil {
		t.Errorf("超管应能看到所有文件: %v", err)
	}

	// 列表隔离
	items, count, err := svc.List(0, 10, other.ID, false)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if count != 0 || len(items) != 0 {
		t.Errorf("非超管列表应看不到他人文件: count=%d", count)
	}
	_, count, err = svc.List(0, 10, other.ID, true)
	if err != nil || count != 1 {
		t.Errorf("超管列表应看到全部文件: count=%d err=%v", count, err)
	}

	// 删除隔离
	if err := svc.Delete(record.ID, other.ID, false); err == nil {
		t.Error("非超管不应能删除他人文件")
	}
	if err := svc.Delete(record.ID, owner.ID, false); err != nil {
		t.Errorf("本人应能删除自己的文件: %v", err)
	}
	var left int64
	if err := db.Model(&model.FileRecord{}).Where("id = ?", record.ID).Count(&left).Error; err != nil {
		t.Fatalf("查询残留记录失败: %v", err)
	}
	if left != 0 {
		t.Error("删除后记录应已移除")
	}
}

// 列表在超管视角下不做过滤。
func TestFileService_ListScope(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFileService(db, nil, slog.Default())
	u := seedUser(t, db, "file-scope", "pass123")

	_ = db.Create(&model.FileRecord{OriginalName: "a.txt", StoragePath: "a", UploadedBy: u.ID}).Error
	_ = db.Create(&model.FileRecord{OriginalName: "b.txt", StoragePath: "b", UploadedBy: "someone-else"}).Error

	items, count, err := svc.List(0, 10, u.ID, false)
	if err != nil || count != 1 || len(items) != 1 {
		t.Errorf("按上传人过滤失败: count=%d err=%v", count, err)
	}
	if _, count, _ := svc.List(0, 10, u.ID, true); count != 2 {
		t.Errorf("超管视角应返回全部: count=%d", count)
	}
}
