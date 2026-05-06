package handlers

import (
	"net/http"
	"strconv"

	"bank-service/internal/services"
)

var predictionErrors = errorMap{
	services.ErrInvalidPredictionDays: {statusCode: http.StatusBadRequest, message: "days must be between 1 and 365"},
	services.ErrAccountNotFound:       {statusCode: http.StatusNotFound, message: "account not found"},
}

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
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
	userID, ok := requireUserID(w, r)
	if !ok {
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
		writeMappedError(w, err, predictionErrors, "predict balance failed")
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
