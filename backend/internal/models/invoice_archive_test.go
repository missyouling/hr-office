package models

import (
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupInvoiceArchiveTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("启用 SQLite 外键失败: %v", err)
	}
	var foreignKeysEnabled int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&foreignKeysEnabled).Error; err != nil || foreignKeysEnabled != 1 {
		t.Fatalf("SQLite 外键未启用: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &SysFile{}, &Invoice{}, &InvoiceItem{}, &InvoiceParsingTask{}, &InvoiceCorrectionAudit{}); err != nil {
		t.Fatalf("迁移发票归档模型失败: %v", err)
	}
	if err := MigrateInvoiceSchema(db); err != nil {
		t.Fatalf("执行发票索引迁移失败: %v", err)
	}
	return db
}

func createArchiveTestInvoice(t *testing.T, db *gorm.DB, identityKey *string) Invoice {
	t.Helper()
	invoice := Invoice{
		InvoiceDate: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC),
		Amount:      100,
		Seller:      "测试销售方",
		IdentityKey: identityKey,
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatalf("创建测试发票失败: %v", err)
	}
	return invoice
}

func TestInvoiceArchiveStatusIsIndependentFromReimbursementStatus(t *testing.T) {
	db := setupInvoiceArchiveTestDB(t)
	invoice := createArchiveTestInvoice(t, db, nil)
	if invoice.ArchiveStatus != InvoiceArchiveStatusPending || invoice.Status != InvoiceStatusDraft {
		t.Fatalf("默认状态不正确：归档=%q，报销=%q", invoice.ArchiveStatus, invoice.Status)
	}
	if err := db.Model(&invoice).Updates(map[string]interface{}{"archive_status": InvoiceArchiveStatusConfirmed, "status": InvoiceStatusApproved}).Error; err != nil {
		t.Fatalf("更新正交状态失败: %v", err)
	}
	var saved Invoice
	if err := db.First(&saved, invoice.ID).Error; err != nil {
		t.Fatalf("查询测试发票失败: %v", err)
	}
	if saved.ArchiveStatus != InvoiceArchiveStatusConfirmed || saved.Status != InvoiceStatusApproved {
		t.Errorf("归档和报销状态未独立保存：归档=%q，报销=%q", saved.ArchiveStatus, saved.Status)
	}
}

func TestInvoiceNumberAllowsEmptyAndDuplicateValues(t *testing.T) {
	db := setupInvoiceArchiveTestDB(t)
	first := createArchiveTestInvoice(t, db, nil)
	second := createArchiveTestInvoice(t, db, nil)
	first.InvoiceNo, second.InvoiceNo = "重复票号", "重复票号"
	if err := db.Save(&first).Error; err != nil {
		t.Fatalf("保存首个重复票号失败: %v", err)
	}
	if err := db.Save(&second).Error; err != nil {
		t.Fatalf("保存第二个重复票号失败: %v", err)
	}
}

func TestInvoiceIdentityKeyBlocksActiveDuplicateAndAllowsSoftDeletedRecord(t *testing.T) {
	db := setupInvoiceArchiveTestDB(t)
	key := "vat:code:number"
	first := createArchiveTestInvoice(t, db, &key)
	if err := db.Create(&Invoice{InvoiceDate: first.InvoiceDate, Amount: 1, Seller: "测试", IdentityKey: &key}).Error; err == nil {
		t.Fatal("活动发票的重复身份键应被拒绝")
	}
	if err := db.Delete(&first).Error; err != nil {
		t.Fatalf("软删除发票失败: %v", err)
	}
	if err := db.Create(&Invoice{InvoiceDate: first.InvoiceDate, Amount: 1, Seller: "测试", IdentityKey: &key}).Error; err != nil {
		t.Fatalf("软删除后应允许相同身份键: %v", err)
	}
}

func TestInvoiceRelationsAndConstraints(t *testing.T) {
	db := setupInvoiceArchiveTestDB(t)
	file := SysFile{Path: "invoices/test.pdf"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}
	invoice := createArchiveTestInvoice(t, db, nil)
	invoice.AttachmentFileID = &file.ID
	invoice.FileSHA256 = "same-hash"
	if err := db.Save(&invoice).Error; err != nil {
		t.Fatalf("关联受控附件失败: %v", err)
	}
	if err := db.Create(&Invoice{InvoiceDate: invoice.InvoiceDate, Amount: 1, Seller: "重复附件", AttachmentFileID: &file.ID}).Error; err == nil {
		t.Error("同一活动附件不能绑定到两张活动发票")
	}
	if err := db.Delete(&invoice).Error; err != nil {
		t.Fatalf("软删除附件发票失败: %v", err)
	}
	reused := Invoice{InvoiceDate: invoice.InvoiceDate, Amount: 1, Seller: "复用附件", AttachmentFileID: &file.ID}
	if err := db.Create(&reused).Error; err != nil {
		t.Fatalf("软删除后应允许复用附件: %v", err)
	}
	if err := db.Unscoped().Delete(&reused).Error; err != nil {
		t.Fatalf("清理复用附件测试发票失败: %v", err)
	}
	if err := db.Unscoped().Delete(&invoice).Error; err != nil {
		t.Fatalf("清理附件测试发票失败: %v", err)
	}
	invoice = createArchiveTestInvoice(t, db, nil)
	invoice.AttachmentFileID = &file.ID
	if err := db.Save(&invoice).Error; err != nil {
		t.Fatalf("恢复附件关联失败: %v", err)
	}
	if err := db.Create(&Invoice{InvoiceDate: invoice.InvoiceDate, Amount: 1, Seller: "其他", FileSHA256: invoice.FileSHA256}).Error; err != nil {
		t.Fatalf("相同文件哈希仅作预警，不应被拒绝: %v", err)
	}
	if err := db.Unscoped().Delete(&file).Error; err == nil {
		t.Error("被发票引用的文件应受 RESTRICT 外键保护")
	}
	item := InvoiceItem{InvoiceID: invoice.ID, LineNo: 1, Name: "办公用品", Amount: 100, TaxAmount: 13}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("创建发票明细失败: %v", err)
	}
	if err := db.Create(&InvoiceItem{InvoiceID: invoice.ID, LineNo: item.LineNo, Name: "重复明细"}).Error; err == nil {
		t.Error("同一发票的重复明细行号应被拒绝")
	}
	if err := db.Create(&InvoiceParsingTask{InvoiceID: invoice.ID, Status: InvoiceParsingTaskRunning}).Error; err != nil {
		t.Fatalf("创建解析任务失败: %v", err)
	}
	if err := db.Create(&InvoiceParsingTask{InvoiceID: invoice.ID}).Error; err == nil {
		t.Error("同一发票的重复解析任务应被唯一约束拒绝")
	}
	corrector := User{Username: "corrector", Email: "corrector@example.test", Password: "test-password", FullName: "更正人"}
	if err := db.Create(&corrector).Error; err != nil {
		t.Fatalf("创建更正人失败: %v", err)
	}
	audit := InvoiceCorrectionAudit{InvoiceID: invoice.ID, CorrectedBy: corrector.ID, Changes: datatypes.JSON(`{"seller":{"old":"旧名称","new":"新名称"}}`), Reason: "人工核验"}
	if err := db.Create(&audit).Error; err != nil {
		t.Fatalf("创建批次更正审计失败: %v", err)
	}
	if err := db.Create(&InvoiceCorrectionAudit{InvoiceID: invoice.ID, Reason: "缺少责任人"}).Error; err == nil {
		t.Error("更正审计缺少责任人应被拒绝")
	}
	if err := db.Unscoped().Delete(&corrector).Error; err == nil {
		t.Error("被更正审计引用的用户应受 RESTRICT 外键保护")
	}
	if err := db.Unscoped().Delete(&invoice).Error; err == nil {
		t.Error("存在更正审计的发票不应被级联删除")
	}
}

func TestInvoiceArchiveActorReferencesUseSetNull(t *testing.T) {
	db := setupInvoiceArchiveTestDB(t)
	actor := User{Username: "archiver", Email: "archiver@example.test", Password: "test-password"}
	if err := db.Create(&actor).Error; err != nil {
		t.Fatalf("创建归档责任人失败: %v", err)
	}
	now := time.Now()
	invoice := createArchiveTestInvoice(t, db, nil)
	invoice.ConfirmedBy, invoice.ConfirmedAt = &actor.ID, &now
	invoice.VoidedBy, invoice.VoidedAt, invoice.VoidedReason = &actor.ID, &now, "作废原因"
	invoice.DeletedBy = &actor.ID
	if err := db.Save(&invoice).Error; err != nil {
		t.Fatalf("保存归档责任人失败: %v", err)
	}
	if err := db.Delete(&actor).Error; err != nil {
		t.Fatalf("删除历史用户失败: %v", err)
	}
	var saved Invoice
	if err := db.First(&saved, invoice.ID).Error; err != nil {
		t.Fatalf("查询发票失败: %v", err)
	}
	if saved.ConfirmedBy != nil || saved.VoidedBy != nil || saved.DeletedBy != nil {
		t.Error("删除历史用户后归档责任人引用应置空")
	}
}

func TestInvoiceArchiveIndexes(t *testing.T) {
	db := setupInvoiceArchiveTestDB(t)
	if !db.Migrator().HasIndex(&Invoice{}, "InvoiceNo") {
		t.Error("缺少发票号码普通索引")
	}
	if !db.Migrator().HasIndex(&Invoice{}, activeInvoiceIdentityIndex) {
		t.Error("缺少活动身份键唯一索引")
	}
	if !db.Migrator().HasIndex(&Invoice{}, activeInvoiceAttachmentIndex) {
		t.Error("缺少活动附件唯一索引")
	}
	if !hasNonUniqueSingleColumnIndex(t, db, &Invoice{}, "file_sha256") {
		t.Error("缺少文件哈希普通索引")
	}
	if !db.Migrator().HasIndex(&InvoiceItem{}, "idx_invoice_items_invoice_line") {
		t.Error("缺少发票明细行号唯一索引")
	}
}

func TestInvoiceSchemaMigratesLegacySQLiteTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开旧结构测试数据库失败: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("启用 SQLite 外键失败: %v", err)
	}
	legacyTable := `CREATE TABLE invoices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		invoice_no TEXT NOT NULL UNIQUE,
		invoice_date DATETIME NOT NULL,
		amount REAL NOT NULL,
		seller TEXT NOT NULL,
		status TEXT DEFAULT 'draft',
		created_at DATETIME,
		updated_at DATETIME
	)`
	if err := db.Exec(legacyTable).Error; err != nil {
		t.Fatalf("创建旧 invoices 表失败: %v", err)
	}
	if err := db.Exec("INSERT INTO invoices (invoice_no, invoice_date, amount, seller) VALUES (?, ?, ?, ?)", "OLD-001", time.Now(), 100, "旧销售方").Error; err != nil {
		t.Fatalf("插入旧发票数据失败: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &SysFile{}, &Invoice{}, &InvoiceItem{}, &InvoiceParsingTask{}, &InvoiceCorrectionAudit{}); err != nil {
		t.Fatalf("升级旧发票表失败: %v", err)
	}
	if err := MigrateInvoiceSchema(db); err != nil {
		t.Fatalf("执行旧结构发票迁移失败: %v", err)
	}
	if err := MigrateInvoiceSchema(db); err != nil {
		t.Fatalf("重复执行发票迁移失败: %v", err)
	}
	var legacy Invoice
	if err := db.Where("invoice_no = ?", "OLD-001").First(&legacy).Error; err != nil {
		t.Fatalf("升级后旧发票数据丢失: %v", err)
	}
	if err := db.Create(&Invoice{InvoiceNo: "OLD-001", InvoiceDate: time.Now(), Amount: 1, Seller: "重复票号"}).Error; err != nil {
		t.Fatalf("旧唯一约束应被移除，重复票号创建失败: %v", err)
	}
	if !db.Migrator().HasIndex(&Invoice{}, "InvoiceNo") || !db.Migrator().HasIndex(&Invoice{}, activeInvoiceIdentityIndex) {
		t.Error("旧结构升级后缺少票号普通索引或身份键部分唯一索引")
	}
	if hasUniqueSingleColumnIndex(t, db, &Invoice{}, "invoice_no") {
		t.Error("旧发票号码唯一索引未移除")
	}
}

func hasNonUniqueSingleColumnIndex(t *testing.T, db *gorm.DB, model interface{}, column string) bool {
	t.Helper()
	indexes, err := db.Migrator().GetIndexes(model)
	if err != nil {
		t.Fatalf("读取索引失败: %v", err)
	}
	for _, index := range indexes {
		unique, supported := index.Unique()
		if len(index.Columns()) == 1 && index.Columns()[0] == column && (!supported || !unique) {
			return true
		}
	}
	return false
}

func hasUniqueSingleColumnIndex(t *testing.T, db *gorm.DB, model interface{}, column string) bool {
	t.Helper()
	indexes, err := db.Migrator().GetIndexes(model)
	if err != nil {
		t.Fatalf("读取索引失败: %v", err)
	}
	for _, index := range indexes {
		unique, supported := index.Unique()
		if len(index.Columns()) == 1 && index.Columns()[0] == column && supported && unique {
			return true
		}
	}
	return false
}
