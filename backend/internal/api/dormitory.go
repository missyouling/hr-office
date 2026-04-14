package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

func parseUintParam(r *http.Request, key string) (uint64, error) {
	value := strings.TrimSpace(chi.URLParam(r, key))
	if value == "" {
		return 0, errors.New("missing id")
	}
	return strconv.ParseUint(value, 10, 64)
}

// region: Sites

func (h *Handler) listDormSites(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var sites []models.DormSite
	if err := h.db.Preload("Buildings").Where("user_id = ?", userID).Order("id DESC").Find(&sites).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load sites", err)
		return
	}
	respondJSON(w, http.StatusOK, sites)
}

type dormSitePayload struct {
	Name            string          `json:"name"`
	Address         string          `json:"address"`
	ContactName     string          `json:"contact_name"`
	ContactPhone    string          `json:"contact_phone"`
	BuildingNumber  string          `json:"building_number"`
	PropertyCompany string          `json:"property_company"`
	PropertyContact string          `json:"property_contact"`
	SupportWechat   string          `json:"support_wechat"`
	Description     string          `json:"description"`
	ChargeConfig    json.RawMessage `json:"charge_config"`
}

func (h *Handler) createDormSite(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload dormSitePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		respondError(w, http.StatusBadRequest, "name is required", nil)
		return
	}
	site := models.DormSite{
		UserID:          uintPointer(userID),
		Name:            payload.Name,
		Address:         payload.Address,
		ContactName:     payload.ContactName,
		ContactPhone:    payload.ContactPhone,
		BuildingNumber:  payload.BuildingNumber,
		PropertyCompany: payload.PropertyCompany,
		PropertyContact: payload.PropertyContact,
		SupportWechat:   payload.SupportWechat,
		Description:     payload.Description,
		ChargeConfig:    datatypes.JSON(payload.ChargeConfig),
	}
	if err := h.db.Create(&site).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create site", err)
		return
	}
	respondJSON(w, http.StatusCreated, site)
}

func (h *Handler) updateDormSite(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	siteID, err := parseUintParam(r, "siteID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id", err)
		return
	}
	var site models.DormSite
	if err := h.db.Where("id = ? AND user_id = ?", siteID, userID).First(&site).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "site not found", err)
		return
	}
	var payload dormSitePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"name":             payload.Name,
		"address":          payload.Address,
		"contact_name":     payload.ContactName,
		"contact_phone":    payload.ContactPhone,
		"building_number":  payload.BuildingNumber,
		"property_company": payload.PropertyCompany,
		"property_contact": payload.PropertyContact,
		"support_wechat":   payload.SupportWechat,
		"description":      payload.Description,
	}
	if payload.ChargeConfig != nil {
		updates["charge_config"] = datatypes.JSON(payload.ChargeConfig)
	}
	if err := h.db.Model(&site).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update site", err)
		return
	}
	site.Name = payload.Name
	site.Address = payload.Address
	site.ContactName = payload.ContactName
	site.ContactPhone = payload.ContactPhone
	site.BuildingNumber = payload.BuildingNumber
	site.PropertyCompany = payload.PropertyCompany
	site.PropertyContact = payload.PropertyContact
	site.SupportWechat = payload.SupportWechat
	site.Description = payload.Description
	respondJSON(w, http.StatusOK, site)
}

func (h *Handler) deleteDormSite(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	siteID, err := parseUintParam(r, "siteID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id", err)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", siteID, userID).Delete(&models.DormSite{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete site", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": siteID})
}

// endregion

// region: Buildings

type dormBuildingPayload struct {
	SiteID      uint   `json:"site_id"`
	Name        string `json:"name"`
	Floors      int    `json:"floors"`
	Description string `json:"description"`
}

func (h *Handler) listDormBuildings(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID).Model(&models.DormBuilding{})
	if v := strings.TrimSpace(r.URL.Query().Get("site_id")); v != "" {
		if siteID, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Where("site_id = ?", siteID)
		}
	}
	var buildings []models.DormBuilding
	if err := query.Order("id DESC").Find(&buildings).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load buildings", err)
		return
	}
	respondJSON(w, http.StatusOK, buildings)
}

func (h *Handler) createDormBuilding(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload dormBuildingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if payload.SiteID == 0 {
		respondError(w, http.StatusBadRequest, "site_id is required", nil)
		return
	}
	var site models.DormSite
	if err := h.db.Where("id = ? AND user_id = ?", payload.SiteID, userID).First(&site).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusBadRequest
		}
		respondError(w, status, "site not found", err)
		return
	}
	building := models.DormBuilding{
		UserID:      uintPointer(userID),
		SiteID:      payload.SiteID,
		Name:        payload.Name,
		Floors:      payload.Floors,
		Description: payload.Description,
	}
	if err := h.db.Create(&building).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create building", err)
		return
	}
	respondJSON(w, http.StatusCreated, building)
}

func (h *Handler) updateDormBuilding(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	buildingID, err := parseUintParam(r, "buildingID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid building id", err)
		return
	}
	var building models.DormBuilding
	if err := h.db.Where("id = ? AND user_id = ?", buildingID, userID).First(&building).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "building not found", err)
		return
	}
	var payload dormBuildingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"name":        payload.Name,
		"floors":      payload.Floors,
		"description": payload.Description,
	}
	if payload.SiteID != 0 {
		var site models.DormSite
		if err := h.db.Where("id = ? AND user_id = ?", payload.SiteID, userID).First(&site).Error; err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusBadRequest
			}
			respondError(w, status, "site not found", err)
			return
		}
		updates["site_id"] = payload.SiteID
	}
	if err := h.db.Model(&building).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update building", err)
		return
	}
	respondJSON(w, http.StatusOK, building)
}

func (h *Handler) deleteDormBuilding(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	buildingID, err := parseUintParam(r, "buildingID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid building id", err)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", buildingID, userID).Delete(&models.DormBuilding{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete building", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": buildingID})
}

// endregion

// region: Rooms

type dormRoomPayload struct {
	SiteID          uint            `json:"site_id"`
	BuildingID      uint            `json:"building_id"`
	RoomNumber      string          `json:"room_number"`
	RoomType        string          `json:"room_type"`
	RoomCategory    string          `json:"room_category"`
	HouseLayout     string          `json:"house_layout"`
	BedCount        int             `json:"bed_count"`
	AreaSquare      float64         `json:"area_square"`
	FirstMonthFee   float64         `json:"first_month_fee"`
	MonthlyRent     float64         `json:"monthly_rent"`
	QuarterlyRent   float64         `json:"quarterly_rent"`
	PropertyFee     float64         `json:"property_fee"`
	GuaranteeFee    float64         `json:"guarantee_fee"`
	DepositFee      float64         `json:"deposit_fee"`
	WaterBase       float64         `json:"water_base"`
	ElectricBase    float64         `json:"electric_base"`
	GasBase         float64         `json:"gas_base"`
	TrashFee        float64         `json:"trash_fee"`
	WaterSupplyFee  float64         `json:"water_supply_fee"`
	SewageFee       float64         `json:"sewage_fee"`
	InventoryNote   string          `json:"inventory_note"`
	Status          string          `json:"status"`
	Notes           string          `json:"notes"`
	ChargeRates     json.RawMessage `json:"charge_rates"`
	CostBearingMode string          `json:"cost_bearing_mode"`
	CompanyName     string          `json:"company_name"`
}

func (h *Handler) listDormRooms(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID).Model(&models.DormRoom{})
	if v := strings.TrimSpace(r.URL.Query().Get("building_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Where("building_id = ?", id)
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		query = query.Where("status = ?", v)
	}
	if v := strings.TrimSpace(r.URL.Query().Get("site_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Where("site_id = ?", id)
		}
	}
	var rooms []models.DormRoom
	if err := query.Preload("Beds").Preload("Assets").Order("id DESC").Find(&rooms).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load rooms", err)
		return
	}
	respondJSON(w, http.StatusOK, rooms)
}

func (h *Handler) createDormRoom(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload dormRoomPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if payload.SiteID == 0 {
		respondError(w, http.StatusBadRequest, "site_id is required", nil)
		return
	}
	if payload.BuildingID == 0 {
		respondError(w, http.StatusBadRequest, "building_id is required", nil)
		return
	}
	var site models.DormSite
	if err := h.db.Where("id = ? AND user_id = ?", payload.SiteID, userID).First(&site).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusBadRequest
		}
		respondError(w, status, "site not found", err)
		return
	}
	var building models.DormBuilding
	if err := h.db.Where("id = ? AND user_id = ? AND site_id = ?", payload.BuildingID, userID, payload.SiteID).First(&building).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusBadRequest
		}
		respondError(w, status, "building not found", err)
		return
	}
	costMode := strings.TrimSpace(payload.CostBearingMode)
	if strings.ToLower(costMode) != "personal" {
		costMode = "company"
	}
	room := models.DormRoom{
		UserID:          uintPointer(userID),
		SiteID:          uintPointer(payload.SiteID),
		BuildingID:      payload.BuildingID,
		RoomNumber:      payload.RoomNumber,
		RoomType:        payload.RoomType,
		RoomCategory:    payload.RoomCategory,
		HouseLayout:     payload.HouseLayout,
		BedCount:        payload.BedCount,
		AreaSquare:      payload.AreaSquare,
		FirstMonthFee:   payload.FirstMonthFee,
		MonthlyRent:     payload.MonthlyRent,
		QuarterlyRent:   payload.QuarterlyRent,
		PropertyFee:     payload.PropertyFee,
		GuaranteeFee:    payload.GuaranteeFee,
		DepositFee:      payload.DepositFee,
		WaterBase:       payload.WaterBase,
		ElectricBase:    payload.ElectricBase,
		GasBase:         payload.GasBase,
		TrashFee:        payload.TrashFee,
		WaterSupplyFee:  payload.WaterSupplyFee,
		SewageFee:       payload.SewageFee,
		InventoryNote:   payload.InventoryNote,
		Status:          defaultString(payload.Status, "available"),
		Notes:           payload.Notes,
		ChargeRates:     datatypes.JSON(payload.ChargeRates),
		CostBearingMode: costMode,
		CompanyName:     payload.CompanyName,
	}
	if err := h.db.Create(&room).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create room", err)
		return
	}
	respondJSON(w, http.StatusCreated, room)
}

func (h *Handler) updateDormRoom(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	roomID, err := parseUintParam(r, "roomID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid room id", err)
		return
	}
	var room models.DormRoom
	if err := h.db.Where("id = ? AND user_id = ?", roomID, userID).First(&room).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "room not found", err)
		return
	}
	var payload dormRoomPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"room_number":      payload.RoomNumber,
		"room_type":        payload.RoomType,
		"room_category":    payload.RoomCategory,
		"house_layout":     payload.HouseLayout,
		"bed_count":        payload.BedCount,
		"area_square":      payload.AreaSquare,
		"first_month_fee":  payload.FirstMonthFee,
		"monthly_rent":     payload.MonthlyRent,
		"quarterly_rent":   payload.QuarterlyRent,
		"property_fee":     payload.PropertyFee,
		"guarantee_fee":    payload.GuaranteeFee,
		"deposit_fee":      payload.DepositFee,
		"water_base":       payload.WaterBase,
		"electric_base":    payload.ElectricBase,
		"gas_base":         payload.GasBase,
		"trash_fee":        payload.TrashFee,
		"water_supply_fee": payload.WaterSupplyFee,
		"sewage_fee":       payload.SewageFee,
		"inventory_note":   payload.InventoryNote,
		"status":           payload.Status,
		"notes":            payload.Notes,
		"company_name":     payload.CompanyName,
	}
	costMode := strings.TrimSpace(payload.CostBearingMode)
	if strings.ToLower(costMode) != "personal" {
		costMode = "company"
	}
	updates["cost_bearing_mode"] = costMode
	if payload.ChargeRates != nil {
		updates["charge_rates"] = datatypes.JSON(payload.ChargeRates)
	}
	if payload.ChargeRates != nil {
		updates["charge_rates"] = datatypes.JSON(payload.ChargeRates)
	}
	if payload.SiteID != 0 {
		var site models.DormSite
		if err := h.db.Where("id = ? AND user_id = ?", payload.SiteID, userID).First(&site).Error; err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusBadRequest
			}
			respondError(w, status, "site not found", err)
			return
		}
		updates["site_id"] = payload.SiteID
	}
	if payload.BuildingID != 0 {
		var building models.DormBuilding
		if err := h.db.Where("id = ? AND user_id = ?", payload.BuildingID, userID).First(&building).Error; err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusBadRequest
			}
			respondError(w, status, "building not found", err)
			return
		}
		updates["building_id"] = payload.BuildingID
	}
	if err := h.db.Model(&room).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update room", err)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", roomID, userID).Preload("Beds").Preload("Assets").First(&room).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load updated room", err)
		return
	}
	respondJSON(w, http.StatusOK, room)
}

func (h *Handler) deleteDormRoom(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	roomID, err := parseUintParam(r, "roomID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid room id", err)
		return
	}
	var room models.DormRoom
	if err := h.db.Where("id = ? AND user_id = ?", roomID, userID).First(&room).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "room not found", err)
		return
	}

	var contractCount int64
	if err := h.db.Model(&models.DormContract{}).Where("user_id = ? AND room_id = ?", userID, roomID).Count(&contractCount).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to inspect room contracts", err)
		return
	}
	if contractCount > 0 {
		respondError(w, http.StatusConflict, "房间仍有关联的入住记录，请先办理退宿或删除对应记录", errors.New("room has active contracts"))
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete room", tx.Error)
		return
	}

	cleanup := func(innerErr error) {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "failed to delete room", innerErr)
	}

	if err := tx.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&models.DormBed{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&models.DormRoomAsset{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&models.DormMeterReading{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&models.DormBillingRule{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&models.DormBill{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("id = ? AND user_id = ?", roomID, userID).Delete(&models.DormRoom{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete room", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"deleted": roomID})
}

// endregion

// region: Beds & Assets

type dormBedPayload struct {
	RoomID    uint   `json:"room_id"`
	BedNumber string `json:"bed_number"`
	Status    string `json:"status"`
}

func (h *Handler) listDormBeds(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID).Model(&models.DormBed{})
	if v := strings.TrimSpace(r.URL.Query().Get("room_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Where("room_id = ?", id)
		}
	}
	var beds []models.DormBed
	if err := query.Order("id DESC").Find(&beds).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load beds", err)
		return
	}
	respondJSON(w, http.StatusOK, beds)
}

func (h *Handler) createDormBed(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload dormBedPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if payload.RoomID == 0 {
		respondError(w, http.StatusBadRequest, "room_id is required", nil)
		return
	}
	var room models.DormRoom
	if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusBadRequest
		}
		respondError(w, status, "room not found", err)
		return
	}
	bed := models.DormBed{
		UserID:    uintPointer(userID),
		RoomID:    payload.RoomID,
		BedNumber: payload.BedNumber,
		Status:    payload.Status,
	}
	if err := h.db.Create(&bed).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create bed", err)
		return
	}
	respondJSON(w, http.StatusCreated, bed)
}

func (h *Handler) updateDormBed(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	bedID, err := parseUintParam(r, "bedID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid bed id", err)
		return
	}
	var bed models.DormBed
	if err := h.db.Where("id = ? AND user_id = ?", bedID, userID).First(&bed).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "bed not found", err)
		return
	}
	var payload dormBedPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"bed_number": payload.BedNumber,
		"status":     payload.Status,
	}
	if payload.RoomID != 0 {
		var room models.DormRoom
		if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusBadRequest
			}
			respondError(w, status, "room not found", err)
			return
		}
		updates["room_id"] = payload.RoomID
	}
	if err := h.db.Model(&bed).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update bed", err)
		return
	}
	respondJSON(w, http.StatusOK, bed)
}

func (h *Handler) deleteDormBed(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	bedID, err := parseUintParam(r, "bedID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid bed id", err)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", bedID, userID).Delete(&models.DormBed{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete bed", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": bedID})
}

type dormAssetPayload struct {
	RoomID      uint       `json:"room_id"`
	AssetType   string     `json:"asset_type"`
	Identifier  string     `json:"identifier"`
	Status      string     `json:"status"`
	PurchasedAt *time.Time `json:"purchased_at"`
	Warranty    *time.Time `json:"warranty_until"`
	Notes       string     `json:"notes"`
}

func (h *Handler) listDormAssets(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID).Model(&models.DormRoomAsset{})
	if v := strings.TrimSpace(r.URL.Query().Get("room_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Where("room_id = ?", id)
		}
	}
	var assets []models.DormRoomAsset
	if err := query.Order("id DESC").Find(&assets).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load assets", err)
		return
	}
	respondJSON(w, http.StatusOK, assets)
}

func (h *Handler) createDormAsset(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload dormAssetPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if payload.RoomID == 0 {
		respondError(w, http.StatusBadRequest, "room_id is required", nil)
		return
	}
	var room models.DormRoom
	if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusBadRequest
		}
		respondError(w, status, "room not found", err)
		return
	}
	asset := models.DormRoomAsset{
		UserID:        uintPointer(userID),
		RoomID:        payload.RoomID,
		AssetType:     payload.AssetType,
		Identifier:    payload.Identifier,
		Status:        payload.Status,
		PurchasedAt:   payload.PurchasedAt,
		WarrantyUntil: payload.Warranty,
		Notes:         payload.Notes,
	}
	if err := h.db.Create(&asset).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create asset", err)
		return
	}
	respondJSON(w, http.StatusCreated, asset)
}

func (h *Handler) updateDormAsset(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	assetID, err := parseUintParam(r, "assetID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id", err)
		return
	}
	var asset models.DormRoomAsset
	if err := h.db.Where("id = ? AND user_id = ?", assetID, userID).First(&asset).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "asset not found", err)
		return
	}
	var payload dormAssetPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"asset_type":     payload.AssetType,
		"identifier":     payload.Identifier,
		"status":         payload.Status,
		"notes":          payload.Notes,
		"purchased_at":   payload.PurchasedAt,
		"warranty_until": payload.Warranty,
	}
	if payload.RoomID != 0 {
		var room models.DormRoom
		if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusBadRequest
			}
			respondError(w, status, "room not found", err)
			return
		}
		updates["room_id"] = payload.RoomID
	}
	if err := h.db.Model(&asset).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update asset", err)
		return
	}
	respondJSON(w, http.StatusOK, asset)
}

func (h *Handler) deleteDormAsset(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	assetID, err := parseUintParam(r, "assetID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id", err)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", assetID, userID).Delete(&models.DormRoomAsset{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete asset", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": assetID})
}

// endregion

// region: Contracts & Checkouts

type dormContractPayload struct {
	EmployeeID    *uint    `json:"employee_id"`
	EmployeeName  string   `json:"employee_name"`
	EmployeeDept  string   `json:"employee_department"`
	EmployeePhone string   `json:"employee_phone"`
	EmployeeIDNo  string   `json:"employee_id_number"`
	ResidenceAddr string   `json:"employee_residence"`
	RoomID        uint     `json:"room_id"`
	BedID         *uint    `json:"bed_id"`
	StartDate     string   `json:"start_date"`
	EndDate       string   `json:"end_date"`
	RentAmount    float64  `json:"rent_amount"`
	DepositAmount float64  `json:"deposit_amount"`
	PaymentMethod string   `json:"payment_method"`
	Attachments   []string `json:"attachments"`
	Status        string   `json:"status"`
	Notes         string   `json:"notes"`
}

func (h *Handler) listDormContracts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID).Model(&models.DormContract{})
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		query = query.Where("status = ?", v)
	}
	if v := strings.TrimSpace(r.URL.Query().Get("room_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Where("room_id = ?", id)
		}
	}
	var contracts []models.DormContract
	if err := query.Preload("Room").Preload("Bed").Order("id DESC").Find(&contracts).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load contracts", err)
		return
	}
	respondJSON(w, http.StatusOK, contracts)
}

func parseISODate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, errors.New("date required")
	}
	return time.Parse("2006-01-02", value)
}

func (h *Handler) createDormContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload dormContractPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if payload.RoomID == 0 {
		respondError(w, http.StatusBadRequest, "room_id is required", nil)
		return
	}
	startDate, err := parseISODate(payload.StartDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid start_date", err)
		return
	}
	endDate, err := parseISODate(payload.EndDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid end_date", err)
		return
	}
	var room models.DormRoom
	if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusBadRequest
		}
		respondError(w, status, "room not found", err)
		return
	}
	if payload.BedID != nil && *payload.BedID != 0 {
		var bed models.DormBed
		if err := h.db.Where("id = ? AND user_id = ? AND room_id = ?", *payload.BedID, userID, room.ID).First(&bed).Error; err != nil {
			respondError(w, http.StatusBadRequest, "bed not found in room", err)
			return
		}
	}
	contract := models.DormContract{
		UserID:        uintPointer(userID),
		RoomID:        payload.RoomID,
		BedID:         payload.BedID,
		StartDate:     startDate,
		EndDate:       endDate,
		RentAmount:    payload.RentAmount,
		DepositAmount: payload.DepositAmount,
		PaymentMethod: payload.PaymentMethod,
		Status:        defaultString(payload.Status, "active"),
		Notes:         payload.Notes,
		Attachments:   datatypes.NewJSONSlice(payload.Attachments),
		EmployeeIDNo:  strings.TrimSpace(payload.EmployeeIDNo),
		ResidenceAddr: strings.TrimSpace(payload.ResidenceAddr),
	}
	if payload.EmployeeID != nil && *payload.EmployeeID != 0 {
		var emp models.Employee
		if err := h.db.Where("id = ?", payload.EmployeeID).First(&emp).Error; err == nil {
			contract.EmployeeID = payload.EmployeeID
			contract.EmployeeName = emp.Name
			contract.EmployeeDept = emp.Department
			contract.EmployeePhone = emp.Phone
		}
	}
	if payload.EmployeeName != "" {
		contract.EmployeeName = payload.EmployeeName
	}
	if payload.EmployeeDept != "" {
		contract.EmployeeDept = payload.EmployeeDept
	}
	if payload.EmployeePhone != "" {
		contract.EmployeePhone = payload.EmployeePhone
	}
	if payload.EmployeeIDNo != "" {
		contract.EmployeeIDNo = payload.EmployeeIDNo
	}
	if payload.ResidenceAddr != "" {
		contract.ResidenceAddr = payload.ResidenceAddr
	}
	if err := h.db.Create(&contract).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create contract", err)
		return
	}
	respondJSON(w, http.StatusCreated, contract)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (h *Handler) updateDormContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contractID, err := parseUintParam(r, "contractID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contract id", err)
		return
	}
	var contract models.DormContract
	if err := h.db.Where("id = ? AND user_id = ?", contractID, userID).First(&contract).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "contract not found", err)
		return
	}
	var payload dormContractPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"employee_name":      payload.EmployeeName,
		"employee_dept":      payload.EmployeeDept,
		"employee_phone":     payload.EmployeePhone,
		"rent_amount":        payload.RentAmount,
		"deposit_amount":     payload.DepositAmount,
		"payment_method":     payload.PaymentMethod,
		"status":             payload.Status,
		"notes":              payload.Notes,
		"employee_id_number": strings.TrimSpace(payload.EmployeeIDNo),
		"employee_residence": strings.TrimSpace(payload.ResidenceAddr),
		"attachments":        datatypes.NewJSONSlice(payload.Attachments),
	}
	if payload.StartDate != "" {
		if start, err := parseISODate(payload.StartDate); err == nil {
			updates["start_date"] = start
		}
	}
	if payload.EndDate != "" {
		if end, err := parseISODate(payload.EndDate); err == nil {
			updates["end_date"] = end
		}
	}
	if payload.RoomID != 0 {
		var room models.DormRoom
		if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusBadRequest
			}
			respondError(w, status, "room not found", err)
			return
		}
		updates["room_id"] = payload.RoomID
	}
	if payload.BedID != nil {
		if *payload.BedID != 0 {
			var bed models.DormBed
			if err := h.db.Where("id = ? AND user_id = ?", *payload.BedID, userID).First(&bed).Error; err != nil {
				respondError(w, http.StatusBadRequest, "bed not found", err)
				return
			}
		}
		updates["bed_id"] = payload.BedID
	}
	if payload.EmployeeID != nil {
		updates["employee_id"] = payload.EmployeeID
	}
	if payload.Attachments != nil {
		updates["attachments"] = datatypes.NewJSONSlice(payload.Attachments)
	}
	if err := h.db.Model(&contract).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update contract", err)
		return
	}
	respondJSON(w, http.StatusOK, contract)
}

func (h *Handler) deleteDormContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contractID, err := parseUintParam(r, "contractID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contract id", err)
		return
	}
	var contract models.DormContract
	if err := h.db.Where("id = ? AND user_id = ?", contractID, userID).First(&contract).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "contract not found", err)
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete contract", tx.Error)
		return
	}

	cleanup := func(innerErr error) {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "failed to delete contract", innerErr)
	}

	if err := tx.Where("contract_id = ? AND user_id = ?", contractID, userID).Delete(&models.DormCheckout{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("contract_id = ? AND user_id = ?", contractID, userID).Delete(&models.DormBillingRule{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("contract_id = ? AND user_id = ?", contractID, userID).Delete(&models.DormBill{}).Error; err != nil {
		cleanup(err)
		return
	}
	if err := tx.Where("id = ? AND user_id = ?", contractID, userID).Delete(&models.DormContract{}).Error; err != nil {
		cleanup(err)
		return
	}

	if contract.RoomID != 0 {
		var remaining int64
		if err := tx.Model(&models.DormContract{}).
			Where("user_id = ? AND room_id = ? AND status = ?", userID, contract.RoomID, "active").
			Count(&remaining).Error; err == nil && remaining == 0 {
			if err := tx.Model(&models.DormRoom{}).Where("id = ? AND user_id = ?", contract.RoomID, userID).Update("status", "空闲").Error; err != nil {
				cleanup(err)
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		cleanup(err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"deleted": contractID})
}

type dormCheckoutPayload struct {
	CheckoutDate        string   `json:"checkout_date"`
	Inspector           string   `json:"inspector"`
	DamageReport        string   `json:"damage_report"`
	ItemsStatus         string   `json:"items_status"`
	FeeSummary          string   `json:"fee_summary"`
	DepositCollected    float64  `json:"deposit_collected"`
	DepositReturn       float64  `json:"deposit_return"`
	DepositDeduct       float64  `json:"deposit_deduct"`
	GuaranteeCollected  float64  `json:"guarantee_collected"`
	GuaranteeDeduct     float64  `json:"guarantee_deduct"`
	GuaranteeReturn     float64  `json:"guarantee_return"`
	GuaranteeReturnDate string   `json:"guarantee_return_date"`
	Attachments         []string `json:"attachments"`
}

func (h *Handler) createDormCheckout(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contractID, err := parseUintParam(r, "contractID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contract id", err)
		return
	}
	var contract models.DormContract
	if err := h.db.Where("id = ? AND user_id = ?", contractID, userID).First(&contract).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "contract not found", err)
		return
	}
	var payload dormCheckoutPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	checkoutDate, err := parseISODate(defaultString(payload.CheckoutDate, time.Now().Format("2006-01-02")))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid checkout_date", err)
		return
	}
	var guaranteeReturnDate *time.Time
	if strings.TrimSpace(payload.GuaranteeReturnDate) != "" {
		parsed, err := parseISODate(payload.GuaranteeReturnDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid guarantee_return_date", err)
			return
		}
		guaranteeReturnDate = &parsed
	}
	var checkout models.DormCheckout
	if err := h.db.Where("contract_id = ? AND user_id = ?", contract.ID, userID).First(&checkout).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			checkout = models.DormCheckout{
				UserID:     uintPointer(userID),
				ContractID: contract.ID,
			}
		} else {
			respondError(w, http.StatusInternalServerError, "failed to load checkout", err)
			return
		}
	}
	checkout.CheckoutDate = checkoutDate
	checkout.Inspector = payload.Inspector
	checkout.DamageReport = payload.DamageReport
	checkout.ItemsStatus = payload.ItemsStatus
	checkout.FeeSummary = payload.FeeSummary
	checkout.DepositCollected = payload.DepositCollected
	checkout.DepositReturn = payload.DepositReturn
	checkout.DepositDeduct = payload.DepositDeduct
	checkout.GuaranteeCollected = payload.GuaranteeCollected
	checkout.GuaranteeDeduct = payload.GuaranteeDeduct
	checkout.GuaranteeReturn = payload.GuaranteeReturn
	checkout.GuaranteeReturnDate = guaranteeReturnDate
	checkout.Attachments = datatypes.NewJSONSlice(payload.Attachments)
	if err := h.db.Save(&checkout).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to record checkout", err)
		return
	}
	contract.Status = "completed"
	_ = h.db.Model(&contract).Update("status", contract.Status).Error
	respondJSON(w, http.StatusCreated, checkout)
}

func (h *Handler) listDormCheckouts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var records []models.DormCheckout
	if err := h.db.Where("user_id = ?", userID).Preload("Contract").Order("id DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load checkouts", err)
		return
	}
	respondJSON(w, http.StatusOK, records)
}

// endregion

// region: Meter items & readings

type meterItemPayload struct {
	Name        string         `json:"name"`
	Category    string         `json:"category"`
	BillingMode string         `json:"billing_mode"`
	PricingMeta map[string]any `json:"pricing_meta"`
}

func (h *Handler) listDormMeterItems(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var items []models.DormMeterItem
	if err := h.db.Where("user_id = ?", userID).Order("id DESC").Find(&items).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load meter items", err)
		return
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *Handler) createDormMeterItem(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload meterItemPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	item := models.DormMeterItem{
		UserID:      uintPointer(userID),
		Name:        payload.Name,
		Category:    payload.Category,
		BillingMode: payload.BillingMode,
		PricingMeta: datatypes.JSONMap(payload.PricingMeta),
	}
	if err := h.db.Create(&item).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create meter item", err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *Handler) updateDormMeterItem(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	itemID, err := parseUintParam(r, "itemID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item id", err)
		return
	}
	var item models.DormMeterItem
	if err := h.db.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "meter item not found", err)
		return
	}
	var payload meterItemPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"name":         payload.Name,
		"category":     payload.Category,
		"billing_mode": payload.BillingMode,
	}
	if payload.PricingMeta != nil {
		updates["pricing_meta"] = datatypes.JSONMap(payload.PricingMeta)
	}
	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update meter item", err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteDormMeterItem(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	itemID, err := parseUintParam(r, "itemID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item id", err)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", itemID, userID).Delete(&models.DormMeterItem{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete meter item", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": itemID})
}

type meterReadingPayload struct {
	RoomID        uint                  `json:"room_id"`
	MeterDate     string                `json:"meter_date"`
	BillingStart  string                `json:"billing_start"`
	BillingEnd    string                `json:"billing_end"`
	Inspector     string                `json:"inspector"`
	ChargeDetails []ChargeDetailPayload `json:"charge_details"`
	Notes         string                `json:"notes"`
}

type ChargeDetailPayload struct {
	Key       string   `json:"key"`
	Label     string   `json:"label,omitempty"`
	Start     *float64 `json:"start,omitempty"`
	End       *float64 `json:"end,omitempty"`
	Usage     *float64 `json:"usage,omitempty"`
	UnitPrice *float64 `json:"unit_price,omitempty"`
	Amount    *float64 `json:"amount,omitempty"`
}

func (h *Handler) listDormMeterReadings(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID).Model(&models.DormMeterReading{})
	if v := strings.TrimSpace(r.URL.Query().Get("room_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Where("room_id = ?", id)
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("building_id")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			query = query.Joins("JOIN dorm_rooms ON dorm_rooms.id = dorm_meter_readings.room_id").Where("dorm_rooms.building_id = ?", id)
		}
	}
	var readings []models.DormMeterReading
	if err := query.Preload("Room").Order("meter_date DESC").Find(&readings).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load readings", err)
		return
	}
	respondJSON(w, http.StatusOK, readings)
}

func (h *Handler) createDormMeterReading(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload meterReadingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if payload.RoomID == 0 {
		respondError(w, http.StatusBadRequest, "room_id is required", nil)
		return
	}
	var room models.DormRoom
	if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
		respondError(w, http.StatusBadRequest, "room not found", err)
		return
	}
	meterDate, err := parseISODate(defaultString(payload.MeterDate, time.Now().Format("2006-01-02")))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid meter_date", err)
		return
	}
	billingStart, err := parseISODate(defaultString(payload.BillingStart, payload.MeterDate))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid billing_start", err)
		return
	}
	billingEnd, err := parseISODate(defaultString(payload.BillingEnd, payload.MeterDate))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid billing_end", err)
		return
	}
	chargeJSON, _ := json.Marshal(payload.ChargeDetails)
	reading := models.DormMeterReading{
		UserID:        uintPointer(userID),
		RoomID:        payload.RoomID,
		MeterDate:     meterDate,
		BillingStart:  billingStart,
		BillingEnd:    billingEnd,
		Inspector:     defaultString(payload.Inspector, "李永娇"),
		ChargeDetails: datatypes.JSON(chargeJSON),
		Notes:         payload.Notes,
	}
	if err := h.db.Create(&reading).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create reading", err)
		return
	}
	if err := h.db.Preload("Room").First(&reading, reading.ID).Error; err != nil {
		reading.Room = &room
	}
	respondJSON(w, http.StatusCreated, reading)
}

func (h *Handler) updateDormMeterReading(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	readingID, err := parseUintParam(r, "readingID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid reading id", err)
		return
	}
	var payload meterReadingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	var reading models.DormMeterReading
	if err := h.db.Where("id = ? AND user_id = ?", readingID, userID).First(&reading).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "reading not found", err)
		return
	}
	if payload.RoomID != 0 {
		var room models.DormRoom
		if err := h.db.Where("id = ? AND user_id = ?", payload.RoomID, userID).First(&room).Error; err != nil {
			respondError(w, http.StatusBadRequest, "room not found", err)
			return
		}
		reading.RoomID = payload.RoomID
	}
	if payload.MeterDate != "" {
		meterDate, err := parseISODate(payload.MeterDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid meter_date", err)
			return
		}
		reading.MeterDate = meterDate
	}
	if payload.BillingStart != "" {
		billingStart, err := parseISODate(payload.BillingStart)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid billing_start", err)
			return
		}
		reading.BillingStart = billingStart
	}
	if payload.BillingEnd != "" {
		billingEnd, err := parseISODate(payload.BillingEnd)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid billing_end", err)
			return
		}
		reading.BillingEnd = billingEnd
	}
	if payload.Inspector != "" {
		reading.Inspector = payload.Inspector
	}
	if payload.ChargeDetails != nil {
		if raw, err := json.Marshal(payload.ChargeDetails); err == nil {
			reading.ChargeDetails = datatypes.JSON(raw)
		} else {
			respondError(w, http.StatusBadRequest, "invalid charge_details", err)
			return
		}
	}
	if payload.Notes != "" {
		reading.Notes = payload.Notes
	}
	if err := h.db.Save(&reading).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update reading", err)
		return
	}
	if err := h.db.Preload("Room").First(&reading, reading.ID).Error; err != nil {
		respondJSON(w, http.StatusOK, reading)
		return
	}
	respondJSON(w, http.StatusOK, reading)
}

func (h *Handler) deleteDormMeterReading(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	readingID, err := parseUintParam(r, "readingID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid reading id", err)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", readingID, userID).Delete(&models.DormMeterReading{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete reading", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": readingID})
}

// endregion

// region: Billing & Bills

type billItemPayload struct {
	ItemType  string  `json:"item_type"`
	Label     string  `json:"label"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Amount    float64 `json:"amount"`
}

type dormBillPayload struct {
	BillCode     string            `json:"bill_code"`
	RoomID       *uint             `json:"room_id"`
	ContractID   *uint             `json:"contract_id"`
	EmployeeID   *uint             `json:"employee_id"`
	EmployeeName string            `json:"employee_name"`
	PeriodLabel  string            `json:"period_label"`
	DueDate      string            `json:"due_date"`
	Items        []billItemPayload `json:"items"`
	Status       string            `json:"status"`
	Metadata     map[string]any    `json:"metadata"`
}

func (h *Handler) listDormBills(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID).Model(&models.DormBill{})
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		query = query.Where("status = ?", v)
	}
	var bills []models.DormBill
	if err := query.Preload("Items").Order("id DESC").Find(&bills).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load bills", err)
		return
	}
	respondJSON(w, http.StatusOK, bills)
}

func (h *Handler) createDormBill(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload dormBillPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if len(payload.Items) == 0 {
		respondError(w, http.StatusBadRequest, "bill items required", nil)
		return
	}
	dueDate := time.Now()
	if payload.DueDate != "" {
		if d, err := parseISODate(payload.DueDate); err == nil {
			dueDate = d
		}
	}
	bill := models.DormBill{
		UserID:       uintPointer(userID),
		BillCode:     payload.BillCode,
		RoomID:       payload.RoomID,
		ContractID:   payload.ContractID,
		EmployeeID:   payload.EmployeeID,
		EmployeeName: payload.EmployeeName,
		PeriodLabel:  payload.PeriodLabel,
		DueDate:      dueDate,
		Status:       defaultString(payload.Status, "pending"),
		Metadata:     datatypes.JSONMap(payload.Metadata),
	}
	var amountDue float64
	for _, item := range payload.Items {
		amt := item.Amount
		if amt == 0 {
			amt = item.Quantity * item.UnitPrice
		}
		bill.Items = append(bill.Items, models.DormBillItem{
			ItemType:  item.ItemType,
			Label:     item.Label,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Amount:    amt,
		})
		amountDue += amt
	}
	bill.AmountDue = amountDue
	if err := h.db.Create(&bill).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create bill", err)
		return
	}
	respondJSON(w, http.StatusCreated, bill)
}

type billStatusPayload struct {
	Status     string  `json:"status"`
	AmountPaid float64 `json:"amount_paid"`
}

func (h *Handler) updateDormBillStatus(w http.ResponseWriter, r *http.Request) {
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
	var bill models.DormBill
	if err := h.db.Where("id = ? AND user_id = ?", billID, userID).Preload("Items").First(&bill).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "bill not found", err)
		return
	}
	var payload billStatusPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updates := map[string]any{
		"status":      payload.Status,
		"amount_paid": payload.AmountPaid,
	}
	if err := h.db.Model(&bill).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update bill", err)
		return
	}
	respondJSON(w, http.StatusOK, bill)
}

// endregion
