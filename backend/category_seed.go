package main

import (
	"log"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// seedDocumentCategories 预填充档案分类数据
func seedDocumentCategories(db *gorm.DB) error {
	categories := []models.DocumentCategory{
		{Code: "01", Name: "综合行政类", Description: "制度公文、证照资质、后勤安保", SortOrder: 1},
		{Code: "02", Name: "人力资源类", Description: "人事档案、考勤排班、培训考核", SortOrder: 2},
		{Code: "03", Name: "固定资产类", Description: "生产设备、车辆特种、IT网络", SortOrder: 3},
		{Code: "04", Name: "合同与法务类", Description: "采购合同、销售业务合同", SortOrder: 4},
	}

	// 创建一级分类
	for _, cat := range categories {
		var existing models.DocumentCategory
		if err := db.Where("code = ?", cat.Code).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&cat).Error; err != nil {
					log.Printf("failed to create category %s: %v", cat.Code, err)
				}
			}
		}
	}

	// 为每个分类创建二级分类
	categorySubCategories := map[string][]models.DocumentSubCategory{
		"01": {
			{Code: "0101", Name: "制度与公文", Description: "企业规章制度、通知公告、会议纪要等", CategoryCode: "01", SortOrder: 1},
			{Code: "0102", Name: "证照与资质", Description: "营业执照、资质证书、许可证等", CategoryCode: "01", SortOrder: 2},
			{Code: "0103", Name: "后勤与安保", Description: "安保记录、后勤管理、消防检查等", CategoryCode: "01", SortOrder: 3},
		},
		"02": {
			{Code: "0201", Name: "员工人事档案", Description: "入职资料、劳动合同、人事变动等", CategoryCode: "02", SortOrder: 1},
			{Code: "0202", Name: "考勤与排班", Description: "考勤记录、排班表、加班申请等", CategoryCode: "02", SortOrder: 2},
			{Code: "0203", Name: "培训与考核", Description: "培训记录、考核结果、资格证书等", CategoryCode: "02", SortOrder: 3},
		},
		"03": {
			{Code: "0301", Name: "生产设备档案", Description: "设备台账、维保记录、操作手册等", CategoryCode: "03", SortOrder: 1},
			{Code: "0302", Name: "车辆与特种设备", Description: "车辆档案、特种设备检验、保险等", CategoryCode: "03", SortOrder: 2},
			{Code: "0303", Name: "IT与网络资产", Description: "服务器、网络设备、软件许可等", CategoryCode: "03", SortOrder: 3},
		},
		"04": {
			{Code: "0401", Name: "采购合同", Description: "原材料采购、设备采购、服务采购等", CategoryCode: "04", SortOrder: 1},
			{Code: "0402", Name: "销售业务合同", Description: "产品销售、技术服务、代理协议等", CategoryCode: "04", SortOrder: 2},
		},
	}

	for catCode, subCats := range categorySubCategories {
		// 获取分类ID
		var cat models.DocumentCategory
		if err := db.Where("code = ?", catCode).First(&cat).Error; err != nil {
			log.Printf("category %s not found: %v", catCode, err)
			continue
		}

		for _, sub := range subCats {
			var existing models.DocumentSubCategory
			if err := db.Where("category_id = ? AND code = ?", cat.ID, sub.Code).First(&existing).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					sub.CategoryID = cat.ID
					if err := db.Create(&sub).Error; err != nil {
						log.Printf("failed to create subcategory %s-%s: %v", catCode, sub.Code, err)
					}
				}
			}
		}
	}

	log.Println("档案分类数据预填充完成")
	return nil
}
