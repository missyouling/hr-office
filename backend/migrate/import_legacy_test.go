package migrate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupTargetDB 创建临时 SQLite 目标库，并自动建表
func setupTargetDB(t *testing.T) *gorm.DB {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "target.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)

	// 自动建表（目标模型）
	err = db.AutoMigrate(
		&models.OfficeCategory{},
		&models.OfficeSupplier{},
		&models.OfficeSupply{},
		&models.OfficePurchase{},
		&models.OfficePurchaseItem{},
		&models.OfficePaymentRequest{},
		&models.CanteenCategory{},
		&models.CanteenSupply{},
		&models.CanteenExpenseCategory{},
		&models.CanteenPurchase{},
		&models.CanteenPurchaseItem{},
		&models.CanteenOtherExpense{},
		&models.CanteenDailyIncome{},
		&models.CanteenResourceFee{},
		&models.CanteenWeeklyMenu{},
		&models.CanteenMenuTemplate{},
		&models.CanteenCardRecharge{},
		&models.CanteenCardRefund{},
	)
	require.NoError(t, err)
	return db
}

// setupSourceDB 创建内存 SQLite 源库
func setupSourceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 创建源库表结构（与 office-supply-analytics 一致）
	ddls := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			sort_order INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now', '+8 hours')),
			updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS suppliers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			contact TEXT DEFAULT '',
			phone TEXT DEFAULT '',
			bank_name TEXT DEFAULT '',
			bank_account TEXT DEFAULT '',
			is_default INTEGER DEFAULT 0,
			remark TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now', '+8 hours')),
			updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS supplies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			spec TEXT DEFAULT '',
			unit TEXT DEFAULT '个',
			reference_price REAL DEFAULT 0,
			safety_stock INTEGER DEFAULT 0,
			category_id INTEGER,
			supplier_id INTEGER,
			status TEXT DEFAULT 'active',
			remark TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now', '+8 hours')),
			updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS purchases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_no TEXT NOT NULL UNIQUE,
			purchase_date TEXT NOT NULL,
			total_amount REAL NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			remark TEXT DEFAULT '',
			supplier_id INTEGER,
			supplier_name TEXT DEFAULT '',
			payment_status TEXT DEFAULT '未付款',
			payment_date TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now','+8 hours')),
			updated_at TEXT DEFAULT (datetime('now','+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS purchase_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			purchase_id INTEGER NOT NULL,
			supply_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			unit_price REAL NOT NULL,
			subtotal REAL NOT NULL,
			date TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS canteen_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			sort_order INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now', '+8 hours')),
			updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS canteen_supplies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			spec TEXT DEFAULT '',
			unit TEXT DEFAULT '斤',
			reference_price REAL DEFAULT 0,
			category_id INTEGER,
			status TEXT DEFAULT 'active',
			remark TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now', '+8 hours')),
			updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS canteen_expense_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			sort_order INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now', '+8 hours')),
			updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS canteen_purchases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_no TEXT NOT NULL UNIQUE,
			purchase_date TEXT NOT NULL,
			total_amount REAL NOT NULL DEFAULT 0,
			supplier_id INTEGER,
			supplier_name TEXT DEFAULT '',
			channel TEXT DEFAULT '',
			actual_pay REAL DEFAULT 0,
			remark TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now', '+8 hours')),
			updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
		)`,
		`CREATE TABLE IF NOT EXISTS canteen_purchase_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			purchase_id INTEGER NOT NULL,
			supply_id INTEGER NOT NULL,
			quantity REAL NOT NULL DEFAULT 0,
			unit_price REAL NOT NULL DEFAULT 0,
			subtotal REAL NOT NULL DEFAULT 0,
			remark TEXT DEFAULT ''
		)`,
	}
	for _, ddl := range ddls {
		require.NoError(t, db.Exec(ddl).Error, "DDL 执行失败: %s", ddl)
	}
	return db
}

// ====== 测试 A: 字典迁移 ======

func TestMigrateDictionariesOnly(t *testing.T) {
	srcDB := setupSourceDB(t)
	// 插入 5 条分类到源库
	inserts := []string{
		"INSERT INTO categories (id, name, sort_order) VALUES (1, '办公文具', 1)",
		"INSERT INTO categories (id, name, sort_order) VALUES (2, '劳保用品', 2)",
		"INSERT INTO categories (id, name, sort_order) VALUES (3, '清洁用品', 3)",
		"INSERT INTO categories (id, name, sort_order) VALUES (4, '耗材', 4)",
		"INSERT INTO categories (id, name, sort_order) VALUES (5, '其他', 5)",
	}
	for _, ins := range inserts {
		require.NoError(t, srcDB.Exec(ins).Error)
	}

	// 保存到临时文件让 Run 能通过 DSN 打开
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.db")
	sqlDB, _ := srcDB.DB()
	sqlDB.Close()
	// 重新创建到文件
	fileDB, err := gorm.Open(sqlite.Open(srcPath), &gorm.Config{})
	require.NoError(t, err)

	// 建表
	_ = fileDB.Exec(`CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		sort_order INTEGER DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now', '+8 hours')),
		updated_at TEXT DEFAULT (datetime('now', '+8 hours'))
	)`)
	for _, ins := range inserts {
		require.NoError(t, fileDB.Exec(ins).Error)
	}

	targetDB := setupTargetDB(t)

	// 执行仅字典迁移
	err = Run(targetDB, srcPath, Options{OnlyDictionaries: true})
	require.NoError(t, err)

	// 验证目标库有 5 条分类
	var count int64
	targetDB.Model(&models.OfficeCategory{}).Count(&count)
	assert.Equal(t, int64(5), count, "应迁入 5 条办公用品分类")
}

// ====== 测试 B: ID 重映射 ======

func TestIDRemapping(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.db")
	fileDB, err := gorm.Open(sqlite.Open(srcPath), &gorm.Config{})
	require.NoError(t, err)

	// 建表（suppliers, supplies, purchases, purchase_items）
	ddls := []string{
		`CREATE TABLE IF NOT EXISTS suppliers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			contact TEXT DEFAULT '',
			phone TEXT DEFAULT '',
			bank_name TEXT DEFAULT '',
			bank_account TEXT DEFAULT '',
			is_default INTEGER DEFAULT 0,
			remark TEXT DEFAULT '',
			created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS supplies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL, spec TEXT DEFAULT '', unit TEXT DEFAULT '个',
			reference_price REAL DEFAULT 0, safety_stock INTEGER DEFAULT 0,
			category_id INTEGER, supplier_id INTEGER,
			status TEXT DEFAULT 'active', remark TEXT DEFAULT '',
			created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS purchases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_no TEXT NOT NULL UNIQUE,
			purchase_date TEXT NOT NULL,
			total_amount REAL NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			remark TEXT DEFAULT '',
			supplier_id INTEGER,
			supplier_name TEXT DEFAULT '',
			payment_status TEXT DEFAULT '未付款',
			payment_date TEXT DEFAULT '',
			created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS purchase_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			purchase_id INTEGER NOT NULL,
			supply_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			unit_price REAL NOT NULL,
			subtotal REAL NOT NULL,
			date TEXT
		)`,
		// tables needed so auto-migrate works (empty)
		`CREATE TABLE IF NOT EXISTS categories (id INTEGER PRIMARY KEY, name TEXT, sort_order INTEGER, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS canteen_categories (id INTEGER PRIMARY KEY, name TEXT, sort_order INTEGER, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS canteen_supplies (id INTEGER PRIMARY KEY, name TEXT, spec TEXT, unit TEXT, reference_price REAL, category_id INTEGER, status TEXT, remark TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS canteen_expense_categories (id INTEGER PRIMARY KEY, name TEXT, sort_order INTEGER, created_at TEXT, updated_at TEXT)`,
	}
	for _, ddl := range ddls {
		require.NoError(t, fileDB.Exec(ddl).Error)
	}

	// 插入供应商 id=1
	fileDB.Exec("INSERT INTO suppliers (id, name) VALUES (1, '测试供应商A')")
	// 插入用品 id=1，关联 supplier_id=1
	fileDB.Exec("INSERT INTO supplies (id, name, supplier_id) VALUES (1, '测试用品A', 1)")
	// 插入采购单 id=1，关联 supplier_id=1
	fileDB.Exec("INSERT INTO purchases (id, order_no, purchase_date, total_amount, supplier_id) VALUES (1, 'PO-001', '2024-06-01 10:00:00', 100, 1)")
	// 插入采购明细引用 purchase_id=1, supply_id=1
	fileDB.Exec("INSERT INTO purchase_items (id, purchase_id, supply_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 10, 10, 100)")

	targetDB := setupTargetDB(t)

	err = Run(targetDB, srcPath, Options{})
	require.NoError(t, err)

	// 验证目标库中的 purchase_items 引用了正确的 supply_id（不是旧的 1）
	var item models.OfficePurchaseItem
	err = targetDB.First(&item).Error
	require.NoError(t, err)
	assert.NotZero(t, item.SupplyID, "SupplyID 不应为零")
	assert.NotZero(t, item.PurchaseID, "PurchaseID 不应为零")

	// 验证 supply 确实存在
	var supply models.OfficeSupply
	err = targetDB.First(&supply).Error
	require.NoError(t, err)
	assert.NotZero(t, supply.ID)
	assert.Equal(t, item.SupplyID, supply.ID, "采购明细的 supply_id 应与新插入的用品 ID 一致")
}

// ====== 测试 C: DryRun ======

func TestDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.db")
	fileDB, err := gorm.Open(sqlite.Open(srcPath), &gorm.Config{})
	require.NoError(t, err)

	// 只建 categories 表
	_ = fileDB.Exec(`CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE, sort_order INTEGER DEFAULT 0,
		created_at TEXT, updated_at TEXT
	)`)
	fileDB.Exec("INSERT INTO categories (id, name, sort_order) VALUES (1, '测试分类1', 1)")
	fileDB.Exec("INSERT INTO categories (id, name, sort_order) VALUES (2, '测试分类2', 2)")

	targetDB := setupTargetDB(t)

	// DryRun 模式
	err = Run(targetDB, srcPath, Options{OnlyDictionaries: true, DryRun: true})
	require.NoError(t, err)

	// 验证目标库为空（DryRun 不应该写入任何数据）
	var count int64
	targetDB.Model(&models.OfficeCategory{}).Count(&count)
	assert.Equal(t, int64(0), count, "DryRun 模式下目标库应无数据写入")
}
