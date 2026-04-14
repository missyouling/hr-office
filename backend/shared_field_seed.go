package main

import (
	"log"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// seedSharedFieldsAndGroups 预填充共用字段和专用字段组
func seedSharedFieldsAndGroups(db *gorm.DB) error {
	// 必填共用字段
	sharedFields := []models.ArchiveSharedField{
		{
			FieldName:    "archive_id",
			FieldLabel:   "档案编号",
			FieldType:    "text",
			IsRequired:   true,
			IsOCRRelated: false,
			SortOrder:    1,
		},
		{
			FieldName:    "archive_title",
			FieldLabel:   "档案标题",
			FieldType:    "text",
			IsRequired:   true,
			IsOCRRelated: true,
			SortOrder:    2,
		},
		{
			FieldName:    "category_id",
			FieldLabel:   "所属分类",
			FieldType:    "select",
			IsRequired:   true,
			IsOCRRelated: false,
			SortOrder:    3,
		},
		{
			FieldName:    "archive_date",
			FieldLabel:   "归档日期",
			FieldType:    "date",
			IsRequired:   true,
			IsOCRRelated: true,
			SortOrder:    4,
		},
		{
			FieldName:    "owner",
			FieldLabel:   "归档人/责任人",
			FieldType:    "user",
			IsRequired:   true,
			IsOCRRelated: false,
			SortOrder:    5,
		},
		{
			FieldName:    "security_level",
			FieldLabel:   "密级",
			FieldType:    "select",
			IsRequired:   false,
			IsOCRRelated: false,
			Options:      "公开,内部,机密",
			SortOrder:    6,
		},
		{
			FieldName:    "retention_period",
			FieldLabel:   "保管期限",
			FieldType:    "select",
			IsRequired:   true,
			IsOCRRelated: false,
			Options:      "3年,5年,10年,永久",
			SortOrder:    7,
		},
		{
			FieldName:    "file_path",
			FieldLabel:   "电子文件/附件",
			FieldType:    "file",
			IsRequired:   true,
			IsOCRRelated: false,
			SortOrder:    8,
		},
		{
			FieldName:    "file_format",
			FieldLabel:   "文件格式",
			FieldType:    "text",
			IsRequired:   false,
			IsOCRRelated: true,
			SortOrder:    9,
		},
		{
			FieldName:    "file_size",
			FieldLabel:   "文件大小",
			FieldType:    "number",
			IsRequired:   false,
			IsOCRRelated: true,
			SortOrder:    10,
		},
		{
			FieldName:    "summary",
			FieldLabel:   "摘要",
			FieldType:    "textarea",
			IsRequired:   false,
			IsOCRRelated: true,
			SortOrder:    11,
		},
		{
			FieldName:    "tags",
			FieldLabel:   "标签",
			FieldType:    "multiselect",
			IsRequired:   false,
			IsOCRRelated: true,
			SortOrder:    12,
		},
	}

	// 创建共用字段
	for _, field := range sharedFields {
		var existing models.ArchiveSharedField
		if err := db.Where("field_name = ?", field.FieldName).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&field).Error; err != nil {
					log.Printf("failed to create shared field %s: %v", field.FieldName, err)
				}
			}
		}
	}

	// 创建专用字段组和字段定义
	if err := createFieldGroupsAndDefinitions(db); err != nil {
		return err
	}

	log.Println("共用字段和专用字段组预填充完成")
	return nil
}

// createFieldGroupsAndDefinitions 创建专用字段组和字段定义
func createFieldGroupsAndDefinitions(db *gorm.DB) error {
	// 获取所有二级分类
	var subCategories []models.DocumentSubCategory
	if err := db.Find(&subCategories).Error; err != nil {
		return err
	}

	// 创建字段定义映射
	fieldDefinitions := map[string][]struct {
		FieldName    string
		FieldLabel   string
		FieldType    string
		IsOCRRelated bool
		Options      string
	}{
		"0101": {
			{FieldName: "document_no", FieldLabel: "发文字号", FieldType: "text", IsOCRRelated: true},
			{FieldName: "effective_date", FieldLabel: "生效日期", FieldType: "date", IsOCRRelated: true},
		},
		"0102": {
			{FieldName: "license_no", FieldLabel: "证件编号", FieldType: "text", IsOCRRelated: true},
			{FieldName: "issuing_authority", FieldLabel: "发证机关", FieldType: "text", IsOCRRelated: true},
			{FieldName: "valid_until", FieldLabel: "有效期至", FieldType: "date", IsOCRRelated: true},
		},
		"0103": {
			{FieldName: "record_period", FieldLabel: "记录周期", FieldType: "text", IsOCRRelated: true},
		},
		"0201": {
			{FieldName: "emp_name", FieldLabel: "员工姓名", FieldType: "text", IsOCRRelated: true},
			{FieldName: "emp_id", FieldLabel: "员工工号", FieldType: "text", IsOCRRelated: true},
			{FieldName: "position", FieldLabel: "岗位职级", FieldType: "text", IsOCRRelated: true},
		},
		"0202": {
			{FieldName: "attendance_month", FieldLabel: "考勤月份", FieldType: "text", IsOCRRelated: true},
			{FieldName: "days_expected", FieldLabel: "应出勤", FieldType: "number", IsOCRRelated: false},
			{FieldName: "days_actual", FieldLabel: "实际出勤", FieldType: "number", IsOCRRelated: false},
		},
		"0203": {
			{FieldName: "cert_no", FieldLabel: "证书编号", FieldType: "text", IsOCRRelated: true},
			{FieldName: "review_date", FieldLabel: "复审日期", FieldType: "date", IsOCRRelated: true},
		},
		"0301": {
			{FieldName: "asset_no", FieldLabel: "资产编号", FieldType: "text", IsOCRRelated: true},
			{FieldName: "model", FieldLabel: "设备型号", FieldType: "text", IsOCRRelated: true},
			{FieldName: "next_maint_date", FieldLabel: "下次维保日期", FieldType: "date", IsOCRRelated: true},
		},
		"0302": {
			{FieldName: "plate_no", FieldLabel: "车牌号", FieldType: "text", IsOCRRelated: true},
			{FieldName: "vehicle_type", FieldLabel: "车辆类型", FieldType: "text", IsOCRRelated: true},
			{FieldName: "insurance_expiry", FieldLabel: "保险/年检到期日", FieldType: "date", IsOCRRelated: true},
		},
		"0303": {
			{FieldName: "asset_type", FieldLabel: "资产类型", FieldType: "select", IsOCRRelated: true, Options: "微型主机,NAS存储,路由器,交换机,服务器,其他"},
			{FieldName: "network_info", FieldLabel: "IP/MAC地址", FieldType: "text", IsOCRRelated: true},
			{FieldName: "config_details", FieldLabel: "配置详情", FieldType: "textarea", IsOCRRelated: true},
		},
		"0401": {
			{FieldName: "vendor_name", FieldLabel: "供应商名称", FieldType: "text", IsOCRRelated: true},
			{FieldName: "amount", FieldLabel: "合同金额", FieldType: "number", IsOCRRelated: true},
			{FieldName: "payment_terms", FieldLabel: "付款节点", FieldType: "text", IsOCRRelated: true},
		},
		"0402": {
			{FieldName: "customer_name", FieldLabel: "客户名称", FieldType: "text", IsOCRRelated: true},
			{FieldName: "end_date", FieldLabel: "合同截止日", FieldType: "date", IsOCRRelated: true},
		},
	}

	// 为每个二级分类创建字段组和字段定义
	for _, sub := range subCategories {
		// 检查是否已有字段组
		var existingGroup models.ArchiveFieldGroup
		if err := db.Where("sub_category_id = ?", sub.ID).First(&existingGroup).Error; err == nil {
			continue // 已有字段组，跳过
		}

		// 获取CategoryCode (一级分类代码)
		var parentCat models.DocumentCategory
		if err := db.First(&parentCat, sub.CategoryID).Error; err != nil {
			log.Printf("failed to find parent category for subcategory %d: %v", sub.ID, err)
			continue
		}

		// 创建字段组
		group := models.ArchiveFieldGroup{
			SubCategoryID:   sub.ID,
			SubCategoryCode: sub.Code,  // 使用二级分类代码，如 0101, 0102...
			Name:            sub.Name + "字段组",
			Description:     "为" + sub.Name + "创建的专用字段组",
			SortOrder:       1,
		}

		if err := db.Create(&group).Error; err != nil {
			log.Printf("failed to create field group for subcategory %s: %v", sub.Code, err)
			continue
		}

		// 为该字段组创建字段定义
		if fields, ok := fieldDefinitions[sub.Code]; ok {
			for i, fieldDef := range fields {
				field := models.ArchiveFieldDefinition{
					SubCategoryID: sub.ID,
					GroupID:       &group.ID,
					FieldName:     fieldDef.FieldName,
					FieldLabel:    fieldDef.FieldLabel,
					FieldType:     fieldDef.FieldType,
					Required:      false,
					IsOCRRelated:  fieldDef.IsOCRRelated,
					Options:       fieldDef.Options,
					SortOrder:     i + 1,
					Visible:       true,
					Editable:      true,
				}

				if err := db.Create(&field).Error; err != nil {
					log.Printf("failed to create field definition %s for group %d: %v", fieldDef.FieldName, group.ID, err)
				}
			}
		}
	}

	return nil
}
