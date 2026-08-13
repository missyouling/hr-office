package api

import (
	"bytes"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/middleware"
	"siapp/internal/models"
)

// ======== 测试辅助函数 ========

// newInvoiceTestRouterNoAuth 创建无权限中间件的发票路由（用于 CRUD 功能测试）
func newInvoiceTestRouterNoAuth(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/invoices", func(ir chi.Router) {
		ir.Get("/", handler.listInvoices)
		ir.Post("/", handler.createInvoice)
		ir.Get("/stats", handler.invoiceStats)
		ir.Route("/{id}", func(sr chi.Router) {
			sr.Get("/", handler.getInvoice)
			sr.Put("/", handler.updateInvoice)
			sr.Delete("/", handler.deleteInvoice)
			sr.Post("/submit", handler.submitInvoice)
			sr.Post("/approve", handler.approveInvoice)
			sr.Post("/reject", handler.rejectInvoice)
			sr.Post("/reimburse", handler.reimburseInvoice)
		})
	})
	return r
}

// newInvoiceTestRouter 创建带权限中间件的发票路由（用于权限测试）
func newInvoiceTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/invoices", func(ir chi.Router) {
		ir.Post("/", handler.createInvoice)
		ir.Group(func(mgr chi.Router) {
			mgr.Use(middleware.RequireManagerOrAbove(handler.db))
			mgr.Get("/", handler.listInvoices)
			mgr.Get("/stats", handler.invoiceStats)
			mgr.Get("/export", handler.exportInvoicesCSV)
		})
		ir.Group(func(adm chi.Router) {
			adm.Use(middleware.RequireAdmin(handler.db))
			adm.Get("/buyer-entity", handler.getBuyerEntitySetting)
			adm.Put("/buyer-entity", handler.updateBuyerEntitySetting)
		})
		ir.Route("/{id}", func(sr chi.Router) {
			sr.Get("/attachment", handler.previewInvoiceAttachment)
			sr.Get("/", handler.getInvoice)
			sr.Put("/", handler.updateInvoice)
			sr.Delete("/", handler.deleteInvoice)
			sr.Post("/submit", handler.submitInvoice)
			sr.Group(func(adm chi.Router) {
				adm.Use(middleware.RequireAdmin(handler.db))
				adm.Post("/approve", handler.approveInvoice)
				adm.Post("/reject", handler.rejectInvoice)
				adm.Post("/confirm", handler.confirmInvoice)
				adm.Post("/void", handler.voidInvoice)
				adm.Post("/correct", handler.correctInvoice)
			})
			sr.With(middleware.RequireManagerOrAbove(handler.db)).
				Post("/reimburse", handler.reimburseInvoice)
		})
	})
	return r
}

// createInvoiceTestUser 创建测试用户
func createInvoiceTestUser(t *testing.T, tx *gorm.DB, username string, fullName string) models.User {
	t.Helper()
	user := models.User{
		Username: username,
		Email:    username + "@invoice-test.local",
		Password: "placeholder",
		FullName: fullName,
		Active:   true,
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

// createInvoiceTestRole 创建测试角色
func createInvoiceTestRole(t *testing.T, tx *gorm.DB, name string) models.Role {
	t.Helper()
	role := models.Role{Name: name, IsSystem: true}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatalf("创建测试角色失败: %v", err)
	}
	return role
}

// assignRole 给用户分配角色
func assignRole(t *testing.T, tx *gorm.DB, userID uint, roleID uint) {
	t.Helper()
	ur := models.UserRole{UserID: userID, RoleID: roleID}
	if err := tx.Create(&ur).Error; err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
}

// migrateInvoiceTables 迁移发票测试所需的表
func migrateInvoiceTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.StorageConfig{},
		&models.StorageModuleConfig{},
		&models.StorageRule{},
		&models.SysFile{},
		&models.Invoice{},
		&models.InvoiceItem{},
		&models.InvoiceParsingTask{},
		&models.InvoiceFileCleanupTask{},
		&models.InvoiceCorrectionAudit{},
		&models.AuditLog{},
		&models.BuyerEntitySetting{},
		&models.OfficePurchase{},
		&models.CanteenPurchase{},
		&models.OfficePaymentRequest{},
	); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// jsonReader 将字节转为 io.Reader（辅助函数）
func jsonReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
