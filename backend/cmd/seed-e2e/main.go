// E2E 测试账号 Seed CLI —— 创建 4 个固定角色测试账号（admin/manager/editor/viewer）
//
// 用法:
//
//	go run ./cmd/seed-e2e
//
// 无参数、幂等：账号已存在则跳过（不覆盖密码），重复运行安全。
// 每个账号创建后建立 user_roles 关联（User.Role 字段已废弃，必须走关联表）。
// 全程不打印密码明文，仅输出用户名与角色。
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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
