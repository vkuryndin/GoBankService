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

type CardHandler struct {
	cardService *services.CardService
}

func NewCardHandler(cardService *services.CardService) *CardHandler {
	return &CardHandler{
		cardService: cardService,
	}
}

func (h *CardHandler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	var request dto.CreateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.cardService.CreateCard(r.Context(), userID, request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCardData) {
			writeError(w, http.StatusBadRequest, "invalid card data")
			return
		}

		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "create card failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *CardHandler) GetUserCards(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
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
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	response, err := h.cardService.GetCard(r.Context(), userID, cardID)
	if err != nil {
		if errors.Is(err, services.ErrCardNotFound) {
			writeError(w, http.StatusNotFound, "card not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "get card failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CardHandler) PayByCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	cardID, err := parseCardID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	var request dto.CardPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.cardService.PayByCard(r.Context(), userID, cardID, request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidAmount) {
			writeError(w, http.StatusBadRequest, "invalid amount")
			return
		}

		if errors.Is(err, services.ErrInvalidDescription) {
			writeError(w, http.StatusBadRequest, "invalid description")
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

		if errors.Is(err, services.ErrInvalidMFAOperation) {
			writeError(w, http.StatusBadRequest, "invalid mfa operation")
			return
		}

		if errors.Is(err, services.ErrInvalidCVV) {
			writeError(w, http.StatusForbidden, "invalid cvv")
			return
		}

		if errors.Is(err, services.ErrCardNotFound) {
			writeError(w, http.StatusNotFound, "card not found")
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

		writeError(w, http.StatusInternalServerError, "card payment failed")
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
