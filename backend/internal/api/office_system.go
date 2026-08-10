package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ======== 系统重置 / 备份 ========

func (h *OfficeSupplyHandler) resetSystem(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	type resetOptions struct {
		Categories      bool `json:"categories"`
		Suppliers       bool `json:"suppliers"`
		Supplies        bool `json:"supplies"`
		Purchases       bool `json:"purchases"`
		PaymentRequests bool `json:"payment_requests"`
	}
	opts := resetOptions{
		Categories: true, Suppliers: true, Supplies: true,
		Purchases: true, PaymentRequests: true,
	}
	json.NewDecoder(r.Body).Decode(&opts)

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if opts.PaymentRequests {
			tx.Where("user_id = ?", userID).Delete(&models.OfficePaymentRequest{})
		}
		if opts.Purchases {
			tx.Where("user_id = ?", userID).Delete(&models.OfficePurchaseItem{})
			tx.Where("user_id = ?", userID).Delete(&models.OfficePurchase{})
		}
		if opts.Supplies {
			tx.Where("user_id = ?", userID).Delete(&models.OfficeSupply{})
		}
		if opts.Suppliers {
			tx.Where("user_id = ?", userID).Delete(&models.OfficeSupplier{})
		}
		if opts.Categories {
			tx.Where("user_id = ?", userID).Delete(&models.OfficeCategory{})
		}
		return nil
	})
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "系统重置失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) listBackups(w http.ResponseWriter, r *http.Request) {
	_, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var backups []models.OfficeBackupLog
	h.db.Order("id DESC").Find(&backups)
	// 返回时不包含 data 字段（太大）
	type backupItem struct {
		ID          uint      `json:"id"`
		Filename    string    `json:"filename"`
		Description string    `json:"description"`
		FileSize    int64     `json:"file_size"`
		CreatedAt   time.Time `json:"created_at"`
	}
	items := make([]backupItem, len(backups))
	for i, b := range backups {
		items[i] = backupItem{
			ID: b.ID, Filename: b.Filename, Description: b.Description,
			FileSize: b.FileSize, CreatedAt: b.CreatedAt,
		}
	}
	respondOfficeOK(w, map[string]interface{}{"items": items})
}

func (h *OfficeSupplyHandler) createBackup(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	type backupPayload struct {
		Description string `json:"description"`
	}
	var payload backupPayload
	json.NewDecoder(r.Body).Decode(&payload)

	// 导出全部数据
	type backupData struct {
		Categories      []models.OfficeCategory       `json:"categories"`
		Suppliers       []models.OfficeSupplier       `json:"suppliers"`
		Supplies        []models.OfficeSupply         `json:"supplies"`
		Purchases       []models.OfficePurchase       `json:"purchases"`
		PurchaseItems   []models.OfficePurchaseItem   `json:"purchase_items"`
		PaymentRequests []models.OfficePaymentRequest `json:"payment_requests"`
		ExportedAt      string                        `json:"exported_at"`
	}
	var bd backupData
	h.db.Where("user_id = ?", userID).Find(&bd.Categories)
	h.db.Where("user_id = ?", userID).Find(&bd.Suppliers)
	h.db.Where("user_id = ?", userID).Find(&bd.Supplies)
	h.db.Where("user_id = ?", userID).Find(&bd.Purchases)
	h.db.Where("user_id = ?", userID).Find(&bd.PurchaseItems)
	h.db.Where("user_id = ?", userID).Find(&bd.PaymentRequests)
	bd.ExportedAt = time.Now().Format(time.RFC3339)

	dataJSON, err := json.Marshal(bd)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "序列化备份数据失败")
		return
	}
	filename := "backup_" + time.Now().Format("2006-01-02T15-04-05") + ".json"

	backup := models.OfficeBackupLog{
		UserID:      uintPointer(userID),
		Filename:    filename,
		Description: payload.Description,
		FileSize:    int64(len(dataJSON)),
		Data:        string(dataJSON),
	}
	if err := h.db.Create(&backup).Error; err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "创建备份失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"filename": filename})
}

func (h *OfficeSupplyHandler) restoreBackup(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的备份ID")
		return
	}
	var backup models.OfficeBackupLog
	if err := h.db.Where("id = ?", id).First(&backup).Error; err != nil {
		respondOfficeError(w, http.StatusNotFound, "备份不存在")
		return
	}

	var bd struct {
		Categories      []models.OfficeCategory       `json:"categories"`
		Suppliers       []models.OfficeSupplier       `json:"suppliers"`
		Supplies        []models.OfficeSupply         `json:"supplies"`
		Purchases       []models.OfficePurchase       `json:"purchases"`
		PurchaseItems   []models.OfficePurchaseItem   `json:"purchase_items"`
		PaymentRequests []models.OfficePaymentRequest `json:"payment_requests"`
	}
	if err := json.Unmarshal([]byte(backup.Data), &bd); err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "解析备份数据失败")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 清除当前用户数据
		tables := []interface{}{
			&models.OfficePurchaseItem{}, &models.OfficePurchase{},
			&models.OfficeSupply{}, &models.OfficeSupplier{},
			&models.OfficeCategory{}, &models.OfficePaymentRequest{},
		}
		for _, t := range tables {
			if txErr := tx.Where("user_id = ?", userID).Delete(t).Error; txErr != nil {
				return txErr
			}
		}
		// 恢复数据
		for _, c := range bd.Categories {
			c.UserID = uintPointer(userID)
			tx.Create(&c)
		}
		for _, s := range bd.Suppliers {
			s.UserID = uintPointer(userID)
			tx.Create(&s)
		}
		for _, s := range bd.Supplies {
			s.UserID = uintPointer(userID)
			tx.Create(&s)
		}
		for _, p := range bd.Purchases {
			p.UserID = uintPointer(userID)
			tx.Create(&p)
		}
		for _, pi := range bd.PurchaseItems {
			pi.UserID = uintPointer(userID)
			tx.Create(&pi)
		}
		for _, pr := range bd.PaymentRequests {
			pr.UserID = uintPointer(userID)
			tx.Create(&pr)
		}
		return nil
	})
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "恢复备份失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	_, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的备份ID")
		return
	}
	h.db.Where("id = ?", id).Delete(&models.OfficeBackupLog{})
	respondOfficeOK(w, map[string]interface{}{})
}
