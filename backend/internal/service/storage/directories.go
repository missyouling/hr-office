package storage

// StorageDirectory represents a storage directory in the system
type StorageDirectory struct {
	Key         string             `json:"key"`
	Label       string             `json:"label"`
	Path        string             `json:"path"`
	Description string             `json:"description"`
	Children    []StorageDirectory `json:"children,omitempty"`
}

// GetStorageDirectories returns the complete directory tree
func GetStorageDirectories() []StorageDirectory {
	return []StorageDirectory{
		{
			Key: "employee", Label: "员工管理", Path: "employee", Description: "员工相关文件",
			Children: []StorageDirectory{
				{Key: "employee/active", Label: "在职员工", Path: "employee/active", Description: "在职员工资料"},
				{Key: "employee/resigned", Label: "离职员工", Path: "employee/resigned", Description: "离职员工资料"},
				{Key: "employee/social_add", Label: "社保增减", Path: "employee/social_add", Description: "社保增减材料"},
				{Key: "employee/provident", Label: "公积金", Path: "employee/provident", Description: "公积金材料"},
			},
		},
		{
			Key: "social", Label: "社保管理", Path: "social", Description: "社保相关文件",
			Children: []StorageDirectory{
				{Key: "social/upload", Label: "社保上传", Path: "social/upload", Description: "社保上传文件"},
				{Key: "social/summary", Label: "社保总表", Path: "social/summary", Description: "社保汇总表"},
				{Key: "social/deduction", Label: "扣款明细", Path: "social/deduction", Description: "扣款明细文件"},
				{Key: "social/response", Label: "回盘记录", Path: "social/response", Description: "社保回盘记录"},
			},
		},
		{
			Key: "dormitory", Label: "宿舍管理", Path: "dormitory", Description: "宿舍相关文件",
			Children: []StorageDirectory{
				{Key: "dormitory/base_info", Label: "基础信息", Path: "dormitory/base_info", Description: "宿舍基础信息"},
				{Key: "dormitory/checkin", Label: "入住管理", Path: "dormitory/checkin", Description: "入住管理文件"},
				{Key: "dormitory/meter", Label: "抄表计费", Path: "dormitory/meter", Description: "抄表计费记录"},
				{Key: "dormitory/billing", Label: "账单中心", Path: "dormitory/billing", Description: "账单文件"},
			},
		},
		{
			Key: "daily", Label: "日常事务", Path: "daily", Description: "日常事务文件",
			Children: []StorageDirectory{},
		},
		{
			Key: "archive", Label: "档案管理", Path: "archive", Description: "档案管理文件（按年度归档）",
			Children: []StorageDirectory{},
		},
		{
			Key: "system", Label: "系统配置", Path: "system", Description: "系统配置文件",
			Children: []StorageDirectory{
				{Key: "system/db_backup", Label: "数据库备份", Path: "system/db_backup", Description: "数据库备份文件"},
				{Key: "system/logs", Label: "系统日志", Path: "system/logs", Description: "系统运行日志"},
				{Key: "system/config_backup", Label: "配置备份", Path: "system/config_backup", Description: "系统配置备份"},
				{Key: "system/notifications", Label: "通知公告", Path: "system/notifications", Description: "通知公告附件"},
			},
		},
	}
}
