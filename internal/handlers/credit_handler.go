package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/services"

	"github.com/gorilla/mux"
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
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	var request dto.CreateCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.creditService.CreateCredit(r.Context(), userID, request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCreditData) {
			writeError(w, http.StatusBadRequest, "invalid credit data")
			return
		}

		if errors.Is(err, services.ErrInvalidAmount) {
			writeError(w, http.StatusBadRequest, "invalid amount")
			return
		}

		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		if errors.Is(err, services.ErrAccountBlocked) {
			writeError(w, http.StatusForbidden, "account is blocked")
			return
		}

		if errors.Is(err, services.ErrMFACodeRequired) {
			writeError(w, http.StatusBadRequest, "mfa code required")
			return
		}

		if errors.Is(err, services.ErrInvalidMFACode) {
			writeError(w, http.StatusForbidden, "invalid mfa code")
			return
		}

		if errors.Is(err, services.ErrInvalidMFAPurpose) {
			writeError(w, http.StatusBadRequest, "invalid mfa purpose")
			return
		}

		if errors.Is(err, services.ErrInvalidMFAOperation) {
			writeError(w, http.StatusBadRequest, "invalid mfa operation")
			return
		}

		writeError(w, http.StatusInternalServerError, "create credit failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *CreditHandler) GetUserCredits(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
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
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	creditID, err := parseCreditID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	response, err := h.creditService.GetCredit(r.Context(), userID, creditID)
	if err != nil {
		if errors.Is(err, services.ErrCreditNotFound) {
			writeError(w, http.StatusNotFound, "credit not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "get credit failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CreditHandler) GetCreditSchedule(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	creditID, err := parseCreditID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	response, err := h.creditService.GetCreditSchedule(r.Context(), userID, creditID)
	if err != nil {
		if errors.Is(err, services.ErrCreditNotFound) {
			writeError(w, http.StatusNotFound, "credit not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "get credit schedule failed")
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
