package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupTagTestDB 创建 SQLite 内存数据库并迁移标签相关表
func setupTagTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	if err := db.AutoMigrate(
		&models.ArchiveTag{},
		&models.Document{},
		&models.DocumentTagLink{},
	); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	return db
}

func TestListTags_UserTags(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	userID := uint(1)
	otherUserID := uint(2)

	if _, err := svc.CreateTag(userID, "user-tag", "#ff0000"); err != nil {
		t.Fatalf("创建用户标签失败: %v", err)
	}
	if _, err := svc.CreateTag(otherUserID, "other-tag", "#00ff00"); err != nil {
		t.Fatalf("创建其他用户标签失败: %v", err)
	}

	tags, err := svc.ListTags(userID)
	if err != nil {
		t.Fatalf("ListTags 失败: %v", err)
	}
	if len(tags) < 1 {
		t.Errorf("tags count = %d, want >= 1", len(tags))
	}

	for _, tag := range tags {
		if tag.Name == "other-tag" {
			t.Error("不应返回其他用户的标签")
		}
	}
}

func TestCreateTag_Success(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	tag, err := svc.CreateTag(1, "test-tag", "#333333")
	if err != nil {
		t.Fatalf("CreateTag 失败: %v", err)
	}
	if tag.ID == 0 {
		t.Error("tag ID 不应为 0")
	}
	if tag.Name != "test-tag" {
		t.Errorf("name = %q, want %q", tag.Name, "test-tag")
	}
	if tag.Color != "#333333" {
		t.Errorf("color = %q, want %q", tag.Color, "#333333")
	}
}

func TestCreateTag_EmptyName(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	_, err := svc.CreateTag(1, "   ", "#333333")
	if err == nil {
		t.Error("期望空标签名返回错误")
	}
}

func TestCreateTag_DuplicateName(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	if _, err := svc.CreateTag(1, "dup-tag", "#111"); err != nil {
		t.Fatalf("首次创建标签失败: %v", err)
	}
	_, err := svc.CreateTag(1, "dup-tag", "#222")
	if err == nil {
		t.Error("期望重复标签名返回错误")
	}
}

func TestCreateTag_GlobalTag(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	tag, err := svc.CreateTag(0, "global-tag", "#444")
	if err != nil {
		t.Fatalf("创建全局标签失败: %v", err)
	}
	if tag.UserID != nil {
		t.Errorf("全局标签 user_id 应为 nil，实际为 %v", *tag.UserID)
	}
}

func TestDeleteTag_Success(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	tag, err := svc.CreateTag(1, "to-delete", "#999")
	if err != nil {
		t.Fatalf("创建待删除标签失败: %v", err)
	}

	if err := svc.DeleteTag(tag.ID); err != nil {
		t.Fatalf("DeleteTag 失败: %v", err)
	}

	var count int64
	if err := db.Model(&models.ArchiveTag{}).Where("id = ?", tag.ID).Count(&count).Error; err != nil {
		t.Fatalf("统计标签失败: %v", err)
	}
	if count != 0 {
		t.Errorf("标签未被删除，剩余计数: %d", count)
	}
}

func TestDeleteTag_ZeroID(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	err := svc.DeleteTag(0)
	if err == nil {
		t.Error("期望标签 ID 为空时返回错误")
	}
}

func TestDeleteTag_CascadesLinks(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	doc := models.Document{UserID: 1, FileName: "test"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	tag, err := svc.CreateTag(1, "cascade-tag", "#abc")
	if err != nil {
		t.Fatalf("创建标签失败: %v", err)
	}
	// 通过标签名建立关联（SetDocumentTags 使用全局标签，名称相同即可）
	if err := svc.SetDocumentTags(doc.ID, []string{tag.Name}); err != nil {
		t.Fatalf("设置文档标签失败: %v", err)
	}

	if err := svc.DeleteTag(tag.ID); err != nil {
		t.Fatalf("DeleteTag 失败: %v", err)
	}

	var linkCount int64
	if err := db.Model(&models.DocumentTagLink{}).Where("tag_id = ?", tag.ID).Count(&linkCount).Error; err != nil {
		t.Fatalf("统计关联失败: %v", err)
	}
	if linkCount != 0 {
		t.Errorf("标签关联未被级联删除，剩余计数: %d", linkCount)
	}
}

func TestSetDocumentTags_Success(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	doc := models.Document{UserID: 1, FileName: "test"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	err := svc.SetDocumentTags(doc.ID, []string{"tag1", "tag2"})
	if err != nil {
		t.Fatalf("SetDocumentTags 失败: %v", err)
	}

	tags, err := svc.GetDocumentTags(doc.ID)
	if err != nil {
		t.Fatalf("GetDocumentTags 失败: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("tags count = %d, want 2", len(tags))
	}
}

func TestSetDocumentTags_ReplaceMode(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	doc := models.Document{UserID: 1, FileName: "test"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	if err := svc.SetDocumentTags(doc.ID, []string{"old-tag"}); err != nil {
		t.Fatalf("首次设置标签失败: %v", err)
	}
	if err := svc.SetDocumentTags(doc.ID, []string{"new-tag"}); err != nil {
		t.Fatalf("替换标签失败: %v", err)
	}

	tags, err := svc.GetDocumentTags(doc.ID)
	if err != nil {
		t.Fatalf("GetDocumentTags 失败: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("tags count = %d, want 1", len(tags))
	}
	if tags[0].Name != "new-tag" {
		t.Errorf("tag name = %q, want %q", tags[0].Name, "new-tag")
	}
}

func TestSetDocumentTags_CreatesGlobalTags(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	doc := models.Document{UserID: 1, FileName: "test"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	if err := svc.SetDocumentTags(doc.ID, []string{"auto-global"}); err != nil {
		t.Fatalf("SetDocumentTags 失败: %v", err)
	}

	var tag models.ArchiveTag
	if err := db.Where("name = ? AND user_id IS NULL", "auto-global").First(&tag).Error; err != nil {
		t.Fatalf("未找到自动创建的全局标签: %v", err)
	}
}

func TestSetDocumentTags_EmptyDocumentID(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	err := svc.SetDocumentTags(0, []string{"tag"})
	if err == nil {
		t.Error("期望文档 ID 为空时返回错误")
	}
}

func TestGetDocumentTags_NoTags(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	doc := models.Document{UserID: 1, FileName: "test"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	tags, err := svc.GetDocumentTags(doc.ID)
	if err != nil {
		t.Fatalf("GetDocumentTags 失败: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("tags count = %d, want 0", len(tags))
	}
}

func TestGetDocumentTags_EmptyDocumentID(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	_, err := svc.GetDocumentTags(0)
	if err == nil {
		t.Error("期望文档 ID 为空时返回错误")
	}
}

func TestTagWithCount(t *testing.T) {
	db := setupTagTestDB(t)
	svc := NewTagService(db)

	doc := models.Document{UserID: 1, FileName: "test"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	tagName := "counted-tag"
	if err := svc.SetDocumentTags(doc.ID, []string{tagName}); err != nil {
		t.Fatalf("设置文档标签失败: %v", err)
	}

	tags, err := svc.ListTags(1)
	if err != nil {
		t.Fatalf("ListTags 失败: %v", err)
	}

	found := false
	for _, tag := range tags {
		if tag.Name == tagName {
			found = true
			if tag.DocumentCount != 1 {
				t.Errorf("tag %q document_count = %d, want 1", tagName, tag.DocumentCount)
			}
		}
	}
	if !found {
		t.Errorf("未在 ListTags 结果中找到标签 %q", tagName)
	}
}
