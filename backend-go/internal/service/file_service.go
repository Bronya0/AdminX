package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperr "djangoadminx/pkg/errors"

	"djangoadminx/internal/config"
	"djangoadminx/internal/model"
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
func (s *FileService) Upload(file *multipart.FileHeader, uploadedBy string) (*model.FileRecord, error) {
	// 限制 100MB
	if file.Size > 100*1024*1024 {
		return nil, apperr.New(413, "文件过大，最大允许 100MB")
	}

	// 生成唯一文件名 + 按日期分目录
	ext := filepath.Ext(file.Filename)
	uniqueName := uuid.New().String() + ext
	subDir := time.Now().Format("2006/01/02")

	baseDir := filepath.Join("media", "uploads")
	saveDir := filepath.Join(baseDir, subDir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, apperr.Wrap(500, "创建目录失败", err)
	}

	storagePath := filepath.ToSlash(filepath.Join(subDir, uniqueName))
	fullPath := filepath.Join(baseDir, storagePath)

	// 写文件
	src, err := file.Open()
	if err != nil {
		return nil, apperr.Wrap(500, "打开上传文件失败", err)
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, apperr.Wrap(500, "创建目标文件失败", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, apperr.Wrap(500, "写入文件失败", err)
	}

	// MIME 类型
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url := fmt.Sprintf("/media/uploads/%s", storagePath)
	record := &model.FileRecord{
		OriginalName:   file.Filename,
		Size:           file.Size,
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
