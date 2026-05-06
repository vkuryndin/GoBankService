package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"bank-service/internal/middleware"
	"bank-service/internal/services"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	response, err := h.analyticsService.GetAnalytics(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get analytics failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AnalyticsHandler) PredictBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	days, err := parsePredictionDays(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid days")
		return
	}

	response, err := h.analyticsService.PredictBalance(r.Context(), userID, accountID, days)
	if err != nil {
		if errors.Is(err, services.ErrInvalidPredictionDays) {
			writeError(w, http.StatusBadRequest, "days must be between 1 and 365")
			return
		}

		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "predict balance failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func parsePredictionDays(r *http.Request) (int, error) {
	daysRaw := r.URL.Query().Get("days")
	if daysRaw == "" {
		return 30, nil
	}

	days, err := strconv.Atoi(daysRaw)
	if err != nil {
		return 0, err
	}

	return days, nil
}
