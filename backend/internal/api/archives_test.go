package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"siapp/internal/models"
)

// setupTestDB 根据环境变量建立数据库连接
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbType := strings.ToLower(os.Getenv("SIAPP_DATABASE_TYPE"))
	if dbType == "" {
		dbType = "sqlite" // 默认使用 sqlite 用于测试
	}

	var db *gorm.DB
	var err error

	if dbType == "postgres" || dbType == "postgresql" {
		dbHost := os.Getenv("SIAPP_DB_HOST")
		dbPort := os.Getenv("SIAPP_DB_PORT")
		dbUser := os.Getenv("SIAPP_DB_USER")
		dbPassword := os.Getenv("SIAPP_DB_PASSWORD")
		dbName := os.Getenv("SIAPP_DB_NAME")
		sslMode := os.Getenv("SIAPP_DB_SSLMODE")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			t.Skip("PostgreSQL 环境变量不完整，跳过")
		}
		if sslMode == "" {
			sslMode = "require"
		}

		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			dbHost, dbUser, dbPassword, dbName, dbPort, sslMode)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("连接 PostgreSQL 失败: %v", err)
		}
	} else {
		// 使用内存 SQLite 数据库
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("创建 SQLite 测试数据库失败: %v", err)
		}

		// 启用外键约束（SQLite 需要显式启用）
		if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			t.Fatalf("启用 SQLite 外键约束失败: %v", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取数据库连接对象失败: %v", err)
	}

	// SQLite 内存库每个连接是独立数据库，固定单连接防止异步隔离
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

// newTestTransaction 返回一个会在测试结束时回滚的事务
func newTestTransaction(t *testing.T, db *gorm.DB) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("开启事务失败: %v", tx.Error)
	}

	t.Cleanup(func() {
		tx.Rollback()
	})

	return tx
}

// TestArchiveAPIs 测试档案配置 API 的级联删除和嵌套结构
func TestArchiveAPIs(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)

	// 自动迁移表结构
	err := tx.AutoMigrate(
		&models.DocumentCategory{},
		&models.DocumentSubCategory{},
		&models.ArchiveFieldDefinition{},
		&models.ArchiveFieldGroup{},
	)
	if err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}

	// 创建 Handler
	handler := NewHandler(tx)

	// 创建路由
	r := chi.NewRouter()
	r.Get("/api/archives/categories", handler.listDocumentCategories)
	r.Delete("/api/archives/categories/{categoryID}", handler.deleteDocumentCategory)

	// 1. 创建一级分类 DocumentCategory
	category := models.DocumentCategory{
		Code:        "TEST",
		Name:        "测试分类",
		Description: "用于测试级联删除",
		SortOrder:   1,
	}
	if err := tx.Create(&category).Error; err != nil {
		t.Fatalf("创建一级分类失败: %v", err)
	}
	t.Logf("创建一级分类成功, ID=%d, Code=%s", category.ID, category.Code)

	// 2. 创建二级分类 DocumentSubCategory
	subCategory := models.DocumentSubCategory{
		CategoryID:   category.ID,
		Code:         "01",
		Name:         "测试子分类",
		Description:  "用于测试级联删除",
		CategoryCode: category.Code,
		SortOrder:    1,
	}
	if err := tx.Create(&subCategory).Error; err != nil {
		t.Fatalf("创建二级分类失败: %v", err)
	}
	t.Logf("创建二级分类成功, ID=%d, Code=%s", subCategory.ID, subCategory.Code)

	// 3. 创建字段定义 ArchiveFieldDefinition
	fieldDef := models.ArchiveFieldDefinition{
		SubCategoryID: subCategory.ID,
		FieldName:     "test_field",
		FieldLabel:    "测试字段",
		FieldType:     "text",
		Required:      true,
		Visible:       true,
		Editable:      true,
		SortOrder:     1,
	}
	if err := tx.Create(&fieldDef).Error; err != nil {
		t.Fatalf("创建字段定义失败: %v", err)
	}
	t.Logf("创建字段定义成功, ID=%d, FieldName=%s", fieldDef.ID, fieldDef.FieldName)

	// 4. 调用 GET /api/archives/categories 验证嵌套结构
	req := httptest.NewRequest("GET", "/api/archives/categories", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/archives/categories 返回状态码 %d, 期望 200, 响应体: %s", w.Code, w.Body.String())
	}

	var categories []models.DocumentCategory
	if err := json.NewDecoder(w.Body).Decode(&categories); err != nil {
		t.Fatalf("解析返回的分类列表失败: %v, 响应体: %s", err, w.Body.String())
	}

	// 验证嵌套结构
	found := false
	for _, cat := range categories {
		if cat.ID == category.ID {
			found = true
			if len(cat.SubCategories) == 0 {
				t.Fatalf("一级分类 %d 的子分类列表为空，期望包含子分类", cat.ID)
			}
			if cat.SubCategories[0].ID != subCategory.ID {
				t.Fatalf("子分类 ID 不匹配: 得到 %d, 期望 %d", cat.SubCategories[0].ID, subCategory.ID)
			}
			t.Logf("✓ 嵌套结构验证通过: 一级分类包含 %d 个子分类", len(cat.SubCategories))
			break
		}
	}
	if !found {
		t.Fatalf("未在返回的分类列表中找到创建的一级分类 ID=%d", category.ID)
	}

	// 5. 调用 DELETE /api/archives/categories/{id}
	deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/archives/categories/%d", category.ID), nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusOK {
		t.Fatalf("DELETE /api/archives/categories/%d 返回状态码 %d, 期望 200, 响应体: %s",
			category.ID, deleteW.Code, deleteW.Body.String())
	}
	t.Logf("✓ 删除一级分类 API 调用成功")

	// 6. 验证级联删除：检查一级分类、二级分类、字段定义是否全部删除
	var deletedCategory models.DocumentCategory
	err = tx.Where("id = ?", category.ID).First(&deletedCategory).Error
	if err == nil {
		t.Fatalf("一级分类未被删除: ID=%d 仍然存在", category.ID)
	}
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("查询一级分类时出错: %v", err)
	}
	t.Logf("✓ 一级分类已删除")

	var deletedSubCategory models.DocumentSubCategory
	err = tx.Where("id = ?", subCategory.ID).First(&deletedSubCategory).Error
	if err == nil {
		t.Fatalf("二级分类未被级联删除: ID=%d 仍然存在", subCategory.ID)
	}
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("查询二级分类时出错: %v", err)
	}
	t.Logf("✓ 二级分类已级联删除")

	var deletedFieldDef models.ArchiveFieldDefinition
	err = tx.Where("id = ?", fieldDef.ID).First(&deletedFieldDef).Error
	if err == nil {
		t.Fatalf("字段定义未被级联删除: ID=%d 仍然存在", fieldDef.ID)
	}
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("查询字段定义时出错: %v", err)
	}
	t.Logf("✓ 字段定义已级联删除")

	t.Log("✓✓✓ 所有测试通过：级联删除和嵌套结构验证成功")
}

// deleteDocumentCategory 删除一级分类的处理器（需要在 archives.go 中实现）
func (h *Handler) deleteDocumentCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "categoryID")

	var category models.DocumentCategory
	if err := h.db.Where("id = ?", categoryID).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "分类不存在", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 直接删除，由数据库的 OnDelete:CASCADE 处理级联删除
	if err := h.db.Delete(&category).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "删除成功"})
}
