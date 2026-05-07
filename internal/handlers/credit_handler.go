package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var (
	createCreditErrorRules = joinErrorRules(
		errorRules{
			{target: services.ErrInvalidCreditData, status: http.StatusBadRequest, message: "invalid credit data"},
			{target: services.ErrInvalidAmount, status: http.StatusBadRequest, message: "invalid amount"},
		},
		creditPolicyErrorRules,
		accountErrorRules,
		mfaErrorRules,
	)
	getCreditErrorRules = errorRules{{target: services.ErrCreditNotFound, status: http.StatusNotFound, message: "credit not found"}}
)

type CreditHandler struct {
	creditService *services.CreditService
	auditRecorder audit.Recorder
}

func NewCreditHandler(creditService *services.CreditService, auditRecorder audit.Recorder) *CreditHandler {
	return &CreditHandler{
		creditService: creditService,
		auditRecorder: auditRecorder,
	}
}

func (h *CreditHandler) CreateCredit(w http.ResponseWriter, r *http.Request) {
	handleAuthedJSON[dto.CreateCreditRequest](w, r, createCreditErrorRules, "create credit failed",
		func(ctx context.Context, userID int64, request dto.CreateCreditRequest) (int, any, error) {
			response, err := h.creditService.CreateCredit(ctx, userID, request)
			if err != nil {
				h.recordCreditCreate(r, userID, 0, "finance.credit_create.failed", audit.StatusFailed, request)
				return http.StatusCreated, nil, err
			}

			h.recordCreditCreate(r, userID, response.ID, "finance.credit_create.success", audit.StatusSuccess, request)
			return http.StatusCreated, response, nil
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

func (h *CreditHandler) recordCreditCreate(
	r *http.Request,
	userID int64,
	creditID int64,
	action string,
	status string,
	request dto.CreateCreditRequest,
) {
	resourceID := creditID
	resourceType := "credit"
	if creditID == 0 {
		resourceID = request.AccountID
		resourceType = "account"
	}

	recordFinancialAudit(h.auditRecorder, r, userID, action, resourceType, resourceID, status, map[string]any{
		"account_id":       request.AccountID,
		"principal_amount": request.PrincipalAmount,
		"term_months":      request.TermMonths,
	})
}
