package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"bank-service/internal/dto"
	"bank-service/internal/services"

	"github.com/gorilla/mux"
)

var (
	createCreditErrorRules = joinErrorRules(
		errorRules{
			{target: services.ErrInvalidCreditData, statusCode: http.StatusBadRequest, message: "invalid credit data"},
			{target: services.ErrInvalidAmount, statusCode: http.StatusBadRequest, message: "invalid amount"},
		},
		accountErrorRules,
		mfaErrorRules,
	)
	getCreditErrorRules = errorRules{{target: services.ErrCreditNotFound, statusCode: http.StatusNotFound, message: "credit not found"}}
)

type CreditHandler struct{ creditService *services.CreditService }

func NewCreditHandler(creditService *services.CreditService) *CreditHandler {
	return &CreditHandler{creditService: creditService}
}

func (h *CreditHandler) CreateCredit(w http.ResponseWriter, r *http.Request) {
	handleAuthedJSON[dto.CreateCreditRequest](w, r, createCreditErrorRules, "create credit failed",
		func(ctx context.Context, userID int64, request dto.CreateCreditRequest) (int, any, error) {
			response, err := h.creditService.CreateCredit(ctx, userID, request)
			return http.StatusCreated, response, err
		})
}

func (h *CreditHandler) GetUserCredits(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "get credits failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.creditService.GetUserCredits(ctx, userID)
		return http.StatusOK, response, err
	})
}

func (h *CreditHandler) GetCredit(w http.ResponseWriter, r *http.Request) {
	creditID, err := parseCreditID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	handleAuthed(w, r, getCreditErrorRules, "get credit failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.creditService.GetCredit(ctx, userID, creditID)
		return http.StatusOK, response, err
	})
}

func (h *CreditHandler) GetCreditSchedule(w http.ResponseWriter, r *http.Request) {
	creditID, err := parseCreditID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	handleAuthed(w, r, getCreditErrorRules, "get credit schedule failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.creditService.GetCreditSchedule(ctx, userID, creditID)
		return http.StatusOK, response, err
	})
}

func parseCreditID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	creditID, err := strconv.ParseInt(vars["creditId"], 10, 64)
	if err != nil || creditID <= 0 {
		return 0, errors.New("invalid credit id")
	}
	return creditID, nil
}
