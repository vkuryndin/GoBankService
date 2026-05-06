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

type AccountHandler struct {
	accountService *services.AccountService
}

func NewAccountHandler(accountService *services.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	response, err := h.accountService.CreateAccount(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create account failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *AccountHandler) GetUserAccounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	response, err := h.accountService.GetUserAccounts(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get accounts failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
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

	response, err := h.accountService.GetAccount(r.Context(), userID, accountID)
	if err != nil {
		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "get account failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AccountHandler) Deposit(w http.ResponseWriter, r *http.Request) {
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

	var request dto.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.accountService.Deposit(r.Context(), userID, accountID, request)
	if err != nil {
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

		writeError(w, http.StatusInternalServerError, "deposit failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AccountHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
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

	var request dto.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.accountService.Withdraw(r.Context(), userID, accountID, request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidAmount) {
			writeError(w, http.StatusBadRequest, "invalid amount")
			return
		}

		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		if errors.Is(err, services.ErrInsufficientFunds) {
			writeError(w, http.StatusConflict, "insufficient funds")
			return
		}
		if errors.Is(err, services.ErrAccountBlocked) {
			writeError(w, http.StatusForbidden, "account is blocked")
			return
		}

		writeError(w, http.StatusInternalServerError, "withdraw failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func parseAccountID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)

	accountID, err := strconv.ParseInt(vars["accountId"], 10, 64)
	if err != nil || accountID <= 0 {
		return 0, errors.New("invalid account id")
	}

	return accountID, nil
}
