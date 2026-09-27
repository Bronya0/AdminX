package service

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperr "adminx/pkg/errors"

	"adminx/internal/config"
	"adminx/internal/model"
)

// maxUploadSize 单文件大小上限（按实际写入字节流式计数，不信任请求里声明的长度）。
const maxUploadSize = 100 * 1024 * 1024

// errTooLarge 超过大小上限的哨兵错误（用于区分 413 与 500）。
var errTooLarge = errors.New("上传文件超过大小上限")

// uploadRootPath 本地存储根目录（相对进程工作目录）。
var uploadRootPath = filepath.Join("media", "uploads")

// FileService 文件中心。
type FileService struct {
	db     *gorm.DB
	cfg    *config.Config
	logger *slog.Logger
}

func NewFileService(db *gorm.DB, cfg *config.Config, logger *slog.Logger) *FileService {
	return &FileService{db: db, cfg: cfg, logger: logger}
}

// Upload 上传文件到本地存储（流式）。
//
// 不用 handler 的 c.FormFile：它会把整个请求体先落到系统临时目录（gin 默认 32MB
// 内存、超出部分写盘），于是"100MB 上限"挡不住先把 /tmp 写满。这里直接消费
// multipart part，超限立刻中断，不在临时盘留副本。
func (s *FileService) Upload(filename string, src io.Reader, uploadedBy string) (*model.FileRecord, error) {
	originalName := sanitizeFilename(filename)
	// 扩展名白名单（不含 svg：可内嵌脚本，防存储型 XSS）
	ext := strings.ToLower(filepath.Ext(originalName))
	if !allowedUploadExt(ext) {
		return nil, apperr.New(400, "不支持的文件类型: "+ext)
	}

	uniqueName := uuid.NewString() + ext
	subDir := time.Now().Format("2006/01/02")
	saveDir := filepath.Join(uploadRootPath, subDir)
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return nil, apperr.Wrap(500, "创建目录失败", err)
	}

	storagePath := filepath.ToSlash(filepath.Join(subDir, uniqueName))
	fullPath := filepath.Join(saveDir, uniqueName)

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, apperr.Wrap(500, "创建目标文件失败", err)
	}

	// 前 512 字节用于嗅探真实类型：客户端的 Content-Type 完全可伪造
	head := make([]byte, 512)
	n, headErr := io.ReadFull(src, head)
	if headErr != nil && headErr != io.EOF && headErr != io.ErrUnexpectedEOF {
		_ = dst.Close()
		_ = os.Remove(fullPath)
		return nil, apperr.Wrap(500, "读取上传文件失败", headErr)
	}
	head = head[:n]
	mimeType := "application/octet-stream"
	if len(head) > 0 {
		mimeType = http.DetectContentType(head)
	}

	written, copyErr := writeLimited(dst, head, src, maxUploadSize)
	syncErr := dst.Sync()
	closeErr := dst.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(fullPath)
		switch {
		case errors.Is(copyErr, errTooLarge):
			return nil, apperr.New(413, fmt.Sprintf("文件过大，最大允许 %dMB", maxUploadSize/1024/1024))
		case copyErr != nil:
			return nil, apperr.Wrap(500, "写入文件失败", copyErr)
		case syncErr != nil:
			return nil, apperr.Wrap(500, "刷盘失败", syncErr)
		default:
			return nil, apperr.Wrap(500, "关闭文件失败", closeErr)
		}
	}

	id := uuid.NewString()
	record := &model.FileRecord{
		ID:             id,
		OriginalName:   originalName,
		Size:           written,
		MimeType:       mimeType,
		StorageBackend: "local",
		StoragePath:    storagePath,
		// 下载走鉴权接口：既不再是"后端没有路由"的死链，
		// 也避免把上传目录直接挂成匿名可读的静态目录。
		URL:        "/api/v1/files/records/" + id + "/download/",
		UploadedBy: uploadedBy,
	}
	if err := s.db.Create(record).Error; err != nil {
		_ = os.Remove(fullPath)
		return nil, apperr.ErrInternal
	}
	return record, nil
}

// writeLimited 先写 head，再从 rest 流式拷贝，总写入量超过 max 时返回 errTooLarge。
func writeLimited(dst io.Writer, head []byte, rest io.Reader, max int64) (int64, error) {
	if int64(len(head)) > max {
		return int64(len(head)), errTooLarge
	}
	if _, err := dst.Write(head); err != nil {
		return int64(len(head)), err
	}
	written, err := io.CopyN(dst, rest, max-int64(len(head))+1)
	total := int64(len(head)) + written
	if total > max {
		return total, errTooLarge
	}
	if err != nil && err != io.EOF {
		return total, err
	}
	return total, nil
}

// allowedUploadExt 允许上传的扩展名白名单（不含 svg：可内嵌脚本，防存储型 XSS）。
func allowedUploadExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".txt", ".md", ".csv", ".zip", ".rar", ".7z", ".tar", ".gz":
		return true
	}
	return false
}

// sanitizeFilename 清理客户端文件名：去掉目录部分与控制字符并限长。
// 它会被回显到列表和下载响应头里，控制字符/引号会破坏响应头。
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == "/" {
		return "unnamed"
	}
	return truncateStr(name, 255)
}

// List 文件记录列表。all=false（非超管）时只返回自己上传的记录，避免横向列举他人文件。
func (s *FileService) List(offset, limit int, uploadedBy string, all bool) ([]model.FileRecord, int64, error) {
	var items []model.FileRecord
	var count int64
	q := s.db.Model(&model.FileRecord{})
	if !all {
		q = q.Where("uploaded_by = ?", uploadedBy)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, count, err
}

// FindForUser 取一条文件记录并校验归属。
// all=false 时他人文件按"不存在"处理（不区分 403/404，避免暴露文件是否存在）。
func (s *FileService) FindForUser(id, uploadedBy string, all bool) (*model.FileRecord, error) {
	var record model.FileRecord
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		return nil, apperr.ErrNotFound
	}
	if !all && record.UploadedBy != uploadedBy {
		return nil, apperr.ErrNotFound
	}
	return &record, nil
}

// DownloadPath 返回下载用的物理路径与原始文件名（已校验归属）。
func (s *FileService) DownloadPath(id, uploadedBy string, all bool) (string, string, error) {
	record, err := s.FindForUser(id, uploadedBy, all)
	if err != nil {
		return "", "", err
	}
	fullPath := filepath.Join(uploadRootPath, filepath.FromSlash(record.StoragePath))
	if _, err := os.Stat(fullPath); err != nil {
		return "", "", apperr.ErrNotFound
	}
	return fullPath, record.OriginalName, nil
}

// Delete 删除文件记录 + 物理文件（已校验归属）。
// 先删记录再删盘：反过来的话，"删盘成功、删行失败"会留下谁也清不掉的孤儿文件。
func (s *FileService) Delete(id, uploadedBy string, all bool) error {
	record, err := s.FindForUser(id, uploadedBy, all)
	if err != nil {
		return err
	}
	if err := s.db.Delete(&model.FileRecord{}, "id = ?", record.ID).Error; err != nil {
		return apperr.ErrInternal
	}
	fullPath := filepath.Join(uploadRootPath, filepath.FromSlash(record.StoragePath))
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("删除物理文件失败（记录已删除，需人工清理）", "path", fullPath, "error", err)
	}
	return nil
}
