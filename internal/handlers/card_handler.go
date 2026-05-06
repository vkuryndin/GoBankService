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
	createCardErrorRules = errorRules{
		{target: services.ErrInvalidCardData, statusCode: http.StatusBadRequest, message: "invalid card data"},
		{target: services.ErrAccountNotFound, statusCode: http.StatusNotFound, message: "account not found"},
	}

	getCardErrorRules = errorRules{{target: services.ErrCardNotFound, statusCode: http.StatusNotFound, message: "card not found"}}

	cardPaymentErrorRules = joinErrorRules(
		errorRules{
			{target: services.ErrInvalidAmount, statusCode: http.StatusBadRequest, message: "invalid amount"},
			{target: services.ErrInvalidDescription, statusCode: http.StatusBadRequest, message: "invalid description"},
			{target: services.ErrInvalidCVV, statusCode: http.StatusForbidden, message: "invalid cvv"},
			{target: services.ErrCardNotFound, statusCode: http.StatusNotFound, message: "card not found"},
		},
		mfaErrorRules,
		accountErrorRules,
	)
)

type CardHandler struct{ cardService *services.CardService }

func NewCardHandler(cardService *services.CardService) *CardHandler {
	return &CardHandler{cardService: cardService}
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

func (h *CardHandler) PayByCard(w http.ResponseWriter, r *http.Request) {
	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	handleAuthedJSON[dto.CardPaymentRequest](w, r, cardPaymentErrorRules, "card payment failed",
		func(ctx context.Context, userID int64, request dto.CardPaymentRequest) (int, any, error) {
			response, err := h.cardService.PayByCard(ctx, userID, cardID, request)
			return http.StatusOK, response, err
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
