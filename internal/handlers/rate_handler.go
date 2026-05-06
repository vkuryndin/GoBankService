package handlers

import (
	"net/http"

	"bank-service/internal/services"
)

type RateHandler struct {
	rateService *services.RateService
}

func NewRateHandler(rateService *services.RateService) *RateHandler {
	return &RateHandler{
		rateService: rateService,
	}
}

func (h *RateHandler) GetKeyRate(w http.ResponseWriter, r *http.Request) {
	response, err := h.rateService.GetKeyRate(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get key rate failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}
