package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileRecord 文件上传记录。
type FileRecord struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OriginalName  string    `gorm:"size:512;not null" json:"original_name"`
	Size          int64     `gorm:"not null" json:"size"` // 字节数
	MimeType      string    `gorm:"size:128;not null;default:''" json:"mime_type"`
	StorageBackend string   `gorm:"size:32;not null;default:'local'" json:"storage_backend"`
	StoragePath   string    `gorm:"size:1024;not null" json:"storage_path"`
	URL           string    `gorm:"size:1024;not null;default:''" json:"url"`
	UploadedBy    string    `gorm:"size:128;not null;default:''" json:"uploaded_by"`
	CreatedAt     time.Time `gorm:"index;autoCreateTime" json:"created_at"`
}

func (FileRecord) TableName() string { return "file_records" }

func (f *FileRecord) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	return nil
}
