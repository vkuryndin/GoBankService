package handlers

import (
	"context"
	"errors"
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var (
	createCardErrorRules = joinErrorRules(
		errorRules{{target: services.ErrInvalidCardData, status: http.StatusBadRequest, message: "invalid card data"}},
		accountErrorRules,
	)
	getCardErrorRules   = cardErrorRules
	cardCloseErrorRules = cardErrorRules

	cardPaymentErrorRules = joinErrorRules(
		errorRules{
			{target: services.ErrInvalidAmount, status: http.StatusBadRequest, message: "invalid amount"},
			{target: services.ErrInvalidDescription, status: http.StatusBadRequest, message: "invalid description"},
			{target: services.ErrInvalidCVV, status: http.StatusForbidden, message: "invalid cvv"},
			{target: services.ErrCVVAttemptsBlocked, status: http.StatusForbidden, message: "invalid cvv"},
		},
		cardErrorRules,
		mfaErrorRules,
		accountErrorRules,
	)

	cardTransferErrorRules = joinErrorRules(
		errorRules{
			{target: services.ErrInvalidAmount, status: http.StatusBadRequest, message: "invalid amount"},
			{target: services.ErrInvalidDescription, status: http.StatusBadRequest, message: "invalid description"},
			{target: services.ErrInvalidCVV, status: http.StatusForbidden, message: "invalid cvv"},
			{target: services.ErrCVVAttemptsBlocked, status: http.StatusForbidden, message: "invalid cvv"},
			{target: services.ErrInvalidCardTransfer, status: http.StatusBadRequest, message: "invalid card transfer"},
		},
		cardErrorRules,
		mfaErrorRules,
		accountErrorRules,
	)
)

type CardHandler struct {
	cardService   *services.CardService
	auditRecorder audit.Recorder
}

func NewCardHandler(cardService *services.CardService, auditRecorder audit.Recorder) *CardHandler {
	return &CardHandler{
		cardService:   cardService,
		auditRecorder: auditRecorder,
	}
}

func (h *CardHandler) CreateCard(w http.ResponseWriter, r *http.Request) {
	handleAuthedJSON[dto.CreateCardRequest](w, r, createCardErrorRules, "create card failed",
		func(ctx context.Context, userID int64, request dto.CreateCardRequest) (int, any, error) {
			response, err := h.cardService.CreateCard(ctx, userID, request)
			return http.StatusCreated, response, err
		})
}

func (h *CardHandler) GetUserCards(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "get cards failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.cardService.GetUserCards(ctx, userID)
		return http.StatusOK, response, err
	})
}

func (h *CardHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	handleAuthed(w, r, getCardErrorRules, "get card failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.cardService.GetCard(ctx, userID, cardID)
		return http.StatusOK, response, err
	})
}

func (h *CardHandler) CloseCard(w http.ResponseWriter, r *http.Request) {
	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	handleAuthed(w, r, cardCloseErrorRules, "close card failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.cardService.CloseCard(ctx, userID, cardID)
		if err != nil {
			recordFinancialAudit(h.auditRecorder, r, userID, "card.close.failed", "card", cardID, audit.StatusFailed, nil)
			return http.StatusOK, nil, err
		}

		recordFinancialAudit(h.auditRecorder, r, userID, "card.close.success", "card", cardID, audit.StatusSuccess, map[string]any{
			"account_id": response.AccountID,
		})
		return http.StatusOK, response, nil
	})
}

func (h *CardHandler) PayByCard(w http.ResponseWriter, r *http.Request) {
	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	handleAuthedJSON[dto.CardPaymentRequest](w, r, cardPaymentErrorRules, "card payment failed",
		func(ctx context.Context, userID int64, request dto.CardPaymentRequest) (int, any, error) {
			response, err := h.cardService.PayByCard(ctx, userID, cardID, request)
			if err != nil {
				h.recordCardFailure(r, userID, cardID, request, err)
				return http.StatusOK, nil, err
			}

			recordFinancialAudit(h.auditRecorder, r, userID, "finance.card_payment.success", "transaction", response.TransactionID, audit.StatusSuccess, map[string]any{
				"card_id":    cardID,
				"account_id": response.AccountID,
				"amount":     request.Amount,
			})
			return http.StatusOK, response, nil
		})
}

func (h *CardHandler) TransferByCard(w http.ResponseWriter, r *http.Request) {
	fromCardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	handleAuthedJSON[dto.CardTransferRequest](w, r, cardTransferErrorRules, "card transfer failed",
		func(ctx context.Context, userID int64, request dto.CardTransferRequest) (int, any, error) {
			response, err := h.cardService.TransferByCard(ctx, userID, fromCardID, request)
			if err != nil {
				h.recordCardTransferFailure(r, userID, fromCardID, request, err)
				return http.StatusOK, nil, err
			}

			recordFinancialAudit(h.auditRecorder, r, userID, "finance.card_transfer.success", "transaction", response.TransactionID, audit.StatusSuccess, map[string]any{
				"from_card_id":    fromCardID,
				"to_card_id":      request.ToCardID,
				"from_account_id": response.FromAccountID,
				"to_account_id":   response.ToAccountID,
				"amount":          request.Amount,
			})
			return http.StatusOK, response, nil
		})
}

func (h *CardHandler) recordCardFailure(
	r *http.Request,
	userID int64,
	cardID int64,
	request dto.CardPaymentRequest,
	err error,
) {
	action := "finance.card_payment.failed"
	status := audit.StatusFailed
	if errors.Is(err, services.ErrCVVAttemptsBlocked) {
		action = "card.cvv.blocked"
		status = audit.StatusBlocked
	} else if errors.Is(err, services.ErrInvalidCVV) {
		action = "card.cvv.failed"
	}

	recordFinancialAudit(h.auditRecorder, r, userID, action, "card", cardID, status, map[string]any{
		"amount": request.Amount,
	})
}

func (h *CardHandler) recordCardTransferFailure(
	r *http.Request,
	userID int64,
	fromCardID int64,
	request dto.CardTransferRequest,
	err error,
) {
	action := "finance.card_transfer.failed"
	status := audit.StatusFailed
	if errors.Is(err, services.ErrCVVAttemptsBlocked) {
		action = "card.cvv.blocked"
		status = audit.StatusBlocked
	} else if errors.Is(err, services.ErrInvalidCVV) {
		action = "card.cvv.failed"
	}

	recordFinancialAudit(h.auditRecorder, r, userID, action, "card", fromCardID, status, map[string]any{
		"from_card_id": fromCardID,
		"to_card_id":   request.ToCardID,
		"amount":       request.Amount,
	})
}
