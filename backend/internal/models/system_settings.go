package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// StorageConfig 存储配置
type StorageConfig struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	UserID              *uint          `json:"user_id" gorm:"index"`
	User                *User          `json:"-" gorm:"foreignKey:UserID"`
	Name                string         `json:"name" gorm:"size:100"`
	Type                string         `json:"type" gorm:"size:30"` // local/s3/webdav/nas/google_drive/onedrive/aliyun_drive/cmcc_cloud/115_drive
	Enabled             bool           `json:"enabled" gorm:"default:true"`
	IsDefault           bool           `json:"is_default" gorm:"default:false"`
	IsBackup            bool           `json:"is_backup" gorm:"default:false"`
	Priority            int            `json:"priority" gorm:"default:0"`
	Config              datatypes.JSON `json:"config" gorm:"type:json"`                   // 统一配置JSON
	ResourceTypes       datatypes.JSON `json:"resource_types" gorm:"type:json"`           // ["texts","images","videos","documents","designs","logs"]
	Status              string         `json:"status" gorm:"size:20;default:active"`      // active/inactive/error/checking
	Description         string         `json:"description" gorm:"size:500"`               // 存储描述
	HealthCheckEnabled  bool           `json:"health_check_enabled" gorm:"default:false"` // 是否启用健康检查
	HealthCheckInterval int            `json:"health_check_interval" gorm:"default:300"`  // 健康检查间隔(秒)
	LastHealthCheck     *time.Time     `json:"last_health_check"`                         // 最后健康检查时间
	FailCount           int            `json:"fail_count" gorm:"default:0"`               // 连续失败次数
	MaxFailCount        int            `json:"max_fail_count" gorm:"default:3"`           // 最大失败次数(超过标记error)
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// StorageModuleConfig 模块与目录映射配置
type StorageModuleConfig struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        *uint     `json:"user_id" gorm:"index"`
	User          *User     `json:"-" gorm:"foreignKey:UserID"`
	ModuleCode    string    `json:"module_code" gorm:"size:50;index"`
	ModuleName    string    `json:"module_name" gorm:"size:100"`
	BaseDirectory string    `json:"base_directory" gorm:"size:255"`
	Description   string    `json:"description" gorm:"size:500"`
	Enabled       bool      `json:"enabled" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// StorageRule 存储路由规则
type StorageRule struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	UserID            *uint     `json:"user_id" gorm:"index"`
	User              *User     `json:"-" gorm:"foreignKey:UserID"`
	StorageID         uint      `json:"storage_id" gorm:"index"`
	ModuleCode        string    `json:"module_code" gorm:"size:50;index"`
	ResourceType      string    `json:"resource_type" gorm:"size:100;index"`
	CategoryCode      string    `json:"category_code" gorm:"size:10;index"`
	Priority          int       `json:"priority" gorm:"default:0"`
	Enabled           bool      `json:"enabled" gorm:"default:true"`
	Name              string    `json:"name" gorm:"size:100"`             // 规则名称
	TargetType        string    `json:"target_type" gorm:"size:50"`       // employee_photo/document/audit_log/config_backup/temp
	TargetPattern     string    `json:"target_pattern" gorm:"size:255"`   // glob模式: *.pdf, *.jpg
	SizeMin           *int64    `json:"size_min"`                         // 最小文件大小(bytes)
	SizeMax           *int64    `json:"size_max"`                         // 最大文件大小(bytes)
	FallbackStorageID *uint     `json:"fallback_storage_id" gorm:"index"` // 降级存储ID
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// SysFile 文件元数据表
type SysFile struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	StorageType     string         `json:"storage_type" gorm:"size:30;index"` // local/s3/webdav
	Path            string         `json:"path" gorm:"size:500;index"`        // storage path
	OriginalName    string         `json:"original_name" gorm:"size:255"`     // original filename
	Size            int64          `json:"size"`                              // file size in bytes
	ContentType     string         `json:"content_type" gorm:"size:100"`      // MIME type
	ETag            string         `json:"etag" gorm:"size:100"`              // file hash/etag
	StorageConfigID *uint          `json:"storage_config_id" gorm:"index"`    // FK to StorageConfig
	CreatedBy       *uint          `json:"created_by" gorm:"index"`           // FK to User who uploaded
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"` // soft delete support
}

// SMTPConfig SMTP配置
type SMTPConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Host      string    `json:"host" gorm:"size:200"`         // SMTP服务器地址
	Port      string    `json:"port" gorm:"size:10"`          // SMTP端口
	Username  string    `json:"username" gorm:"size:200"`     // 用户名
	Password  string    `json:"-" gorm:"size:200"`            // 密码 (不返回前端)
	From      string    `json:"from" gorm:"size:200"`         // 发件人邮箱
	FromName  string    `json:"from_name" gorm:"size:100"`    // 发件人名称
	UseTLS    bool      `json:"use_tls" gorm:"default:false"` // 是否使用TLS
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationConfig 通知配置（统一表）
type NotificationConfig struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Channel   string         `json:"channel" gorm:"size:20;not null;index"`
	Name      string         `json:"name" gorm:"size:100"`
	Enabled   bool           `json:"enabled" gorm:"default:false"`
	Config    datatypes.JSON `json:"config" gorm:"type:json"`
	Status    string         `json:"status" gorm:"size:20;default:active"`
	Remark    string         `json:"remark" gorm:"size:500"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
