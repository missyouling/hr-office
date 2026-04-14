package api

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	"siapp/internal/auth"
)

//go:embed data/social_template_constants.json
var socialTemplateOptionsRaw []byte

var socialTemplateOptions socialTemplateOptionsPayload

func init() {
	if err := json.Unmarshal(socialTemplateOptionsRaw, &socialTemplateOptions); err != nil {
		panic(fmt.Sprintf("failed to parse social template options: %v", err))
	}
}

type optionSet struct {
	Options []string `json:"options"`
	Default string   `json:"default"`
}

type socialTemplateOptionsPayload struct {
	GeneratedAt        string    `json:"generated_at"`
	PersonalIdentity   optionSet `json:"personal_identity"`
	HouseholdType      optionSet `json:"household_type"`
	EducationLevel     optionSet `json:"education_level"`
	SpecialSkill       optionSet `json:"special_skill"`
	SkillLevel         optionSet `json:"skill_level"`
	DecreaseReason     optionSet `json:"decrease_reason"`
	UnemploymentReason optionSet `json:"unemployment_reason"`
	ReductionFlag      optionSet `json:"reduction_flag"`
}

type socialTemplateOptionsResponse struct {
	Options socialTemplateOptionsPayload `json:"options"`
}

func (h *Handler) getSocialInsuranceOptions(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.GetUserIDFromContext(r.Context()); err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	respondJSON(w, http.StatusOK, socialTemplateOptionsResponse{Options: socialTemplateOptions})
}
