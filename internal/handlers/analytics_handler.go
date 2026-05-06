package handlers

import (
	"context"
	"net/http"
	"strconv"

	"bank-service/internal/services"
)

var predictionErrorRules = errorRules{
	{target: services.ErrInvalidPredictionDays, statusCode: http.StatusBadRequest, message: "days must be between 1 and 365"},
	{target: services.ErrAccountNotFound, statusCode: http.StatusNotFound, message: "account not found"},
}

type AnalyticsHandler struct{ analyticsService *services.AnalyticsService }

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "get analytics failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.analyticsService.GetAnalytics(ctx, userID)
		return http.StatusOK, response, err
	})
}

func (h *AnalyticsHandler) PredictBalance(w http.ResponseWriter, r *http.Request) {
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

	handleAuthed(w, r, predictionErrorRules, "predict balance failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.analyticsService.PredictBalance(ctx, userID, accountID, days)
		return http.StatusOK, response, err
	})
}

func parsePredictionDays(r *http.Request) (int, error) {
	daysRaw := r.URL.Query().Get("days")
	if daysRaw == "" {
		return 30, nil
	}
	return strconv.Atoi(daysRaw)
}
