package models

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	activeInvoiceAttachmentIndex = "idx_invoices_attachment_file_active"
	activeInvoiceIdentityIndex   = "idx_invoices_identity_key_active"
	invoiceForeignKeyVersion     = "invoice_foreign_keys_v1"
)

// MigrateInvoiceSchema 补齐发票的历史索引迁移。调用方必须在 AutoMigrate 后调用本函数。
func MigrateInvoiceSchema(db *gorm.DB) error {
	if err := migrateInvoiceTaskStatuses(db); err != nil {
		return err
	}
	if err := migrateInvoiceForeignKeysOnce(db); err != nil {
		return err
	}
	if err := removeLegacyInvoiceNoUniqueIndexes(db); err != nil {
		return err
	}
	if err := ensureInvoiceNoIndex(db); err != nil {
		return err
	}
	if err := ensureFileSHA256Index(db); err != nil {
		return err
	}
	if err := ensureActiveIdentityKeyIndex(db); err != nil {
		return err
	}
	return ensureActiveAttachmentFileIndex(db)
}

func migrateInvoiceTaskStatuses(db *gorm.DB) error {
	updates := map[string]string{
		"processing": string(InvoiceParsingTaskRunning),
		"completed":  string(InvoiceParsingTaskSucceeded),
	}
	for oldStatus, newStatus := range updates {
		if err := db.Model(&InvoiceParsingTask{}).Where("status = ?", oldStatus).Update("status", newStatus).Error; err != nil {
			return fmt.Errorf("迁移解析任务状态 %q 失败: %w", oldStatus, err)
		}
	}
	return nil
}

func migrateInvoiceForeignKeysOnce(db *gorm.DB) error {
	if err := db.Exec("CREATE TABLE IF NOT EXISTS invoice_schema_migrations (version VARCHAR(100) PRIMARY KEY)").Error; err != nil {
		return fmt.Errorf("创建发票迁移版本表失败: %w", err)
	}
	var count int64
	if err := db.Table("invoice_schema_migrations").Where("version = ?", invoiceForeignKeyVersion).Count(&count).Error; err != nil {
		return fmt.Errorf("读取发票迁移版本失败: %w", err)
	}
	if count > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, relation := range invoiceForeignKeyRelations() {
			if tx.Migrator().HasConstraint(relation.model, relation.relation) {
				continue
			}
			if err := tx.Migrator().CreateConstraint(relation.model, relation.relation); err != nil {
				return fmt.Errorf("创建发票外键 %q 失败: %w", relation.relation, err)
			}
		}
		if err := tx.Exec("INSERT INTO invoice_schema_migrations (version) VALUES (?)", invoiceForeignKeyVersion).Error; err != nil {
			return fmt.Errorf("记录发票迁移版本失败: %w", err)
		}
		return nil
	})
}

func invoiceForeignKeyRelations() []struct {
	model    interface{}
	relation string
} {
	return []struct {
		model    interface{}
		relation string
	}{
		{&Invoice{}, "AttachmentFile"},
		{&Invoice{}, "User"},
		{&Invoice{}, "Applicant"},
		{&Invoice{}, "Approver"},
		{&Invoice{}, "ConfirmedByUser"},
		{&Invoice{}, "VoidedByUser"},
		{&Invoice{}, "DeletedByUser"},
		{&InvoiceCorrectionAudit{}, "Invoice"},
		{&InvoiceCorrectionAudit{}, "Corrector"},
	}
}

func removeLegacyInvoiceNoUniqueIndexes(db *gorm.DB) error {
	indexes, err := db.Migrator().GetIndexes(&Invoice{})
	if err != nil {
		return fmt.Errorf("读取发票索引失败: %w", err)
	}
	for _, index := range indexes {
		unique, supported := index.Unique()
		if !supported || !unique || !isInvoiceNoOnlyIndex(index.Columns()) {
			continue
		}
		if err := db.Migrator().DropIndex(&Invoice{}, index.Name()); err != nil {
			return fmt.Errorf("删除旧发票号码唯一索引 %q 失败: %w", index.Name(), err)
		}
	}
	return nil
}

func isInvoiceNoOnlyIndex(columns []string) bool {
	return len(columns) == 1 && strings.EqualFold(columns[0], "invoice_no")
}

func ensureInvoiceNoIndex(db *gorm.DB) error {
	if db.Migrator().HasIndex(&Invoice{}, "InvoiceNo") {
		return nil
	}
	if err := db.Migrator().CreateIndex(&Invoice{}, "InvoiceNo"); err != nil {
		return fmt.Errorf("创建发票号码普通索引失败: %w", err)
	}
	return nil
}

func ensureFileSHA256Index(db *gorm.DB) error {
	if db.Migrator().HasIndex(&Invoice{}, "FileSHA256") {
		return nil
	}
	if err := db.Migrator().CreateIndex(&Invoice{}, "FileSHA256"); err != nil {
		return fmt.Errorf("创建文件哈希普通索引失败: %w", err)
	}
	return nil
}

func ensureActiveIdentityKeyIndex(db *gorm.DB) error {
	if db.Migrator().HasIndex(&Invoice{}, activeInvoiceIdentityIndex) {
		return nil
	}
	statement := "CREATE UNIQUE INDEX " + db.Statement.Quote(activeInvoiceIdentityIndex) +
		" ON " + db.Statement.Quote(Invoice{}.TableName()) +
		" (identity_key) WHERE identity_key IS NOT NULL AND deleted_at IS NULL"
	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf("创建活动发票身份键唯一索引失败: %w", err)
	}
	return nil
}

func ensureActiveAttachmentFileIndex(db *gorm.DB) error {
	if db.Migrator().HasIndex(&Invoice{}, activeInvoiceAttachmentIndex) {
		return nil
	}
	statement := "CREATE UNIQUE INDEX " + db.Statement.Quote(activeInvoiceAttachmentIndex) +
		" ON " + db.Statement.Quote(Invoice{}.TableName()) +
		" (attachment_file_id) WHERE attachment_file_id IS NOT NULL AND deleted_at IS NULL"
	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf("创建活动发票附件唯一索引失败: %w", err)
	}
	return nil
}
