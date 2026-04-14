package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

const (
	defaultProvidentUnitName    = "重庆星达铜业有限公司"
	defaultProvidentUnitAccount = "201005128130"
	defaultProvidentRatio       = 6.0
)

type providentRecordPayload struct {
	PersonalAccount   string  `json:"personal_account"`
	Name              string  `json:"name"`
	IdentityNumber    string  `json:"identity_number"`
	PersonalBase      float64 `json:"personal_base"`
	ContributionRatio float64 `json:"contribution_ratio"`
	PersonalAmount    float64 `json:"personal_amount"`
	CompanyAmount     float64 `json:"company_amount"`
	Notes             string  `json:"notes"`
}

type providentSettingsPayload struct {
	UnitName    string `json:"unit_name"`
	UnitAccount string `json:"unit_account"`
}

type createProvidentBillPayload struct {
	Month     string `json:"month"`
	Overwrite bool   `json:"overwrite"`
}

type sealProvidentPayload struct {
	Date string `json:"date"`
}

type unsealProvidentPayload struct {
	Date string `json:"date"`
}

func parseOptionalDate(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func normalizeMonthLabel(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("month is required")
	}
	parsed, err := time.Parse("2006-01", trimmed)
	if err != nil {
		return "", err
	}
	return parsed.Format("2006-01"), nil
}

func (h *Handler) listProvidentRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	keyword := strings.TrimSpace(r.URL.Query().Get("q"))

	query := h.db.Where("user_id = ?", userID).Model(&models.ProvidentFundRecord{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR personal_account LIKE ? OR identity_number LIKE ?", like, like, like)
	}

	var records []models.ProvidentFundRecord
	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load provident fund records", err)
		return
	}
	respondJSON(w, http.StatusOK, records)
}

func (h *Handler) createProvidentRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload providentRecordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if strings.TrimSpace(payload.PersonalAccount) == "" || strings.TrimSpace(payload.IdentityNumber) == "" {
		respondError(w, http.StatusBadRequest, "personal_account and identity_number are required", nil)
		return
	}

	var existing models.ProvidentFundRecord
	if err := h.db.Where("user_id = ? AND identity_number = ?", userID, payload.IdentityNumber).First(&existing).Error; err == nil {
		respondError(w, http.StatusConflict, "record already exists", nil)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(w, http.StatusInternalServerError, "failed to check duplicates", err)
		return
	}

	ratio := payload.ContributionRatio
	if ratio <= 0 {
		ratio = defaultProvidentRatio
	}

	record := models.ProvidentFundRecord{
		UserID:            uintPointer(userID),
		PersonalAccount:   strings.TrimSpace(payload.PersonalAccount),
		Name:              strings.TrimSpace(payload.Name),
		IdentityNumber:    strings.TrimSpace(payload.IdentityNumber),
		PersonalBase:      payload.PersonalBase,
		ContributionRatio: ratio,
		PersonalAmount:    payload.PersonalAmount,
		CompanyAmount:     payload.CompanyAmount,
		TotalAmount:       payload.PersonalAmount + payload.CompanyAmount,
		Notes:             strings.TrimSpace(payload.Notes),
		Status:            "active",
	}

	if err := h.db.Create(&record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create record", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

func (h *Handler) updateProvidentRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	recordID, err := parseUintParam(r, "recordID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid record id", err)
		return
	}

	var record models.ProvidentFundRecord
	if err := h.db.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "record not found", err)
		return
	}

	var payload providentRecordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	identity := strings.TrimSpace(payload.IdentityNumber)
	if identity == "" {
		identity = record.IdentityNumber
	}
	if identity != record.IdentityNumber {
		var tmp models.ProvidentFundRecord
		if err := h.db.Where("user_id = ? AND identity_number = ?", userID, identity).First(&tmp).Error; err == nil {
			respondError(w, http.StatusConflict, "record already exists", nil)
			return
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusInternalServerError, "failed to check duplicates", err)
			return
		}
	}

	ratio := payload.ContributionRatio
	if ratio <= 0 {
		ratio = defaultProvidentRatio
	}

	updates := map[string]any{
		"personal_account":   strings.TrimSpace(payload.PersonalAccount),
		"name":               strings.TrimSpace(payload.Name),
		"identity_number":    identity,
		"personal_base":      payload.PersonalBase,
		"contribution_ratio": ratio,
		"personal_amount":    payload.PersonalAmount,
		"company_amount":     payload.CompanyAmount,
		"total_amount":       payload.PersonalAmount + payload.CompanyAmount,
		"notes":              strings.TrimSpace(payload.Notes),
	}

	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update record", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}

func (h *Handler) sealProvidentRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	recordID, err := parseUintParam(r, "recordID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid record id", err)
		return
	}

	var record models.ProvidentFundRecord
	if err := h.db.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "record not found", err)
		return
	}
	if record.Status == "sealed" {
		respondError(w, http.StatusBadRequest, "record already sealed", nil)
		return
	}

	var payload sealProvidentPayload
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}
	}
	sealedAt := time.Now()
	if payload.Date != "" {
		parsedDate, err := parseOptionalDate(payload.Date)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid seal date", err)
			return
		}
		if parsedDate != nil {
			sealedAt = *parsedDate
		}
	}

	if err := h.db.Model(&record).Updates(map[string]any{
		"status":      "sealed",
		"sealed_at":   &sealedAt,
		"unsealed_at": nil,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to seal record", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}

func (h *Handler) unsealProvidentRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	recordID, err := parseUintParam(r, "recordID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid record id", err)
		return
	}

	var record models.ProvidentFundRecord
	if err := h.db.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "record not found", err)
		return
	}
	if record.Status != "sealed" {
		respondError(w, http.StatusBadRequest, "record is not sealed", nil)
		return
	}

	var payload unsealProvidentPayload
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}
	}
	unsealDate := time.Now()
	if payload.Date != "" {
		parsedDate, err := parseOptionalDate(payload.Date)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid unseal date", err)
			return
		}
		if parsedDate != nil {
			unsealDate = *parsedDate
		}
	}

	updates := map[string]any{
		"status":      "active",
		"sealed_at":   nil,
		"unsealed_at": &unsealDate,
	}
	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to unseal record", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}

func (h *Handler) getProvidentSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var settings models.ProvidentFundSettings
	if err := h.db.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondJSON(w, http.StatusOK, map[string]string{
				"unit_name":    defaultProvidentUnitName,
				"unit_account": defaultProvidentUnitAccount,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load settings", err)
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

func (h *Handler) updateProvidentSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload providentSettingsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if strings.TrimSpace(payload.UnitName) == "" || strings.TrimSpace(payload.UnitAccount) == "" {
		respondError(w, http.StatusBadRequest, "unit_name and unit_account are required", nil)
		return
	}

	var settings models.ProvidentFundSettings
	err = h.db.Where("user_id = ?", userID).First(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		settings = models.ProvidentFundSettings{
			UserID:      uintPointer(userID),
			UnitName:    strings.TrimSpace(payload.UnitName),
			UnitAccount: strings.TrimSpace(payload.UnitAccount),
		}
		if err := h.db.Create(&settings).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to save settings", err)
			return
		}
		respondJSON(w, http.StatusOK, settings)
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load settings", err)
		return
	}

	if err := h.db.Model(&settings).Updates(map[string]any{
		"unit_name":    strings.TrimSpace(payload.UnitName),
		"unit_account": strings.TrimSpace(payload.UnitAccount),
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update settings", err)
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

func (h *Handler) createProvidentBill(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload createProvidentBillPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	monthLabel, err := normalizeMonthLabel(payload.Month)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid month", err)
		return
	}

	var existing models.ProvidentFundBill
	if err := h.db.Where("user_id = ? AND month_label = ?", userID, monthLabel).First(&existing).Error; err == nil {
		if !payload.Overwrite {
			respondError(w, http.StatusConflict, "bill already exists for this month", nil)
			return
		}
		if err := h.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("bill_id = ?", existing.ID).Delete(&models.ProvidentFundBillItem{}).Error; err != nil {
				return err
			}
			return tx.Delete(&existing).Error
		}); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to overwrite existing bill", err)
			return
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(w, http.StatusInternalServerError, "failed to check bill", err)
		return
	}

	var records []models.ProvidentFundRecord
	if err := h.db.Where("user_id = ? AND status = ?", userID, "active").Order("name ASC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load records", err)
		return
	}
	if len(records) == 0 {
		respondError(w, http.StatusBadRequest, "no active records to generate bill", nil)
		return
	}

	bill := models.ProvidentFundBill{
		UserID:      uintPointer(userID),
		MonthLabel:  monthLabel,
		RecordCount: len(records),
	}

	items := make([]models.ProvidentFundBillItem, 0, len(records))
	for _, record := range records {
		bill.PersonalAmountTotal += record.PersonalAmount
		bill.CompanyAmountTotal += record.CompanyAmount
		bill.CombinedAmountTotal += record.TotalAmount
		items = append(items, models.ProvidentFundBillItem{
			RecordID:        record.ID,
			PersonalAccount: record.PersonalAccount,
			Name:            record.Name,
			IdentityNumber:  record.IdentityNumber,
			PersonalAmount:  record.PersonalAmount,
			CompanyAmount:   record.CompanyAmount,
			TotalAmount:     record.TotalAmount,
		})
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&bill).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].BillID = bill.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate bill", err)
		return
	}

	bill.Items = items
	respondJSON(w, http.StatusCreated, bill)
}

func (h *Handler) listProvidentBills(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	withItems := strings.EqualFold(r.URL.Query().Get("with_items"), "true")
	monthFilter := strings.TrimSpace(r.URL.Query().Get("month"))

	query := h.db.Where("user_id = ?", userID).Model(&models.ProvidentFundBill{})
	if monthFilter != "" {
		query = query.Where("month_label = ?", monthFilter)
	}
	if withItems {
		query = query.Preload("Items")
	}
	var bills []models.ProvidentFundBill
	if err := query.Order("month_label DESC").Find(&bills).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load bills", err)
		return
	}
	respondJSON(w, http.StatusOK, bills)
}

func (h *Handler) getProvidentBill(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	billID, err := parseUintParam(r, "billID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid bill id", err)
		return
	}

	var bill models.ProvidentFundBill
	if err := h.db.Preload("Items").Where("id = ? AND user_id = ?", billID, userID).First(&bill).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "bill not found", err)
		return
	}
	respondJSON(w, http.StatusOK, bill)
}

func (h *Handler) deleteProvidentBill(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	billID, err := parseUintParam(r, "billID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid bill id", err)
		return
	}

	var bill models.ProvidentFundBill
	if err := h.db.Where("id = ? AND user_id = ?", billID, userID).First(&bill).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "bill not found", err)
		return
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("bill_id = ?", bill.ID).Delete(&models.ProvidentFundBillItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&bill).Error
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete bill", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
