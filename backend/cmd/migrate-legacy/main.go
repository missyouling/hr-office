// CLI 入口：office-supply-analytics SQLite → hr-office 数据迁移
//
// 用法:
//
//	migrate-legacy --source ./data/source.db --target file:./data/target.db
//	migrate-legacy --source ./data/source.db --only-dictionaries
//	migrate-legacy --source ./data/source.db --dry-run
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/migrate"
)

func main() {
	sourcePath := flag.String("source", "", "源 SQLite 数据库文件路径（必填）")
	onlyDicts := flag.Bool("only-dictionaries", false, "仅迁移字典数据")
	dryRun := flag.Bool("dry-run", false, "仅打印不写入")
	targetDSN := flag.String("target", "", "目标数据库 DSN（默认读环境变量）")
	flag.Parse()

	if *sourcePath == "" {
		fmt.Fprintln(os.Stderr, "错误: --source 参数为必填项")
		flag.Usage()
		os.Exit(1)
	}

	// 目标 DSN：优先 --target，其次 SIAPP_DATABASE_PATH，再其次 DATABASE_URL
	dsn := *targetDSN
	if dsn == "" {
		dsn = os.Getenv("SIAPP_DATABASE_PATH")
	}
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "错误: 未指定目标数据库，请使用 --target 或设置 SIAPP_DATABASE_PATH / DATABASE_URL")
		os.Exit(1)
	}

	// 打开目标库
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("打开目标数据库失败: %v", err)
	}

	// 自动建表（确保目标表存在）
	if err := db.AutoMigrate(
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
	); err != nil {
		log.Fatalf("目标库建表失败: %v", err)
	}

	opts := migrate.Options{
		OnlyDictionaries: *onlyDicts,
		DryRun:           *dryRun,
	}

	fmt.Printf("开始迁移: 源=%s 目标=%s 仅字典=%v 试运行=%v\n",
		*sourcePath, dsn, *onlyDicts, *dryRun)

	if err := migrate.Run(db, *sourcePath, opts); err != nil {
		log.Fatalf("迁移失败: %v", err)
	}

	fmt.Println("迁移完成")
}
