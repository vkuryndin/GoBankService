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
	getAccountErrors = errorMap{
		services.ErrAccountNotFound: {statusCode: http.StatusNotFound, message: "account not found"},
	}

	depositErrors = errorMap{
		services.ErrInvalidAmount:   {statusCode: http.StatusBadRequest, message: "invalid amount"},
		services.ErrAccountNotFound: {statusCode: http.StatusNotFound, message: "account not found"},
		services.ErrAccountBlocked:  {statusCode: http.StatusForbidden, message: "account is blocked"},
	}

	withdrawErrors = errorMap{
		services.ErrInvalidAmount:     {statusCode: http.StatusBadRequest, message: "invalid amount"},
		services.ErrAccountNotFound:   {statusCode: http.StatusNotFound, message: "account not found"},
		services.ErrInsufficientFunds: {statusCode: http.StatusConflict, message: "insufficient funds"},
		services.ErrAccountBlocked:    {statusCode: http.StatusForbidden, message: "account is blocked"},
	}
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
	userID, ok := requireUserID(w, r)
	if !ok {
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
	userID, ok := requireUserID(w, r)
	if !ok {
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
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	response, err := h.accountService.GetAccount(r.Context(), userID, accountID)
	if err != nil {
		writeMappedError(w, err, getAccountErrors, "get account failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AccountHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var request dto.DepositRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.accountService.Deposit(r.Context(), userID, accountID, request)
	if err != nil {
		writeMappedError(w, err, depositErrors, "deposit failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AccountHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var request dto.WithdrawRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.accountService.Withdraw(r.Context(), userID, accountID, request)
	if err != nil {
		writeMappedError(w, err, withdrawErrors, "withdraw failed")
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
