package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var (
	getAccountErrorRules = errorRules{{target: services.ErrAccountNotFound, status: http.StatusNotFound, message: "account not found"}}
	depositErrorRules    = joinErrorRules(errorRules{{target: services.ErrInvalidAmount, status: http.StatusBadRequest, message: "invalid amount"}}, accountErrorRules)
	withdrawErrorRules   = joinErrorRules(errorRules{{target: services.ErrInvalidAmount, status: http.StatusBadRequest, message: "invalid amount"}}, accountErrorRules, mfaErrorRules)
)

type AccountHandler struct {
	accountService *services.AccountService
	auditRecorder  audit.Recorder
}

func NewAccountHandler(accountService *services.AccountService, auditRecorder audit.Recorder) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		auditRecorder:  auditRecorder,
	}
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
			if err != nil {
				h.recordAccountOperation(r, userID, accountID, "finance.deposit.failed", audit.StatusFailed, request.Amount)
				return http.StatusOK, nil, err
			}

			h.recordAccountOperation(r, userID, accountID, "finance.deposit.success", audit.StatusSuccess, request.Amount)
			return http.StatusOK, response, nil
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
			if err != nil {
				h.recordAccountOperation(r, userID, accountID, "finance.withdraw.failed", audit.StatusFailed, request.Amount)
				return http.StatusOK, nil, err
			}

			h.recordAccountOperation(r, userID, accountID, "finance.withdraw.success", audit.StatusSuccess, request.Amount)
			return http.StatusOK, response, nil
		})
}

func (h *AccountHandler) recordAccountOperation(
	r *http.Request,
	userID int64,
	accountID int64,
	action string,
	status string,
	amount string,
) {
	recordFinancialAudit(h.auditRecorder, r, userID, action, "account", accountID, status, map[string]any{
		"amount": amount,
	})
}
