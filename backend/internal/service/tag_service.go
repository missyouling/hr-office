package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"siapp/internal/models"
)

// TagService 标签管理服务
// 提供档案标签的增删改查、文档标签绑定及旧数据迁移能力
type TagService struct {
	db *gorm.DB
}

// NewTagService 创建标签服务实例
func NewTagService(db *gorm.DB) *TagService {
	return &TagService{db: db}
}

// ListTags 列出指定用户可用标签（用户自建 + 全局标签），并附带每个标签关联的文档数量
func (s *TagService) ListTags(userID uint) ([]models.TagWithCount, error) {
	var tags []models.TagWithCount
	err := s.db.Model(&models.ArchiveTag{}).
		Select("archive_tags.*, COUNT(document_tag_links.id) AS document_count").
		Joins("LEFT JOIN document_tag_links ON document_tag_links.tag_id = archive_tags.id").
		Where("archive_tags.user_id = ? OR archive_tags.user_id IS NULL", userID).
		Group("archive_tags.id").
		Order("archive_tags.name ASC").
		Scan(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	return tags, nil
}

// CreateTag 创建标签；当 userID 为 0 时创建全局标签，否则创建该用户的私有标签
// 同一用户（或全局）下标签名不允许重复
func (s *TagService) CreateTag(userID uint, name string, color string) (*models.ArchiveTag, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, errors.New("标签名称不能为空")
	}

	// 颜色默认值
	if strings.TrimSpace(color) == "" {
		color = "#3b82f6"
	}

	var existing int64
	query := s.db.Model(&models.ArchiveTag{}).Where("name = ?", trimmedName)
	if userID == 0 {
		query = query.Where("user_id IS NULL")
	} else {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.Count(&existing).Error; err != nil {
		return nil, fmt.Errorf("检查标签唯一性失败: %w", err)
	}
	if existing > 0 {
		return nil, fmt.Errorf("标签名称 '%s' 已存在", trimmedName)
	}

	tag := &models.ArchiveTag{
		Name:  trimmedName,
		Color: color,
	}
	if userID > 0 {
		tag.UserID = &userID
	}

	if err := s.db.Create(tag).Error; err != nil {
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}
	return tag, nil
}

// DeleteTag 删除标签并级联删除其文档关联
func (s *TagService) DeleteTag(tagID uint) error {
	if tagID == 0 {
		return errors.New("标签 ID 不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 先删除关联记录，再删除标签本身，避免依赖数据库级联产生意外行为
		if err := tx.Where("tag_id = ?", tagID).Delete(&models.DocumentTagLink{}).Error; err != nil {
			return fmt.Errorf("删除标签关联失败: %w", err)
		}
		if err := tx.Delete(&models.ArchiveTag{}, tagID).Error; err != nil {
			return fmt.Errorf("删除标签失败: %w", err)
		}
		return nil
	})
}

// SetDocumentTags 设置文档标签（替换模式）
// 先清空该文档所有旧关联，再根据 tagNames 重新建立；不存在的标签自动创建为全局标签
func (s *TagService) SetDocumentTags(documentID uint, tagNames []string) error {
	if documentID == 0 {
		return errors.New("文档 ID 不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除文档现有标签关联
		if err := tx.Where("document_id = ?", documentID).Delete(&models.DocumentTagLink{}).Error; err != nil {
			return fmt.Errorf("清除文档旧标签关联失败: %w", err)
		}

		// 2. 去重并过滤空名称
		seen := make(map[string]struct{})
		uniqueNames := make([]string, 0, len(tagNames))
		for _, n := range tagNames {
			name := strings.TrimSpace(n)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			uniqueNames = append(uniqueNames, name)
		}

		// 3. 获取或创建全局标签，并建立关联
		for _, name := range uniqueNames {
			var tag models.ArchiveTag
			err := tx.Where("name = ? AND user_id IS NULL", name).First(&tag).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					tag = models.ArchiveTag{
						Name:  name,
						Color: "#3b82f6",
					}
					if err := tx.Create(&tag).Error; err != nil {
						return fmt.Errorf("自动创建标签 '%s' 失败: %w", name, err)
					}
				} else {
					return fmt.Errorf("查询标签 '%s' 失败: %w", name, err)
				}
			}

			link := models.DocumentTagLink{
				DocumentID: documentID,
				TagID:      tag.ID,
			}
			if err := tx.Create(&link).Error; err != nil {
				return fmt.Errorf("建立文档标签关联失败: %w", err)
			}
		}

		return nil
	})
}

// GetDocumentTags 获取指定文档关联的全部标签
func (s *TagService) GetDocumentTags(documentID uint) ([]models.ArchiveTag, error) {
	if documentID == 0 {
		return nil, errors.New("文档 ID 不能为空")
	}

	var tags []models.ArchiveTag
	err := s.db.
		Joins("JOIN document_tag_links ON document_tag_links.tag_id = archive_tags.id").
		Where("document_tag_links.document_id = ?", documentID).
		Order("archive_tags.name ASC").
		Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("查询文档标签失败: %w", err)
	}
	return tags, nil
}

// MigrateLegacyTags 将 documents 表遗留的 Tags JSON 字符串迁移到标签表与关联表
// 每个文档解析其 Tags 字段，为每个标签名获取或创建全局标签（user_id=NULL），并建立关联
// 迁移完成后将原 Tags 字段清空；整个过程在事务中执行
func (s *TagService) MigrateLegacyTags() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var documents []models.Document
		if err := tx.Where("tags IS NOT NULL AND tags <> ''").Find(&documents).Error; err != nil {
			return fmt.Errorf("查询待迁移文档失败: %w", err)
		}

		for _, doc := range documents {
			var names []string
			if err := json.Unmarshal([]byte(doc.Tags), &names); err != nil {
				// JSON 格式异常时跳过该文档，避免影响整体迁移
				fmt.Printf("文档 %d 的 Tags 字段 JSON 解析失败，已跳过: %v\n", doc.ID, err)
				continue
			}

			for _, raw := range names {
				name := strings.TrimSpace(raw)
				if name == "" {
					continue
				}

				var tag models.ArchiveTag
				err := tx.Where("name = ? AND user_id IS NULL", name).First(&tag).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						tag = models.ArchiveTag{
							Name:  name,
							Color: "#3b82f6",
						}
						if err := tx.Create(&tag).Error; err != nil {
							return fmt.Errorf("迁移时创建标签 '%s' 失败: %w", name, err)
						}
					} else {
						return fmt.Errorf("迁移时查询标签 '%s' 失败: %w", name, err)
					}
				}

				link := models.DocumentTagLink{
					DocumentID: doc.ID,
					TagID:      tag.ID,
				}
				// 使用 FirstOrCreate 防止重复关联（理论上不会重复，但可防御）
				if err := tx.Where("document_id = ? AND tag_id = ?", doc.ID, tag.ID).FirstOrCreate(&link).Error; err != nil {
					return fmt.Errorf("迁移时建立文档 %d 与标签 %d 关联失败: %w", doc.ID, tag.ID, err)
				}
			}

			// 清空原 Tags 字段
			if err := tx.Model(&models.Document{}).Where("id = ?", doc.ID).Update("tags", "").Error; err != nil {
				return fmt.Errorf("清空文档 %d 旧 Tags 字段失败: %w", doc.ID, err)
			}
		}

		return nil
	})
}
