package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"

	"siapp/internal/auth"
	"siapp/internal/models"
)

func (h *Handler) registerPreferenceRoutes(r chi.Router) {
	r.Get("/preferences", h.getUserPreferences)
	r.Put("/preferences", h.updateUserPreferences)
}

type userPreferencesPayload map[string]interface{}

func (h *Handler) getUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var prefs []models.UserPreference
	if err := h.db.Where("user_id = ?", userID).Find(&prefs).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load preferences", err)
		return
	}
	result := make(map[string]interface{}, len(prefs))
	for _, pref := range prefs {
		var parsed interface{}
		if len(pref.Value) > 0 {
			if err := json.Unmarshal(pref.Value, &parsed); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to parse preferences", err)
				return
			}
		}
		result[pref.PrefKey] = parsed
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) updateUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload userPreferencesPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if len(payload) == 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}
	response := make(map[string]interface{}, len(payload))
	for key, value := range payload {
		bytes, err := json.Marshal(value)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid preference payload", err)
			return
		}
		var pref models.UserPreference
		err = h.db.Where("user_id = ? AND pref_key = ?", userID, key).
			Assign(models.UserPreference{Value: datatypes.JSON(bytes)}).
			FirstOrCreate(&pref).Error
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update preferences", err)
			return
		}
		response[key] = value
	}
	respondJSON(w, http.StatusOK, response)
}
