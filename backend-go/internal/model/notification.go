package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification 通知消息。
type Notification struct {
	ID               string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID           *string   `gorm:"type:uuid;index" json:"user_id"` // null = 广播给所有用户
	Title            string    `gorm:"size:255;not null" json:"title"`
	Content          string    `gorm:"type:text;not null;default:''" json:"content"`
	NotificationType string    `gorm:"size:20;not null;default:'info'" json:"notification_type"` // info/success/warning/error
	// IsRead 是"当前用户是否已读"的计算字段，真值按用户存在 notification_reads。
	// 不能落成通知行上的单列：广播通知（user_id IS NULL）只有一份 is_read，
	// 任意一个用户标记已读就会把所有人的未读数一起清零。
	IsRead    bool      `gorm:"-" json:"is_read"`
	CreatedAt time.Time `gorm:"index;autoCreateTime" json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}

// NotificationRead 用户对某条通知的已读记录（已读状态的真值来源，按用户隔离）。
type NotificationRead struct {
	NotificationID string    `gorm:"type:uuid;primaryKey" json:"notification_id"`
	UserID         string    `gorm:"type:uuid;primaryKey" json:"user_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (NotificationRead) TableName() string { return "notification_reads" }

// WebhookConfig Webhook 外发配置。
type WebhookConfig struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	URL       string    `gorm:"size:512;not null" json:"url"`
	Secret    string    `gorm:"size:256;not null;default:''" json:"secret"` // HMAC 签名密钥
	Events    string    `gorm:"size:256;not null;default:'info,success,warning,error'" json:"events"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (WebhookConfig) TableName() string { return "webhook_configs" }

func (w *WebhookConfig) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}

// WebhookLog Webhook 外发记录。
type WebhookLog struct {
	ID             string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	WebhookID      string     `gorm:"type:uuid;not null;index" json:"webhook_id"`
	NotificationID *string    `gorm:"type:uuid;index" json:"notification_id"`
	Status         string     `gorm:"size:20;not null;default:'success'" json:"status"` // success/failed
	ResponseStatus *int       `json:"response_status"`
	ResponseBody   string     `gorm:"type:text;not null;default:''" json:"response_body"`
	ErrorMessage   string     `gorm:"type:text;not null;default:''" json:"error_message"`
	CreatedAt      time.Time  `gorm:"index;autoCreateTime" json:"created_at"`
}

func (WebhookLog) TableName() string { return "webhook_logs" }

func (l *WebhookLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	return nil
}
