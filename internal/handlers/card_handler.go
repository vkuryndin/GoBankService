package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/services"

	"github.com/gorilla/mux"
)

var (
	createCardErrorRules = errorRules{
		{target: services.ErrInvalidCardData, statusCode: http.StatusBadRequest, message: "invalid card data"},
		{target: services.ErrAccountNotFound, statusCode: http.StatusNotFound, message: "account not found"},
	}

	getCardErrorRules = errorRules{{target: services.ErrCardNotFound, statusCode: http.StatusNotFound, message: "card not found"}}

	cardCloseErrorRules = errorRules{
		{target: services.ErrInvalidCardData, statusCode: http.StatusBadRequest, message: "invalid card data"},
		{target: services.ErrCardNotFound, statusCode: http.StatusNotFound, message: "card not found"},
		{target: services.ErrCardAlreadyClosed, statusCode: http.StatusConflict, message: "card already closed"},
	}

	cardPaymentErrorRules = joinErrorRules(
		errorRules{
			{target: services.ErrInvalidAmount, statusCode: http.StatusBadRequest, message: "invalid amount"},
			{target: services.ErrInvalidDescription, statusCode: http.StatusBadRequest, message: "invalid description"},
			{target: services.ErrInvalidCVV, statusCode: http.StatusForbidden, message: "invalid cvv"},
			{target: services.ErrCVVAttemptsBlocked, statusCode: http.StatusForbidden, message: "invalid cvv"},
			{target: services.ErrCardNotFound, statusCode: http.StatusNotFound, message: "card not found"},
			{target: services.ErrCardClosed, statusCode: http.StatusConflict, message: "card is closed"},
		},
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

func parseCardID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	cardID, err := strconv.ParseInt(vars["cardId"], 10, 64)
	if err != nil || cardID <= 0 {
		return 0, errors.New("invalid card id")
	}
	return cardID, nil
}
