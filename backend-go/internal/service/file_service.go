package service

import (
	"fmt"
	"io"
	"mime/multipart"
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

// FileService 文件中心。
type FileService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewFileService(db *gorm.DB, cfg *config.Config) *FileService {
	return &FileService{db: db, cfg: cfg}
}

// Upload 上传文件到本地存储。
// 安全: 用 io.CopyN 流式限制实际写入字节数（multipart 头里的 Size 可被伪造，
// 不能仅依赖 file.Size 做大小校验）。
func (s *FileService) Upload(file *multipart.FileHeader, uploadedBy string) (*model.FileRecord, error) {
	// 限制 100MB（基于实际写入的流式计数）
	const maxSize = 100 * 1024 * 1024

	// 扩展名白名单（防上传可执行/危险类型；svg 可内嵌脚本，排除防存储型 XSS）
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedUploadExt(ext) {
		return nil, apperr.New(400, "不支持的文件类型: "+ext)
	}

	// 生成唯一文件名 + 按日期分目录
	uniqueName := uuid.New().String() + ext
	subDir := time.Now().Format("2006/01/02")

	baseDir := filepath.Join("media", "uploads")
	saveDir := filepath.Join(baseDir, subDir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, apperr.Wrap(500, "创建目录失败", err)
	}

	storagePath := filepath.ToSlash(filepath.Join(subDir, uniqueName))
	fullPath := filepath.Join(baseDir, storagePath)

	// 写文件（流式计数限制，超限即失败并清理）
	src, err := file.Open()
	if err != nil {
		return nil, apperr.Wrap(500, "打开上传文件失败", err)
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, apperr.Wrap(500, "创建目标文件失败", err)
	}

	// io.CopyN → Sync → Close 三步都必须成功，任一失败都清理已写文件
	var written int64
	copyErr := func() error {
		var err error
		written, err = io.CopyN(dst, src, maxSize+1)
		if err != nil && err != io.EOF {
			return apperr.Wrap(500, "写入文件失败", err)
		}
		if written > maxSize {
			return apperr.New(413, "文件过大，最大允许 100MB")
		}
		// 刷盘，防止系统崩溃时文件内容缺失
		if err := dst.Sync(); err != nil {
			return apperr.Wrap(500, "刷盘失败", err)
		}
		return nil
	}()
	// 关闭错误独立检查（NFS 等网络文件系统只在 Close 时上报写错误）
	if err := dst.Close(); err != nil {
		_ = os.Remove(fullPath)
		return nil, apperr.Wrap(500, "关闭文件失败", err)
	}
	if copyErr != nil {
		_ = os.Remove(fullPath)
		return nil, copyErr
	}

	// MIME 类型
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url := fmt.Sprintf("/media/uploads/%s", storagePath)
	record := &model.FileRecord{
		OriginalName:   file.Filename,
		Size:           written, // 存实际写入字节数（multipart 头声明的 Size 可伪造）
		MimeType:       contentType,
		StorageBackend: "local",
		StoragePath:    storagePath,
		URL:            url,
		UploadedBy:     uploadedBy,
	}
	if err := s.db.Create(record).Error; err != nil {
		// 清理已写文件
		_ = os.Remove(fullPath)
		return nil, apperr.ErrInternal
	}
	return record, nil
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

// List 文件记录列表。
func (s *FileService) List(offset, limit int) ([]model.FileRecord, int64, error) {
	var items []model.FileRecord
	var count int64
	q := s.db.Model(&model.FileRecord{})
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, count, err
}

// Delete 删除文件记录 + 物理文件。
func (s *FileService) Delete(id string) error {
	var record model.FileRecord
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		return apperr.ErrNotFound
	}
	// 删物理文件
	fullPath := filepath.Join("media", "uploads", record.StoragePath)
	_ = os.Remove(fullPath)
	return s.db.Delete(&record).Error
}
