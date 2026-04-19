package models

import (
	"time"

	"gorm.io/datatypes"
)

// DocumentCategory 一级分类
type DocumentCategory struct {
	ID            uint                  `json:"id" gorm:"primaryKey"`
	UserID        *uint                 `json:"user_id" gorm:"index"`
	User          *User                 `json:"-" gorm:"foreignKey:UserID"`
	Code          string                `json:"code" gorm:"size:10;uniqueIndex"` // WS/KJ/DZ/ZM
	Name          string                `json:"name" gorm:"size:100"`            // 文书/科技/电子/专门
	Description   string                `json:"description" gorm:"size:500"`
	SubCategories []DocumentSubCategory `json:"sub_categories" gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE"`
	SortOrder     int                   `json:"sort_order" gorm:"default:0"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// DocumentSubCategory 二级分类
type DocumentSubCategory struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	CategoryID   uint              `json:"category_id" gorm:"index"`
	Category     *DocumentCategory `json:"-" gorm:"foreignKey:CategoryID"`
	Code         string            `json:"code" gorm:"size:10"` // 01/02/03/04/05
	Name         string            `json:"name" gorm:"size:100"`
	Description  string            `json:"description" gorm:"size:500"`
	FieldsConfig datatypes.JSON    `json:"fields_config" gorm:"type:json"` // 字段配置
	CategoryCode string            `json:"category_code" gorm:"size:10;index"`   // 01/02/03/04
	FieldGroupID *uint             `json:"field_group_id" gorm:"index"`          // 关联专用字段组
	SortOrder    int               `json:"sort_order" gorm:"default:0"`
	Fields       []ArchiveFieldDefinition `json:"fields,omitempty" gorm:"foreignKey:SubCategoryID;constraint:OnDelete:CASCADE"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Document 档案主表
type Document struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID uint  `json:"user_id" gorm:"index"`
	User   *User `json:"-" gorm:"foreignKey:UserID"`

	// 编号信息
	DocumentCode    string `json:"document_code" gorm:"uniqueIndex"` // WS-01-2026-001
	CategoryCode    string `json:"category_code" gorm:"index"`       // WS
	SubCategoryCode string `json:"sub_category_code" gorm:"index"`   // 01
	Year            int    `json:"year" gorm:"index"`                // 2026
	Sequence        int    `json:"sequence" gorm:"index"`            // 001

	// 基本信息
	FileName        string     `json:"file_name" gorm:"size:500"`       // 文件名/标题
	DocumentType    string     `json:"document_type" gorm:"size:100"`   // 一级分类名称
	SubType         string     `json:"sub_type" gorm:"size:100"`        // 二级分类名称
	Summary         string     `json:"summary" gorm:"type:text"`        // 摘要描述
	Tags            string     `json:"tags" gorm:"type:text"`          // 标签数组 JSON 字符串
	SignedDate      *time.Time `json:"signed_date"`                     // 签署/形成日期
	ExpirationDate  *time.Time `json:"expiration_date"`                 // 到期日期
	RetentionPeriod string     `json:"retention_period" gorm:"size:20"` // 保管期限: 永久/30年/10年

	// 自定义字段（JSON）
	CustomFields datatypes.JSON `json:"custom_fields" gorm:"type:json"`

	// 文书档案字段
	PartyA          string  `json:"party_a" gorm:"size:200"`         // 甲方
	PartyB          string  `json:"party_b" gorm:"size:200"`         // 乙方
	Amount          float64 `json:"amount"`                          // 金额
	PaymentProgress string  `json:"payment_progress" gorm:"size:20"` // 付款进度

	// 科技档案字段
	ProjectName    string     `json:"project_name" gorm:"size:300"`    // 项目名称
	DesignUnit     string     `json:"design_unit" gorm:"size:200"`     // 设计单位
	Designer       string     `json:"designer" gorm:"size:100"`        // 设计人员
	ProjectLeader  string     `json:"project_leader" gorm:"size:100"`  // 项目负责人
	EquipmentName  string     `json:"equipment_name" gorm:"size:200"`  // 设备名称
	EquipmentModel string     `json:"equipment_model" gorm:"size:100"` // 设备型号
	PurchaseDate   *time.Time `json:"purchase_date"`                   // 采购日期

	// 电子档案字段
	ContentDescription string     `json:"content_description" gorm:"type:text"` // 内容描述
	CaptureDate        *time.Time `json:"capture_date"`                         // 拍摄/录制日期
	Capturer           string     `json:"capturer" gorm:"size:100"`             // 拍摄人
	ActivityName       string     `json:"activity_name" gorm:"size:200"`        // 活动/会议名称
	CarrierType        string     `json:"carrier_type" gorm:"size:50"`          // 载体类型
	StorageLocation    string     `json:"storage_location" gorm:"size:200"`     // 存储位置
	FileFormat         string     `json:"file_format" gorm:"size:50"`           // 文件格式

	// 专门档案字段
	Petitioner     string     `json:"petitioner" gorm:"size:200"`   // 来信人/单位
	Respondent     string     `json:"respondent" gorm:"size:200"`   // 被反映人
	AuditUnit      string     `json:"audit_unit" gorm:"size:200"`   // 审计单位
	AuditPeriod    string     `json:"audit_period" gorm:"size:50"`  // 审计期间
	Counterparty   string     `json:"counterparty" gorm:"size:200"` // 案件对方
	Lawyer         string     `json:"lawyer" gorm:"size:100"`       // 律师
	Winner         string     `json:"winner" gorm:"size:200"`       // 中标单位
	StartDate      *time.Time `json:"start_date"`                   // 立项/开始日期
	CompletionDate *time.Time `json:"completion_date"`              // 竣工/结束日期

	// 文件信息
	FilePath         string `json:"file_path" gorm:"size:500"`          // 存储路径
	FileNameOriginal string `json:"file_name_original" gorm:"size:500"` // 原始文件名
	FileSize         int64  `json:"file_size"`                          // 文件大小
	FileType         string `json:"file_type" gorm:"size:50"`           // 文件类型
	FileID           string `json:"file_id" gorm:"size:100"`            // 存储服务返回的文件ID

	// 知识库相关
	ContentText string `json:"content_text" gorm:"type:text"`                    // OCR 提取的文本内容
	OCRStatus   string `json:"ocr_status" gorm:"size:20;default:none"`           // OCR 状态

	// 其他
	Remarks   string    `json:"remarks" gorm:"type:text"`             // 备注
	Status    string    `json:"status" gorm:"size:20;default:active"` // 状态
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExpirationReminder 到期提醒设置
type ExpirationReminder struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	UserID           uint      `json:"user_id" gorm:"index"`
	User             *User     `json:"-" gorm:"foreignKey:UserID"`
	DocumentCategory string    `json:"document_category" gorm:"size:10"` // 档案分类，空表示全部
	RemindDays       int       `json:"remind_days"`                      // 提前提醒天数
	Enabled          bool      `json:"enabled"`                          // 是否启用
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ArchiveSharedField 共用字段定义
type ArchiveSharedField struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	FieldName    string    `json:"field_name" gorm:"size:50;uniqueIndex"`
	FieldLabel   string    `json:"field_label" gorm:"size:200"`
	FieldType    string    `json:"field_type" gorm:"size:20"`          // text/number/date/select/file/textarea/multiselect/user
	IsRequired   bool      `json:"is_required" gorm:"default:false"`
	IsOCRRelated bool      `json:"is_ocr_related" gorm:"default:false"`
	Options      string    `json:"options" gorm:"type:text"`           // 下拉选项逗号分隔
	DefaultValue string    `json:"default_value" gorm:"size:200"`
	Placeholder  string    `json:"placeholder" gorm:"size:200"`
	SortOrder    int       `json:"sort_order" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ArchiveFieldGroup 档案字段分组（用于分类管理自定义字段）
type ArchiveFieldGroup struct {
	ID              uint                     `json:"id" gorm:"primaryKey"`
	SubCategoryID   uint                     `json:"sub_category_id" gorm:"index"`
	SubCategory     *DocumentSubCategory     `json:"-" gorm:"foreignKey:SubCategoryID"`
	SubCategoryCode string                   `json:"sub_category_code" gorm:"size:20;index"` // 0101/0102...
	Name            string                   `json:"name" gorm:"size:100"`       // 分组名称
	Description     string                   `json:"description" gorm:"size:200"` // 分组描述
	SortOrder       int                      `json:"sort_order" gorm:"default:0"`
	Fields          []ArchiveFieldDefinition  `json:"fields" gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

// ConditionConfig 条件显示配置
type ConditionConfig struct {
	FieldName   string `json:"field_name"`   // 触发条件的字段名
	Operator    string `json:"operator"`    // 操作符：equals/contains/gt/lt/in/not_empty
	Value       string `json:"value"`       // 期望值
}

// ArchiveFieldDefinition 档案自定义字段定义
type ArchiveFieldDefinition struct {
	ID             uint             `json:"id" gorm:"primaryKey"`
	SubCategoryID  uint             `json:"sub_category_id" gorm:"index"`
	SubCategory    *DocumentSubCategory `json:"-" gorm:"foreignKey:SubCategoryID"`
	GroupID        *uint            `json:"group_id" gorm:"index"`                  // 关联分组，可为空
	Group          *ArchiveFieldGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	FieldName      string           `json:"field_name" gorm:"size:50"`               // 字段英文名（唯一标识）
	FieldLabel     string           `json:"field_label" gorm:"size:100"`             // 显示名称
	FieldType      string           `json:"field_type" gorm:"size:20"`               // 类型：text/textarea/number/date/select/multiselect/checkbox
	Required       bool             `json:"required"`                                // 是否必填
	DefaultValue   string           `json:"default_value" gorm:"size:200"`           // 默认值
	Options        string           `json:"options" gorm:"type:text"`               // 下拉选项（逗号分隔）
	Placeholder    string           `json:"placeholder" gorm:"size:200"`             // 占位提示
	SortOrder      int              `json:"sort_order" gorm:"default:0"`             // 排序
	Visible        bool             `json:"visible" gorm:"default:true"`             // 是否可见
	Editable       bool             `json:"editable" gorm:"default:true"`            // 是否可编辑
	IsOCRRelated   bool             `json:"is_ocr_related" gorm:"default:false"`     // 是否可OCR填充
	ConditionConfig *ConditionConfig `json:"condition_config,omitempty" gorm:"type:json"` // 条件显示配置
	HelpText       string           `json:"help_text" gorm:"size:500"`               // 帮助说明文本
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// ShareLink 分享链接
type ShareLink struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Token       string    `json:"token" gorm:"size:64;uniqueIndex"` // 分享token
	UserID      uint      `json:"user_id" gorm:"index"`            // 创建者
	DocumentID  uint      `json:"document_id" gorm:"index"`       // 关联的文档
	Document    *Document `json:"document,omitempty" gorm:"foreignKey:DocumentID"`
	ExpiresAt   time.Time `json:"expires_at" gorm:"index"`        // 过期时间
	CreatedAt   time.Time `json:"created_at"`
}

// IsExpired 检查链接是否过期
func (s *ShareLink) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// ============ 档案配置模型 ============

// RetentionPeriod 保管期限配置
type RetentionPeriod struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`          // 用户ID，NULL表示全局配置
	User      *User     `json:"-" gorm:"foreignKey:UserID"`
	Name      string    `json:"name" gorm:"size:50;not null"` // 保管期限名称
	Years     int       `json:"years" gorm:"default:0"`       // 保管年限，0表示永久
	SortOrder int       `json:"sort_order" gorm:"default:0"`  // 排序
	IsDefault bool      `json:"is_default" gorm:"default:false"` // 是否为默认选项
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StorageLocation 存档地点配置
type StorageLocation struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      *uint     `json:"user_id" gorm:"index"`            // 用户ID，NULL表示全局配置
	User        *User     `json:"-" gorm:"foreignKey:UserID"`
	Name        string    `json:"name" gorm:"size:100;not null"`    // 存档地点名称
	Description string    `json:"description" gorm:"size:500"`     // 描述
	SortOrder   int       `json:"sort_order" gorm:"default:0"`     // 排序
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CodeRule 编码规则配置
type CodeRule struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:100"`                 // 规则名称
	CodeFormat  string    `json:"code_format" gorm:"size:200"`           // 编码格式模式
	Separator   string    `json:"separator" gorm:"size:10;default:-"`    // 分隔符
	Description string    `json:"description" gorm:"size:500"`          // 规则描述
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CodeRulePlaceholder 编码规则占位符配置
type CodeRulePlaceholder struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	RuleID   uint   `json:"rule_id" gorm:"index;uniqueIndex"` // 关联规则ID
	Rule     *CodeRule `json:"-" gorm:"foreignKey:RuleID"`
	Placeholder string `json:"placeholder" gorm:"size:50;not null"`  // 占位符名称，如 {YYYY}, {SEQ}, {CATEGORY_CODE}
	Label      string `json:"label" gorm:"size:100"`                // 显示标签
	IsAuto     bool   `json:"is_auto" gorm:"default:true"`          // 是否自动生成
	Format     string `json:"format" gorm:"size:50"`                // 格式化模板，如 0000（序号）
	SortOrder  int    `json:"sort_order" gorm:"default:0"`          // 排序
}

// ArchiveConfig 档案全局配置
type ArchiveConfig struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	UserID              *uint     `json:"user_id" gorm:"uniqueIndex"`        // 用户ID，NULL表示全局配置
	User                *User     `json:"-" gorm:"foreignKey:UserID"`
	DefaultCodeRuleID   *uint     `json:"default_code_rule_id" gorm:"index"` // 默认编码规则ID
	DefaultCodeRule     *CodeRule `json:"-" gorm:"foreignKey:DefaultCodeRuleID"`
	AutoGenerateCode    bool      `json:"auto_generate_code" gorm:"default:true"`  // 是否自动生成编号
	RequireCodePrefix   bool      `json:"require_code_prefix" gorm:"default:true"` // 是否要求编号前缀
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
