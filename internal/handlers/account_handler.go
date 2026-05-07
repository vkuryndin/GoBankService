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
	getAccountErrorRules = errorRules{{target: services.ErrAccountNotFound, statusCode: http.StatusNotFound, message: "account not found"}}
	depositErrorRules    = joinErrorRules(errorRules{{target: services.ErrInvalidAmount, statusCode: http.StatusBadRequest, message: "invalid amount"}}, accountErrorRules)
	withdrawErrorRules   = joinErrorRules(errorRules{{target: services.ErrInvalidAmount, statusCode: http.StatusBadRequest, message: "invalid amount"}}, accountErrorRules, mfaErrorRules)
)

type AccountHandler struct{ accountService *services.AccountService }

func NewAccountHandler(accountService *services.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "create account failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.accountService.CreateAccount(ctx, userID)
		return http.StatusCreated, response, err
	})
}

func (h *AccountHandler) GetUserAccounts(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "get accounts failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.accountService.GetUserAccounts(ctx, userID)
		return http.StatusOK, response, err
	})
}

func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	handleAuthed(w, r, getAccountErrorRules, "get account failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.accountService.GetAccount(ctx, userID, accountID)
		return http.StatusOK, response, err
	})
}

func (h *AccountHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	handleAuthedJSON[dto.DepositRequest](w, r, depositErrorRules, "deposit failed",
		func(ctx context.Context, userID int64, request dto.DepositRequest) (int, any, error) {
			response, err := h.accountService.Deposit(ctx, userID, accountID, request)
			return http.StatusOK, response, err
		})
}

func (h *AccountHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	handleAuthedJSON[dto.WithdrawRequest](w, r, withdrawErrorRules, "withdraw failed",
		func(ctx context.Context, userID int64, request dto.WithdrawRequest) (int, any, error) {
			response, err := h.accountService.Withdraw(ctx, userID, accountID, request)
			return http.StatusOK, response, err
		})
}

func parseAccountID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	accountID, err := strconv.ParseInt(vars["accountId"], 10, 64)
	if err != nil || accountID <= 0 {
		return 0, errors.New("invalid account id")
	}
	return accountID, nil
}
