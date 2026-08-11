package main

import (
	"log"

	"gorm.io/gorm"

	"siapp/internal/models"
)

func seedKnowledgeBases(db *gorm.DB) error {
	type tmpl struct {
		Name, Desc, Module, Vis string
		Masks                   []struct{ Field, Pattern, Exempt string }
		Rules                   []struct{ RoleLevel string }
	}
	list := []tmpl{
		{Name: "员工花名册", Desc: "全体员工人事档案信息", Module: "employee", Vis: "public",
			Masks: []struct{ Field, Pattern, Exempt string }{
				{"id_card", "front3back4", "admin"},
				{"phone", "front3back4", "admin"},
				{"address", "all_star", "admin"},
				{"emergency_contact", "all_star", "admin"},
			},
			Rules: []struct{ RoleLevel string }{{"viewer"}},
		},
		{Name: "宿舍管理", Desc: "宿舍入住与缴费信息", Module: "dormitory", Vis: "restricted",
			Rules: []struct{ RoleLevel string }{{"manager"}, {"admin"}},
		},
		{Name: "薪酬社保", Desc: "社保公积金缴费记录", Module: "insurance", Vis: "restricted",
			Masks: []struct{ Field, Pattern, Exempt string }{{"bank_card", "front3back4", "admin"}},
			Rules: []struct{ RoleLevel string }{{"admin"}},
		},
		{Name: "档案管理", Desc: "四类法定档案全生命周期", Module: "archives", Vis: "private",
			Rules: []struct{ RoleLevel string }{{"admin"}},
		},
		{Name: "办公用品", Desc: "办公用品采购与请款", Module: "office", Vis: "restricted",
			Rules: []struct{ RoleLevel string }{{"manager"}, {"admin"}},
		},
		{Name: "食堂采购", Desc: "食堂食材采购与收入", Module: "canteen", Vis: "restricted",
			Rules: []struct{ RoleLevel string }{{"manager"}, {"admin"}},
		},
		{Name: "财务发票", Desc: "发票录入与报销审批", Module: "invoice", Vis: "private",
			Masks: []struct{ Field, Pattern, Exempt string }{{"amount", "all_star", "admin"}},
			Rules: []struct{ RoleLevel string }{{"admin"}},
		},
		{Name: "手动创建", Desc: "用户自定义知识库（模板）", Module: "custom", Vis: "restricted"},
	}
	for _, t := range list {
		var existing models.KnowledgeBase
		if err := db.Where("name = ?", t.Name).First(&existing).Error; err == nil {
			continue
		}
		kb := &models.KnowledgeBase{Name: t.Name, Description: t.Desc, SourceModule: t.Module, Visibility: t.Vis, IsSystem: true, ChunkingConfig: models.DefaultChunkingConfig()}
		if err := db.Create(kb).Error; err != nil {
			log.Printf("seed knowledge base %s: %v", t.Name, err)
			continue
		}
		for _, m := range t.Masks {
			db.Create(&models.KBFieldMask{KnowledgeBaseID: kb.ID, FieldName: m.Field, MaskPattern: m.Pattern, ExemptRole: &m.Exempt})
		}
		for _, r := range t.Rules {
			db.Create(&models.KBAccessRule{KnowledgeBaseID: kb.ID, RoleLevel: &r.RoleLevel})
		}
	}
	log.Printf("知识库模板种子完成: %d 个模板已创建", len(list))
	return nil
}
