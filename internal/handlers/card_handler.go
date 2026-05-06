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
	createCardErrors = errorMap{
		services.ErrInvalidCardData: {statusCode: http.StatusBadRequest, message: "invalid card data"},
		services.ErrAccountNotFound: {statusCode: http.StatusNotFound, message: "account not found"},
	}

	getCardErrors = errorMap{
		services.ErrCardNotFound: {statusCode: http.StatusNotFound, message: "card not found"},
	}

	cardPaymentErrors = errorMap{
		services.ErrInvalidAmount:       {statusCode: http.StatusBadRequest, message: "invalid amount"},
		services.ErrInvalidDescription:  {statusCode: http.StatusBadRequest, message: "invalid description"},
		services.ErrMFACodeRequired:     {statusCode: http.StatusBadRequest, message: "mfa code required"},
		services.ErrInvalidMFACode:      {statusCode: http.StatusForbidden, message: "invalid mfa code"},
		services.ErrInvalidMFAOperation: {statusCode: http.StatusBadRequest, message: "invalid mfa operation"},
		services.ErrInvalidCVV:          {statusCode: http.StatusForbidden, message: "invalid cvv"},
		services.ErrCardNotFound:        {statusCode: http.StatusNotFound, message: "card not found"},
		services.ErrAccountNotFound:     {statusCode: http.StatusNotFound, message: "account not found"},
		services.ErrInsufficientFunds:   {statusCode: http.StatusConflict, message: "insufficient funds"},
		services.ErrAccountBlocked:      {statusCode: http.StatusForbidden, message: "account is blocked"},
	}
)

type CardHandler struct {
	cardService *services.CardService
}

func NewCardHandler(cardService *services.CardService) *CardHandler {
	return &CardHandler{
		cardService: cardService,
	}
}

func (h *CardHandler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var request dto.CreateCardRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.cardService.CreateCard(r.Context(), userID, request)
	if err != nil {
		writeMappedError(w, err, createCardErrors, "create card failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *CardHandler) GetUserCards(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	response, err := h.cardService.GetUserCards(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get cards failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CardHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	response, err := h.cardService.GetCard(r.Context(), userID, cardID)
	if err != nil {
		writeMappedError(w, err, getCardErrors, "get card failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CardHandler) PayByCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	var request dto.CardPaymentRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.cardService.PayByCard(r.Context(), userID, cardID, request)
	if err != nil {
		writeMappedError(w, err, cardPaymentErrors, "card payment failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func parseCardID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)

	cardID, err := strconv.ParseInt(vars["cardId"], 10, 64)
	if err != nil || cardID <= 0 {
		return 0, errors.New("invalid card id")
	}

	return cardID, nil
}
