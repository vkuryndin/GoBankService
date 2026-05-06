package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"bank-service/internal/dto"
	"bank-service/internal/services"

	"github.com/gorilla/mux"
)

var (
	createCreditErrors = errorMap{
		services.ErrInvalidCreditData:   {statusCode: http.StatusBadRequest, message: "invalid credit data"},
		services.ErrInvalidAmount:       {statusCode: http.StatusBadRequest, message: "invalid amount"},
		services.ErrAccountNotFound:     {statusCode: http.StatusNotFound, message: "account not found"},
		services.ErrAccountBlocked:      {statusCode: http.StatusForbidden, message: "account is blocked"},
		services.ErrMFACodeRequired:     {statusCode: http.StatusBadRequest, message: "mfa code required"},
		services.ErrInvalidMFACode:      {statusCode: http.StatusForbidden, message: "invalid mfa code"},
		services.ErrInvalidMFAPurpose:   {statusCode: http.StatusBadRequest, message: "invalid mfa purpose"},
		services.ErrInvalidMFAOperation: {statusCode: http.StatusBadRequest, message: "invalid mfa operation"},
	}

	getCreditErrors = errorMap{
		services.ErrCreditNotFound: {statusCode: http.StatusNotFound, message: "credit not found"},
	}
)

type CreditHandler struct {
	creditService *services.CreditService
}

func NewCreditHandler(creditService *services.CreditService) *CreditHandler {
	return &CreditHandler{
		creditService: creditService,
	}
}

func (h *CreditHandler) CreateCredit(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var request dto.CreateCreditRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.creditService.CreateCredit(r.Context(), userID, request)
	if err != nil {
		writeMappedError(w, err, createCreditErrors, "create credit failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *CreditHandler) GetUserCredits(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	response, err := h.creditService.GetUserCredits(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get credits failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CreditHandler) GetCredit(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	creditID, err := parseCreditID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	response, err := h.creditService.GetCredit(r.Context(), userID, creditID)
	if err != nil {
		writeMappedError(w, err, getCreditErrors, "get credit failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CreditHandler) GetCreditSchedule(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	creditID, err := parseCreditID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	response, err := h.creditService.GetCreditSchedule(r.Context(), userID, creditID)
	if err != nil {
		writeMappedError(w, err, getCreditErrors, "get credit schedule failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func parseCreditID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)

	creditID, err := strconv.ParseInt(vars["creditId"], 10, 64)
	if err != nil || creditID <= 0 {
		return 0, errors.New("invalid credit id")
	}

	return creditID, nil
}
