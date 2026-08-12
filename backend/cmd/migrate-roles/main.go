// 角色迁移 CLI —— 将 User.Role 字符串字段迁移到 user_roles 关联表
//
// 用法:
//
//	go run ./cmd/migrate-roles              # 默认 dry-run
//	go run ./cmd/migrate-roles --dry-run    # 显式 dry-run
//	go run ./cmd/migrate-roles --apply      # 实际执行迁移
//
// dry-run 模式（默认）：扫描所有 User 记录，打印每个用户将执行的迁移操作，
// 收集异常角色值并给出映射建议，汇总统计，不写入任何数据。
//
// --apply 模式：在事务中为每个 User 创建对应的 UserRole 记录（已存在则跳过），
// 同时确保 roles 表及默认权限映射已就绪。
package main

import (
	"flag"
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

// ───────────────────── 角色常量 ─────────────────────

var validRoleNames = map[string]bool{
	models.RoleAdmin:   true,
	models.RoleManager: true,
	models.RoleEditor:  true,
	models.RoleViewer:  true,
}

// ───────────────────── 异常记录 ─────────────────────

type anomalyRecord struct {
	UserID   uint
	Email    string
	OldValue string
	NewValue string
	Reason   string
}

// ───────────────────── 入口 ─────────────────────

func main() {
	dryRun := flag.Bool("dry-run", true, "试运行模式（默认），仅预览不写入")
	apply := flag.Bool("apply", false, "实际执行迁移写入")
	flag.Parse()

	// 两个 flag 互斥：显式传 --dry-run=false 等价于 --apply
	if *apply {
		*dryRun = false
	}

	dsn := resolveDSN()
	db := connectDB(dsn)

	// 自动建表确保 RBAC 表存在
	if err := db.AutoMigrate(&models.Role{}, &models.Permission{}, &models.RolePermission{}, &models.UserRole{}); err != nil {
		log.Fatalf("AutoMigrate RBAC 表失败: %v", err)
	}

	// 确保角色和默认权限就绪
	if err := ensureRoles(db); err != nil {
		log.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		log.Fatalf("初始化权限失败: %v", err)
	}

	// 执行迁移
	if *dryRun {
		fmt.Println("═══ DRY-RUN 模式 —— 仅预览，不写入任何数据 ═══")
		fmt.Println()
		runDryRun(db)
		fmt.Println()
		fmt.Println("确认无误后运行: go run ./cmd/migrate-roles --apply")
	} else {
		fmt.Println("═══ APPLY 模式 —— 开始执行迁移 ═══")
		fmt.Println()
		runApply(db)
		fmt.Println()
		fmt.Println("迁移完成。")
	}
}

// ───────────────────── 数据库连接 ─────────────────────

// resolveDSN 按优先级获取数据库连接字符串：--target → SIAPP_DATABASE_PATH → DATABASE_URL
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

	// 静默 GORM 日志，避免 seed 阶段大量 SQL 干扰迁移输出
	silentLogger := logger.New(log.New(io.Discard, "", 0), logger.Config{})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: silentLogger,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	return db
}

// ───────────────────── 角色与权限 Seed ─────────────────────

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

// ensurePermissions 确保默认权限数据及角色-权限映射存在（幂等）
func ensurePermissions(db *gorm.DB) error {
	if err := seedPermissions(db); err != nil {
		return err
	}
	return seedRolePermissions(db)
}

// seedPermissions 按 module+action 组合幂等创建权限记录
func seedPermissions(db *gorm.DB) error {
	perms := []models.Permission{
		{Module: "employee", Action: "view", Label: "查看", SortOrder: 1},
		{Module: "employee", Action: "create", Label: "创建", SortOrder: 2},
		{Module: "employee", Action: "edit", Label: "编辑", SortOrder: 3},
		{Module: "employee", Action: "delete", Label: "删除", SortOrder: 4},
		{Module: "insurance", Action: "view", Label: "查看", SortOrder: 10},
		{Module: "insurance", Action: "create", Label: "创建", SortOrder: 11},
		{Module: "insurance", Action: "edit", Label: "编辑", SortOrder: 12},
		{Module: "insurance", Action: "delete", Label: "删除", SortOrder: 13},
		{Module: "dormitory", Action: "view", Label: "查看", SortOrder: 20},
		{Module: "dormitory", Action: "create", Label: "创建", SortOrder: 21},
		{Module: "dormitory", Action: "edit", Label: "编辑", SortOrder: 22},
		{Module: "dormitory", Action: "delete", Label: "删除", SortOrder: 23},
		{Module: "archives", Action: "view", Label: "查看", SortOrder: 30},
		{Module: "archives", Action: "create", Label: "创建", SortOrder: 31},
		{Module: "archives", Action: "edit", Label: "编辑", SortOrder: 32},
		{Module: "archives", Action: "delete", Label: "删除", SortOrder: 33},
		{Module: "settings", Action: "view", Label: "查看", SortOrder: 40},
		{Module: "settings", Action: "create", Label: "创建", SortOrder: 41},
		{Module: "settings", Action: "edit", Label: "编辑", SortOrder: 42},
		{Module: "settings", Action: "delete", Label: "删除", SortOrder: 43},
		{Module: "announcements", Action: "view", Label: "查看", SortOrder: 50},
		{Module: "announcements", Action: "create", Label: "创建", SortOrder: 51},
		{Module: "announcements", Action: "edit", Label: "编辑", SortOrder: 52},
		{Module: "announcements", Action: "delete", Label: "删除", SortOrder: 53},
		{Module: "backups", Action: "view", Label: "查看", SortOrder: 60},
		{Module: "backups", Action: "create", Label: "创建", SortOrder: 61},
		{Module: "backups", Action: "edit", Label: "编辑", SortOrder: 62},
		{Module: "backups", Action: "delete", Label: "删除", SortOrder: 63},
		{Module: "users", Action: "view", Label: "查看", SortOrder: 70},
		{Module: "users", Action: "create", Label: "创建", SortOrder: 71},
		{Module: "users", Action: "edit", Label: "编辑", SortOrder: 72},
		{Module: "users", Action: "delete", Label: "删除", SortOrder: 73},
	}

	for _, p := range perms {
		var existing models.Permission
		if err := db.Where("module = ? AND action = ?", p.Module, p.Action).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&p).Error; err != nil {
				return fmt.Errorf("创建权限 %s-%s 失败: %w", p.Module, p.Action, err)
			}
		}
	}
	return nil
}

// seedRolePermissions 为 4 个核心角色分配默认权限（幂等）
func seedRolePermissions(db *gorm.DB) error {
	// admin：全部权限
	if err := assignAllToRole(db, models.RoleAdmin); err != nil {
		return err
	}
	// manager：所有模块 view/create/edit（不含 delete）
	if err := assignPermsToRole(db, models.RoleManager, []string{
		"employee-view", "employee-create", "employee-edit",
		"insurance-view", "insurance-create", "insurance-edit",
		"dormitory-view", "dormitory-create", "dormitory-edit",
		"archives-view", "archives-create", "archives-edit",
		"announcements-view", "announcements-create", "announcements-edit",
		"settings-view", "backups-view", "users-view",
	}); err != nil {
		return err
	}
	// editor：所有模块 view+edit
	if err := assignPermsToRole(db, models.RoleEditor, []string{
		"employee-view", "employee-edit",
		"insurance-view", "insurance-edit",
		"dormitory-view", "dormitory-edit",
		"archives-view", "archives-edit",
		"announcements-view", "announcements-edit",
		"settings-view", "backups-view",
	}); err != nil {
		return err
	}
	// viewer：所有业务模块 view（不含 settings/backups，验收标准：viewer 看不到系统设置）
	if err := assignPermsToRole(db, models.RoleViewer, []string{
		"employee-view", "insurance-view", "dormitory-view", "archives-view",
		"announcements-view",
	}); err != nil {
		return err
	}
	return nil
}

// assignAllToRole 将全部权限分配给指定角色
func assignAllToRole(db *gorm.DB, roleName string) error {
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("查找角色 %s 失败: %w", roleName, err)
	}
	var allPerms []models.Permission
	db.Find(&allPerms)
	for _, p := range allPerms {
		var existing models.RolePermission
		if err := db.Where("role_id = ? AND permission_id = ?", role.ID, p.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
			db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: p.ID})
		}
	}
	return nil
}

// assignPermsToRole 按 module-action 键列表为角色分配权限
func assignPermsToRole(db *gorm.DB, roleName string, permKeys []string) error {
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("查找角色 %s 失败: %w", roleName, err)
	}
	for _, key := range permKeys {
		parts := strings.SplitN(key, "-", 2)
		if len(parts) != 2 {
			continue
		}
		module, action := parts[0], parts[1]
		var perm models.Permission
		db.Where("module = ? AND action = ?", module, action).First(&perm)
		if perm.ID == 0 {
			continue
		}
		var existing models.RolePermission
		if err := db.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
			db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID})
		}
	}
	return nil
}

// ───────────────────── 角色规范化 ─────────────────────

// normalizeRole 将原始角色字符串映射为标准角色，返回 (新值, 是否异常, 原因)
func normalizeRole(raw string) (string, bool, string) {
	trimmed := strings.TrimSpace(raw)

	// 空值或 NULL → viewer
	if trimmed == "" {
		return models.RoleViewer, true, "空值或 NULL，映射为 viewer"
	}

	// 先走 models.NormalizeRole 的已有映射（如 super_admin → admin）
	mapped := models.NormalizeRole(trimmed)

	// 检查是否是合法角色值
	if validRoleNames[mapped] {
		// 发生了映射（如 super_admin → admin）也算异常
		if mapped != trimmed {
			return mapped, true, fmt.Sprintf("非标准值 %q，映射为 %s", trimmed, mapped)
		}
		return mapped, false, ""
	}

	// 未知值 → viewer
	return models.RoleViewer, true, fmt.Sprintf("未知角色值 %q，映射为 viewer", trimmed)
}

// ───────────────────── 迁移执行 ─────────────────────

// loadAllUsers 加载所有 User 记录（仅加载 id, username, email, role 字段）
func loadAllUsers(db *gorm.DB) ([]models.User, error) {
	var users []models.User
	if err := db.Select("id", "username", "email", "role").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}
	return users, nil
}

// loadRoleIDs 预加载角色名称→ID 映射
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

// userRoleExists 检查 user_roles 是否已存在记录（幂等）
func userRoleExists(db *gorm.DB, userID, roleID uint) bool {
	var ur models.UserRole
	err := db.Where("user_id = ? AND role_id = ?", userID, roleID).First(&ur).Error
	return err == nil
}

// runDryRun 试运行：扫描并输出迁移预览
func runDryRun(db *gorm.DB) {
	users, err := loadAllUsers(db)
	if err != nil {
		log.Fatalf("加载用户失败: %v", err)
	}

	fmt.Printf("扫描到 %d 个用户\n\n", len(users))

	var totalLegal, totalAnomaly int
	var anomalies []anomalyRecord

	for _, u := range users {
		newRole, isAnomaly, reason := normalizeRole(u.Role)
		if isAnomaly {
			totalAnomaly++
			anomalies = append(anomalies, anomalyRecord{
				UserID:   u.ID,
				Email:    u.Email,
				OldValue: u.Role,
				NewValue: newRole,
				Reason:   reason,
			})
		} else {
			totalLegal++
		}
		if isAnomaly {
			oldDisplay := u.Role
			if oldDisplay == "" {
				oldDisplay = "(空)"
			}
			fmt.Printf("  [异常] userID=%d  %s  %s → %s\n", u.ID, u.Email, oldDisplay, newRole)
		} else {
			fmt.Printf("  [合法] userID=%d  %s → %s\n", u.ID, u.Email, u.Role)
		}
	}

	// 异常汇总
	if len(anomalies) > 0 {
		fmt.Println()
		fmt.Println("─── 异常角色值清单 ───")
		for _, a := range anomalies {
			oldDisplay := a.OldValue
			if oldDisplay == "" {
				oldDisplay = "(空)"
			}
			fmt.Printf("  [异常] userID=%d  %s  原始值=%s → %s  原因: %s\n",
				a.UserID, a.Email, oldDisplay, a.NewValue, a.Reason)
		}
	}

	// 汇总统计
	fmt.Println()
	fmt.Println("─── 汇总统计 ───")
	fmt.Printf("  用户总数:     %d\n", len(users))
	fmt.Printf("  合法角色:     %d\n", totalLegal)
	fmt.Printf("  异常角色:     %d\n", totalAnomaly)
}

// runApply 实际执行迁移
func runApply(db *gorm.DB) {
	roleIDs, err := loadRoleIDs(db)
	if err != nil {
		log.Fatalf("加载角色映射失败: %v", err)
	}

	users, err := loadAllUsers(db)
	if err != nil {
		log.Fatalf("加载用户失败: %v", err)
	}

	var totalMigrated, totalSkipped int

	// 在事务中执行
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, u := range users {
			newRole, isAnomaly, reason := normalizeRole(u.Role)

			roleID, ok := roleIDs[newRole]
			if !ok {
				return fmt.Errorf("角色 %q 在 roles 表中不存在，请先运行 seed", newRole)
			}

			// 幂等：已存在则跳过
			if userRoleExists(tx, u.ID, roleID) {
				totalSkipped++
				continue
			}

			ur := models.UserRole{UserID: u.ID, RoleID: roleID}
			if err := tx.Create(&ur).Error; err != nil {
				return fmt.Errorf("为用户 %d (%s) 创建 UserRole 失败: %w", u.ID, u.Email, err)
			}

			totalMigrated++
			if isAnomaly {
				fmt.Printf("  [迁移-异常] userID=%d  %s  %q → %s (%s)\n",
					u.ID, u.Email, u.Role, newRole, reason)
			} else {
				fmt.Printf("  [迁移] userID=%d  %s → %s\n", u.ID, u.Email, newRole)
			}
		}
		return nil
	})

	if err != nil {
		// 事务已自动回滚
		log.Fatalf("迁移事务失败（已回滚）: %v", err)
	}

	fmt.Println()
	fmt.Println("─── 迁移结果 ───")
	fmt.Printf("  用户总数:     %d\n", len(users))
	fmt.Printf("  已迁移:       %d\n", totalMigrated)
	fmt.Printf("  已跳过(幂等): %d\n", totalSkipped)
}
