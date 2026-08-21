// E2E 测试账号 Seed CLI —— 创建 4 个固定角色测试账号（admin/manager/editor/viewer）
//
// 用法:
//
//	go run ./cmd/seed-e2e
//
// 无参数、幂等：账号已存在则跳过（不覆盖密码），重复运行安全。
// 每个账号创建后建立 user_roles 关联（User.Role 字段已废弃，必须走关联表）。
// 同时确保存在一条全局启用（user_id IS NULL）的 LLM 配置（幂等），
// 解除 feedback-closure E2E 的"未找到可用的 LLM 配置"阻塞。
// 全程不打印密码明文，仅输出用户名与角色。
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"siapp/internal/models"
)

// e2eAccount 描述一个 E2E 测试账号的固定信息。
// Password 仅在首次创建时用于 bcrypt 哈希，任何输出路径均不打印明文。
type e2eAccount struct {
	Username string
	Password string
	Email    string
	FullName string
	RoleName string
}

// e2eAccounts 定义 4 个固定 E2E 测试账号（固定密码与 Playwright 测试配置保持一致）
var e2eAccounts = []e2eAccount{
	{Username: "admin", Password: "Admin@123456", Email: "admin@e2e.local", FullName: "E2E 管理员", RoleName: models.RoleAdmin},
	{Username: "manager", Password: "Manager@123456", Email: "manager@e2e.local", FullName: "E2E 部门经理", RoleName: models.RoleManager},
	{Username: "editor", Password: "Editor@123456", Email: "editor@e2e.local", FullName: "E2E 编辑者", RoleName: models.RoleEditor},
	{Username: "viewer", Password: "Viewer@123456", Email: "viewer@e2e.local", FullName: "E2E 只读用户", RoleName: models.RoleViewer},
}

// ───────────────────── 入口 ─────────────────────

func main() {
	dsn := resolveDSN()
	db := connectDB(dsn)

	// 自动建表，保证在全新库（如 CI E2E 环境）也能直接运行；已存在的表为 no-op
	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserRole{},
		&models.ModelConfig{},
		&models.Employee{},
		&models.StorageConfig{},
		&models.StorageModuleConfig{},
		&models.StorageRule{},
		// P12.3.2 入职 E2E 数据底座：仅建表，不预建任何 onboarding 记录
		&models.Department{},
		&models.OnboardingRecord{},
		&models.WorkTodo{},
		&models.OnboardingImportRun{},
		// P12.3.3 转正 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.RegularizationRecord{},
		&models.RegularizationEffectRun{},
		// P12.3.2 劳动合同 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.LaborContract{},
		// P12.3.5 行政合同 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.AdminContract{},
		// P12.3.6 奖惩记录 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.RewardRecord{},
		// P12.3.7 人事异动 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.PersonnelChange{},
		// P12.3.8 培训管理 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.TrainingRecord{},
		// P12 最小真实功能：职业卫生检查 E2E 数据底座
		&models.OccupationalHealthCheck{},
		// P12.3.9 安全管理 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.SafetyInspection{},
		// P12 车队管理 E2E 数据底座：仅建表，记录由 E2E 流程创建
		&models.FleetVehicle{},
	); err != nil {
		log.Fatalf("AutoMigrate 表失败: %v", err)
	}

	// 确保 4 个核心角色存在（幂等）
	if err := ensureRoles(db); err != nil {
		log.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		log.Fatalf("初始化权限失败: %v", err)
	}

	fmt.Println("═══ E2E 测试账号 Seed ═══")
	created, skipped, err := seedAccounts(db)
	if err != nil {
		log.Fatalf("Seed 账号失败: %v", err)
	}

	// 确保存在一名固定 E2E 在职员工（供前端离职流程 E2E 使用，幂等）。
	// 挂在 admin 账号下（admin 拥有 employee.edit，可操作离职/恢复流程）。
	var adminUser models.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		log.Fatalf("查询 admin 用户失败: %v", err)
	}
	if err := ensureE2EEmployee(db, adminUser.ID); err != nil {
		log.Fatalf("初始化 E2E 员工失败: %v", err)
	}

	// 确保存在 admin 所属的 E2E 测试部门（供入职流程 E2E 使用，幂等）。
	// 仅操作精确匹配 E2E 部门名的记录；同名部门归属非 admin 时返回明确错误，不触碰非 E2E 数据。
	if err := ensureE2EDepartment(db, adminUser.ID); err != nil {
		log.Fatalf("初始化 E2E 测试部门失败: %v", err)
	}

	// 确保转正 E2E 三名审批用户同租户（admin/manager/editor，幂等）。
	// 仅更新精确匹配用户名的 company_id，不触碰其他用户。
	if err := ensureE2ERegularizationUsers(db); err != nil {
		log.Fatalf("初始化转正审批用户租户失败: %v", err)
	}

	// 确保存在两名稳定试用期员工（供转正流程 E2E 使用，幂等）。
	// 挂在 admin 账号下（admin 拥有 employee.edit，可发起转正申请）。
	if err := ensureE2ETrialEmployees(db, adminUser.ID); err != nil {
		log.Fatalf("初始化转正 E2E 员工失败: %v", err)
	}

	// 确保存在一名固定在职员工（供奖惩记录 E2E 使用，幂等）。
	// 挂在 admin 账号下（admin 拥有 reward.create/edit/delete，可创建/生效/作废奖惩记录）。
	// 独立于离职/转正员工，避免被其他流程 E2E 变更状态，保证「奖惩不改变员工状态」验收稳定成立。
	if err := ensureE2ERewardEmployee(db, adminUser.ID); err != nil {
		log.Fatalf("初始化奖惩 E2E 员工失败: %v", err)
	}

	// 确保人事异动 E2E 的基线员工和目标部门存在（幂等）。
	// 记录由 E2E 流程创建；每次 seed 恢复员工异动前资料，避免跨次执行互相影响。
	if err := ensureE2EPersonnelChangeData(db, adminUser.ID); err != nil {
		log.Fatalf("初始化人事异动 E2E 数据失败: %v", err)
	}
	if err := ensureE2ETrainingEmployee(db, adminUser.ID); err != nil {
		log.Fatalf("初始化培训 E2E 员工失败: %v", err)
	}

	// 确保存在 E2E 本地测试存储配置（离职证明上传所需，幂等）。
	// 仅创建/恢复 Name 明确标记为 E2E 的记录，不触碰非 E2E/既有配置；
	// 存储根目录固定在 /tmp 隔离临时范围（可清理），不写入仓库或生产路径。
	if _, err := ensureE2EStorage(db); err != nil {
		log.Fatalf("初始化 E2E 存储配置失败: %v", err)
	}

	// 确保存在可用的全局 LLM 配置（幂等），解除 feedback-closure E2E 的 LLM 配置阻塞
	if err := ensureGlobalLLMConfig(db); err != nil {
		log.Fatalf("初始化全局 LLM 配置失败: %v", err)
	}

	fmt.Println()
	fmt.Println("─── 汇总 ───")
	fmt.Printf("  创建: %d\n", created)
	fmt.Printf("  跳过(已存在): %d\n", skipped)
	fmt.Println("  账号固定密码请参考 e2e 测试配置（本命令不打印密码明文）")
}

// ───────────────────── 数据库连接 ─────────────────────

// resolveDSN 按优先级获取数据库连接字符串：SIAPP_DATABASE_PATH → DATABASE_URL
func resolveDSN() string {
	if v := os.Getenv("SIAPP_DATABASE_PATH"); v != "" {
		return v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	fmt.Fprintln(os.Stderr, "错误: 未设置数据库连接。请设置 SIAPP_DATABASE_PATH 或 DATABASE_URL 环境变量")
	os.Exit(1)
	return ""
}

// connectDB 根据 DSN 自动选择驱动建立连接（限定 Logger 减少噪音）
func connectDB(dsn string) *gorm.DB {
	var dialector gorm.Dialector
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		dialector = postgres.Open(dsn)
	} else {
		dialector = sqlite.Open(dsn)
	}

	// 静默 GORM 日志，避免 seed 阶段大量 SQL 干扰输出
	silentLogger := logger.New(log.New(io.Discard, "", 0), logger.Config{})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: silentLogger,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	return db
}

// ───────────────────── 角色与账号 Seed ─────────────────────

// ensureRoles 确保 4 个核心角色存在于 roles 表（按 name 去重，已存在跳过）
func ensureRoles(db *gorm.DB) error {
	roles := []models.Role{
		{Name: models.RoleAdmin, Label: "管理员", Description: "系统管理权限", IsSystem: true},
		{Name: models.RoleManager, Label: "部门经理", Description: "本部门全模块查看、创建、编辑", IsSystem: true},
		{Name: models.RoleEditor, Label: "编辑者", Description: "全模块查看、编辑（不可创建/删除）", IsSystem: true},
		{Name: models.RoleViewer, Label: "只读用户", Description: "仅查看数据", IsSystem: true},
	}

	for _, r := range roles {
		var existing models.Role
		if err := db.Where("name = ?", r.Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&r).Error; err != nil {
				return fmt.Errorf("创建角色 %s 失败: %w", r.Name, err)
			}
		}
	}
	return nil
}

// loadRoleIDs 加载角色名→ID 映射
func loadRoleIDs(db *gorm.DB) (map[string]uint, error) {
	var roles []models.Role
	if err := db.Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}
	m := make(map[string]uint, len(roles))
	for _, r := range roles {
		m[r.Name] = r.ID
	}
	return m, nil
}

// seedAccounts 逐账号创建（已存在则跳过），返回创建数与跳过数
func seedAccounts(db *gorm.DB) (created, skipped int, err error) {
	roleIDs, err := loadRoleIDs(db)
	if err != nil {
		return 0, 0, err
	}

	for _, acc := range e2eAccounts {
		roleID, ok := roleIDs[acc.RoleName]
		if !ok {
			return 0, 0, fmt.Errorf("角色 %q 在 roles 表中不存在，请先执行 seed", acc.RoleName)
		}

		// 幂等：用户名已存在则跳过，不覆盖密码
		var existing models.User
		if err := db.Where("username = ?", acc.Username).First(&existing).Error; err == nil {
			fmt.Printf("  [跳过] %s（角色: %s）已存在\n", acc.Username, acc.RoleName)
			skipped++
			continue
		} else if err != gorm.ErrRecordNotFound {
			return 0, 0, fmt.Errorf("查询用户 %s 失败: %w", acc.Username, err)
		}

		if err := createAccountWithRole(db, acc, roleID); err != nil {
			return 0, 0, err
		}
		fmt.Printf("  [创建] %s（角色: %s）\n", acc.Username, acc.RoleName)
		created++
	}
	return created, skipped, nil
}

// createAccountWithRole 创建用户并建立 UserRole 关联（User.Role 字段已废弃，必须走关联表）
func createAccountWithRole(db *gorm.DB, acc e2eAccount, roleID uint) error {
	u := models.User{
		Username:      acc.Username,
		Email:         acc.Email,
		FullName:      acc.FullName,
		Active:        true,
		EmailVerified: true,
	}
	if err := u.SetPassword(acc.Password); err != nil {
		return fmt.Errorf("用户 %s 密码哈希失败: %w", acc.Username, err)
	}
	if err := db.Create(&u).Error; err != nil {
		return fmt.Errorf("创建用户 %s 失败: %w", acc.Username, err)
	}
	ur := models.UserRole{UserID: u.ID, RoleID: roleID}
	if err := db.Create(&ur).Error; err != nil {
		return fmt.Errorf("创建用户角色关联 %s 失败: %w", acc.Username, err)
	}
	return nil
}

// ───────────────────── E2E 员工 Seed ─────────────────────

// E2E 离职流程测试员工的固定信息。
// IDNumber 为虚构测试身份证号（非真实个人信息），仅用于幂等定位与前端 E2E 识别；
// 全程不打印身份证号明文，避免敏感信息落日志。
const (
	e2eEmployeeName       = "E2E离职测试员工"
	e2eEmployeeIDNumber   = "110101199001011234" // 虚构测试身份证号，稳定唯一
	e2eEmployeeDepartment = "E2E测试部"
	e2eEmployeePosition   = "测试专员"
)

// ensureE2EEmployee 幂等创建/恢复一名固定 E2E 在职员工（供前端离职流程 E2E 使用）。
// 按 (user_id, id_number) 唯一索引定位：
//   - 不存在则创建（status=active）；
//   - 已存在（含旧残留）则强制恢复为 active 并清空离职字段，保证重复运行安全。
//
// 仅操作精确匹配 (user_id, id_number) 的记录，不删除、不触碰其他员工数据。
func ensureE2EEmployee(db *gorm.DB, userID uint) error {
	var employee models.Employee
	err := db.Where("user_id = ? AND id_number = ?", userID, e2eEmployeeIDNumber).First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		employee = models.Employee{
			UserID:     userID,
			Name:       e2eEmployeeName,
			IDNumber:   e2eEmployeeIDNumber,
			Department: e2eEmployeeDepartment,
			Position:   e2eEmployeePosition,
			Status:     "active",
		}
		if err := db.Create(&employee).Error; err != nil {
			return fmt.Errorf("创建 E2E 员工失败: %w", err)
		}
		fmt.Printf("  [创建] E2E 员工（姓名: %s，部门: %s，岗位: %s，状态: active）\n",
			e2eEmployeeName, e2eEmployeeDepartment, e2eEmployeePosition)
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询 E2E 员工失败: %w", err)
	}

	// 已存在：恢复为 active 并清空离职字段（幂等恢复目标状态）
	updates := map[string]any{
		"name":              e2eEmployeeName,
		"department":        e2eEmployeeDepartment,
		"position":          e2eEmployeePosition,
		"status":            "active",
		"resign_date":       "",
		"resign_proof_path": "",
		"resign_proof_name": "",
		"resign_reasons":    "",
		"updated_at":        time.Now(),
	}
	if err := db.Model(&employee).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复 E2E 员工失败: %w", err)
	}
	fmt.Printf("  [恢复] E2E 员工（姓名: %s，部门: %s，岗位: %s，状态: active）\n",
		e2eEmployeeName, e2eEmployeeDepartment, e2eEmployeePosition)
	return nil
}

// ───────────────────── E2E 存储配置 Seed ─────────────────────

// E2E 离职证明上传所需的本地测试存储配置。
// 仅操作 Name/ModuleName 明确标记为 E2E 的记录，绝不覆盖或删除非 E2E/既有配置。
// 存储根目录固定在 /tmp 隔离临时范围（系统可清理），不写入仓库或生产路径。
const (
	e2eStorageConfigName   = "E2E本地测试存储"
	e2eStorageModuleName   = "E2E-archives模块配置"
	e2eStorageRuleName     = "E2E-archives/resign_proof规则"
	e2eStorageRootPath     = "/tmp/siapp-e2e-storage"
	e2eStorageRulePriority = 100 // 高优先级，确保精确规则优先于模块默认/全局默认
)

// ensureE2EStorage 幂等创建/恢复 E2E 本地测试存储配置、archives 模块启用配置与
// archives/resign_proof 规则，返回 E2E 存储配置 ID（供规则引用）。
// 解析链路：StorageRouter.Resolve(archives, resign_proof) 命中精确规则 → 指向 E2E 本地配置。
func ensureE2EStorage(db *gorm.DB) (uint, error) {
	// 1. 确保本地存储根目录存在（/tmp 隔离范围，可清理）
	if err := os.MkdirAll(e2eStorageRootPath, 0o755); err != nil {
		return 0, fmt.Errorf("创建 E2E 存储目录失败: %w", err)
	}

	// 2. 本地存储配置（按 Name 幂等定位）
	config, err := upsertE2EStorageConfig(db)
	if err != nil {
		return 0, err
	}

	// 3. archives 模块启用配置（按 ModuleCode+ModuleName 幂等定位）
	if err := upsertE2EStorageModuleConfig(db); err != nil {
		return 0, err
	}

	// 4. archives/resign_proof 精确规则（按 ModuleCode+ResourceType+Name 幂等定位）
	if err := upsertE2EStorageRule(db, config.ID); err != nil {
		return 0, err
	}

	return config.ID, nil
}

// upsertE2EStorageConfig 幂等创建/恢复 E2E 本地存储配置
func upsertE2EStorageConfig(db *gorm.DB) (*models.StorageConfig, error) {
	var config models.StorageConfig
	err := db.Where("name = ?", e2eStorageConfigName).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		config = models.StorageConfig{
			Name:        e2eStorageConfigName,
			Type:        "local",
			Enabled:     true,
			IsDefault:   false,
			IsBackup:    false,
			Status:      "active",
			Config:      datatypes.JSON([]byte(`{"root_path":"` + e2eStorageRootPath + `"}`)),
			Description: "E2E 离职证明上传本地测试存储（隔离临时目录，可清理）",
		}
		if err := db.Create(&config).Error; err != nil {
			return nil, fmt.Errorf("创建 E2E 存储配置失败: %w", err)
		}
		fmt.Printf("  [创建] E2E 存储配置（名称: %s，类型: local，状态: active）\n", config.Name)
		return &config, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 E2E 存储配置失败: %w", err)
	}

	// 已存在：恢复可用（enabled=true, status=active），并确保 root_path 指向隔离目录
	updates := map[string]any{
		"type":        "local",
		"enabled":     true,
		"is_default":  false,
		"status":      "active",
		"config":      datatypes.JSON([]byte(`{"root_path":"` + e2eStorageRootPath + `"}`)),
		"description": "E2E 离职证明上传本地测试存储（隔离临时目录，可清理）",
		"updated_at":  time.Now(),
	}
	if err := db.Model(&config).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("恢复 E2E 存储配置失败: %w", err)
	}
	fmt.Printf("  [恢复] E2E 存储配置（名称: %s，类型: local，状态: active）\n", config.Name)
	return &config, nil
}

// upsertE2EStorageModuleConfig 幂等创建/恢复 archives 模块启用配置
func upsertE2EStorageModuleConfig(db *gorm.DB) error {
	var module models.StorageModuleConfig
	err := db.Where("module_code = ? AND module_name = ?", "archives", e2eStorageModuleName).First(&module).Error
	if err == gorm.ErrRecordNotFound {
		module = models.StorageModuleConfig{
			ModuleCode:    "archives",
			ModuleName:    e2eStorageModuleName,
			BaseDirectory: e2eStorageRootPath,
			Description:   "E2E archives 模块存储启用配置（隔离临时目录，可清理）",
			Enabled:       true,
		}
		if err := db.Create(&module).Error; err != nil {
			return fmt.Errorf("创建 E2E archives 模块配置失败: %w", err)
		}
		fmt.Printf("  [创建] E2E archives 模块配置（模块: archives，状态: enabled）\n")
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询 E2E archives 模块配置失败: %w", err)
	}

	// 已存在：恢复启用
	updates := map[string]any{
		"base_directory": e2eStorageRootPath,
		"description":    "E2E archives 模块存储启用配置（隔离临时目录，可清理）",
		"enabled":        true,
		"updated_at":     time.Now(),
	}
	if err := db.Model(&module).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复 E2E archives 模块配置失败: %w", err)
	}
	fmt.Printf("  [恢复] E2E archives 模块配置（模块: archives，状态: enabled）\n")
	return nil
}

// upsertE2EStorageRule 幂等创建/恢复 archives/resign_proof 精确规则
func upsertE2EStorageRule(db *gorm.DB, storageID uint) error {
	var rule models.StorageRule
	err := db.Where("module_code = ? AND resource_type = ? AND name = ?", "archives", "resign_proof", e2eStorageRuleName).First(&rule).Error
	if err == gorm.ErrRecordNotFound {
		rule = models.StorageRule{
			StorageID:    storageID,
			ModuleCode:   "archives",
			ResourceType: "resign_proof",
			Priority:     e2eStorageRulePriority,
			Enabled:      true,
			Name:         e2eStorageRuleName,
			TargetType:   "document",
		}
		if err := db.Create(&rule).Error; err != nil {
			return fmt.Errorf("创建 E2E 存储规则失败: %w", err)
		}
		fmt.Printf("  [创建] E2E 存储规则（模块: archives，资源类型: resign_proof，优先级: %d）\n", e2eStorageRulePriority)
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询 E2E 存储规则失败: %w", err)
	}

	// 已存在：恢复可用并指向 E2E 存储配置
	updates := map[string]any{
		"storage_id":  storageID,
		"priority":    e2eStorageRulePriority,
		"enabled":     true,
		"target_type": "document",
		"updated_at":  time.Now(),
	}
	if err := db.Model(&rule).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复 E2E 存储规则失败: %w", err)
	}
	fmt.Printf("  [恢复] E2E 存储规则（模块: archives，资源类型: resign_proof，优先级: %d）\n", e2eStorageRulePriority)
	return nil
}

// ───────────────────── 全局 LLM 配置 Seed ─────────────────────

// 默认 Siliconflow 测试 LLM 的固定值（E2E_LLM_* 环境变量未设置时回退到这些值）
const (
	globalLLMModelDefault    = "Qwen/Qwen3-8B"
	globalLLMEndpointDefault = "https://api.siliconflow.cn/v1"
	globalLLMKeyDefault      = "sk-lqmkmhmzqhynebyaseuaduwvedicorvqnvoqmqldpkpkjfhi"
)

// ensureGlobalLLMConfig 确保存在一条全局启用（user_id IS NULL）的 LLM 配置（幂等）。
// ChatService.GetLLMConfig 查询策略：先查当前用户自有配置，查不到再回退到全局配置；
// viewer 等无自有配置的账号依赖全局配置才能走通 /api/knowledge/chat/stream，
// 否则会报"未找到可用的 LLM 配置"。
// 字段值优先取环境变量 E2E_LLM_MODEL / E2E_LLM_ENDPOINT / E2E_LLM_API_KEY，
// 未设置时使用内置默认值；全程不打印 APIKey 明文。
func ensureGlobalLLMConfig(db *gorm.DB) error {
	// 幂等：已存在全局启用 LLM 配置则跳过
	var existing models.ModelConfig
	err := db.Where("user_id IS NULL AND config_type = ? AND enabled = ?", "llm", true).First(&existing).Error
	if err == nil {
		fmt.Printf("  [跳过] 全局 LLM 配置已存在（模型: %s，provider: %s）\n", existing.ModelName, existing.Provider)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询全局 LLM 配置失败: %w", err)
	}

	cfg := models.ModelConfig{
		UserID:      nil, // nil 表示全局配置，所有用户可用
		ConfigType:  "llm",
		Provider:    "siliconflow",
		ModelName:   envOrDefault("E2E_LLM_MODEL", globalLLMModelDefault),
		APIEndpoint: envOrDefault("E2E_LLM_ENDPOINT", globalLLMEndpointDefault),
		APIKey:      envOrDefault("E2E_LLM_API_KEY", globalLLMKeyDefault),
		// 关闭 Qwen3 推理流（enable_thinking=false），避免反馈闭环 E2E 被推理 token 拖至 60 秒 HTTP 超时
		ExtraParams: datatypes.JSON(`{"enable_thinking": false}`),
		Enabled:     true,
		IsDefault:   true,
		Role:        "primary",
		IsBuiltIn:   false,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return fmt.Errorf("创建全局 LLM 配置失败: %w", err)
	}
	fmt.Printf("  [创建] 全局 LLM 配置（模型: %s，provider: %s，端点: %s）\n", cfg.ModelName, cfg.Provider, cfg.APIEndpoint)
	return nil
}

// envOrDefault 读取环境变量，未设置时返回默认值
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
